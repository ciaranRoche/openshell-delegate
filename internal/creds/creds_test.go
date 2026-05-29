package creds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ciaranRoche/openshell-delegate/internal/config"
)

func TestValidate_AllPresent(t *testing.T) {
	// Create a temp file for the upload source
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "creds.json")
	if err := os.WriteFile(srcFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_TOKEN", "abc123")

	cfg := &config.Config{
		Env: []string{"TEST_TOKEN"},
		Uploads: []config.Upload{
			{Src: srcFile, Dst: "/sandbox/creds.json"},
		},
	}

	errs := Validate(cfg)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_MissingEnvVar(t *testing.T) {
	t.Setenv("EXISTING_VAR", "value")

	cfg := &config.Config{
		Env: []string{"EXISTING_VAR", "MISSING_VAR"},
	}

	// Ensure MISSING_VAR is unset
	os.Unsetenv("MISSING_VAR")

	errs := Validate(cfg)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0], "MISSING_VAR") {
		t.Errorf("expected error about MISSING_VAR, got %q", errs[0])
	}
}

func TestValidate_MissingUploadSource(t *testing.T) {
	cfg := &config.Config{
		Uploads: []config.Upload{
			{Src: "/nonexistent/path/file.json", Dst: "/sandbox/file.json"},
		},
	}

	errs := Validate(cfg)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0], "upload source not found") {
		t.Errorf("expected 'upload source not found' error, got %q", errs[0])
	}
}

func TestValidate_EmptyConfig(t *testing.T) {
	cfg := &config.Config{}

	errs := Validate(cfg)
	if len(errs) != 0 {
		t.Errorf("expected no errors for empty config, got %v", errs)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	os.Unsetenv("MISSING_1")
	os.Unsetenv("MISSING_2")

	cfg := &config.Config{
		Env: []string{"MISSING_1", "MISSING_2"},
		Uploads: []config.Upload{
			{Src: "/nonexistent/file", Dst: "/sandbox/file"},
		},
	}

	errs := Validate(cfg)
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestBuildLaunchScript_WithStaticEnv(t *testing.T) {
	cfg := &config.Config{
		StaticEnv: map[string]string{
			"MY_STATIC": "static_value",
		},
	}

	script := BuildLaunchScript(cfg)

	if !strings.Contains(script, "#!/bin/sh") {
		t.Error("script should start with shebang")
	}
	if !strings.Contains(script, `export MY_STATIC="static_value"`) {
		t.Error("script should contain static env export")
	}
	if !strings.Contains(script, "exec sleep infinity") {
		t.Error("script should end with sleep infinity")
	}
	if !strings.Contains(script, "GOPATH") {
		t.Error("script should set GOPATH")
	}
}

func TestBuildLaunchScript_WithForwardedEnv(t *testing.T) {
	t.Setenv("MY_FORWARDED", "forwarded_value")

	cfg := &config.Config{
		Env: []string{"MY_FORWARDED"},
	}

	script := BuildLaunchScript(cfg)

	if !strings.Contains(script, `export MY_FORWARDED="forwarded_value"`) {
		t.Error("script should contain forwarded env export")
	}
}

func TestBuildLaunchScript_SkipsUnsetForwardedEnv(t *testing.T) {
	os.Unsetenv("UNSET_VAR")

	cfg := &config.Config{
		Env: []string{"UNSET_VAR"},
	}

	script := BuildLaunchScript(cfg)

	if strings.Contains(script, "UNSET_VAR") {
		t.Error("script should not contain unset env var")
	}
}

func TestBuildLaunchScript_GoConfig(t *testing.T) {
	cfg := &config.Config{}

	script := BuildLaunchScript(cfg)

	if !strings.Contains(script, `export PATH="/usr/local/go/bin:/sandbox/go/bin:$PATH"`) {
		t.Error("script should set Go PATH")
	}
	if !strings.Contains(script, `export GOPATH=/sandbox/go`) {
		t.Error("script should set GOPATH")
	}
	if !strings.Contains(script, `export GOBIN=/sandbox/go/bin`) {
		t.Error("script should set GOBIN")
	}
}

func TestUploadMap_Empty(t *testing.T) {
	cfg := &config.Config{}

	uploads := UploadMap(cfg)
	if len(uploads) != 0 {
		t.Errorf("expected empty map, got %v", uploads)
	}
}

func TestUploadMap_WithUploads(t *testing.T) {
	cfg := &config.Config{
		Uploads: []config.Upload{
			{Src: "/host/path/file1", Dst: "/sandbox/file1"},
			{Src: "/host/path/file2", Dst: "/sandbox/file2"},
		},
	}

	uploads := UploadMap(cfg)
	if len(uploads) != 2 {
		t.Fatalf("expected 2 uploads, got %d", len(uploads))
	}

	if uploads["/host/path/file1"] != "/sandbox/file1" {
		t.Errorf("wrong mapping for file1: %q", uploads["/host/path/file1"])
	}
	if uploads["/host/path/file2"] != "/sandbox/file2" {
		t.Errorf("wrong mapping for file2: %q", uploads["/host/path/file2"])
	}
}

func TestUploadMap_ExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home: %v", err)
	}

	cfg := &config.Config{
		Uploads: []config.Upload{
			{Src: "~/.config/test", Dst: "/sandbox/.config/test"},
		},
	}

	uploads := UploadMap(cfg)
	expected := filepath.Join(home, ".config/test")
	if uploads[expected] != "/sandbox/.config/test" {
		t.Errorf("expected key %q, got map: %v", expected, uploads)
	}
}
