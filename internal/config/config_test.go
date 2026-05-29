package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde with subpath",
			input:    "~/foo/bar",
			expected: filepath.Join(home, "foo/bar"),
		},
		{
			name:     "tilde alone",
			input:    "~",
			expected: filepath.Join(home),
		},
		{
			name:     "tilde with slash",
			input:    "~/",
			expected: filepath.Join(home, "/"),
		},
		{
			name:     "no tilde absolute path",
			input:    "/usr/local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "no tilde relative path",
			input:    "foo/bar",
			expected: "foo/bar",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.input)
			if got != tt.expected {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoad_MissingFile(t *testing.T) {
	// Override XDG_CONFIG_HOME to point to a temp dir with no config
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	// Should return zero-value config
	if cfg.Image != "" {
		t.Errorf("expected empty Image, got %q", cfg.Image)
	}
	if cfg.Policy != "" {
		t.Errorf("expected empty Policy, got %q", cfg.Policy)
	}
	if cfg.Provider != "" {
		t.Errorf("expected empty Provider, got %q", cfg.Provider)
	}
	if len(cfg.Env) != 0 {
		t.Errorf("expected empty Env, got %v", cfg.Env)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, appName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `
image: "ghcr.io/test/image:latest"
policy: "~/policies/test.yaml"
provider: "test-provider"
env:
  - FOO_TOKEN
  - BAR_KEY
static_env:
  MY_VAR: "my_value"
uploads:
  - src: "~/.config/test"
    dst: "/sandbox/.config/test"
`
	if err := os.WriteFile(filepath.Join(configDir, configFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Image != "ghcr.io/test/image:latest" {
		t.Errorf("Image = %q, want %q", cfg.Image, "ghcr.io/test/image:latest")
	}
	if cfg.Policy != "~/policies/test.yaml" {
		t.Errorf("Policy = %q, want %q", cfg.Policy, "~/policies/test.yaml")
	}
	if cfg.Provider != "test-provider" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "test-provider")
	}
	if len(cfg.Env) != 2 {
		t.Fatalf("len(Env) = %d, want 2", len(cfg.Env))
	}
	if cfg.Env[0] != "FOO_TOKEN" || cfg.Env[1] != "BAR_KEY" {
		t.Errorf("Env = %v, want [FOO_TOKEN BAR_KEY]", cfg.Env)
	}
	if v, ok := cfg.StaticEnv["MY_VAR"]; !ok || v != "my_value" {
		t.Errorf("StaticEnv[MY_VAR] = %q, want %q", v, "my_value")
	}
	if len(cfg.Uploads) != 1 {
		t.Fatalf("len(Uploads) = %d, want 1", len(cfg.Uploads))
	}
	if cfg.Uploads[0].Src != "~/.config/test" {
		t.Errorf("Uploads[0].Src = %q, want %q", cfg.Uploads[0].Src, "~/.config/test")
	}
	if cfg.Uploads[0].Dst != "/sandbox/.config/test" {
		t.Errorf("Uploads[0].Dst = %q, want %q", cfg.Uploads[0].Dst, "/sandbox/.config/test")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, appName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(configDir, configFile), []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error for invalid YAML")
	}
}

func TestConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() returned error: %v", err)
	}

	expected := filepath.Join(tmpDir, appName)
	if dir != expected {
		t.Errorf("ConfigDir() = %q, want %q", dir, expected)
	}
}
