package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	return tmpDir
}

func TestState_GetSetRemove(t *testing.T) {
	s := &State{Delegations: make(map[string]Delegation)}

	// Test Get on empty state
	_, ok := s.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent branch")
	}

	// Test Set
	d := Delegation{
		Sandbox:  "od-test",
		Repo:     "/path/to/repo",
		Worktree: "/path/to/repo/.delegate/test",
		Branch:   "test",
		Created:  time.Now(),
	}
	s.Set("test", d)

	got, ok := s.Get("test")
	if !ok {
		t.Fatal("Get should return true after Set")
	}
	if got.Sandbox != "od-test" {
		t.Errorf("Sandbox = %q, want %q", got.Sandbox, "od-test")
	}
	if got.Repo != "/path/to/repo" {
		t.Errorf("Repo = %q, want %q", got.Repo, "/path/to/repo")
	}
	if got.Branch != "test" {
		t.Errorf("Branch = %q, want %q", got.Branch, "test")
	}

	// Test Remove
	s.Remove("test")
	_, ok = s.Get("test")
	if ok {
		t.Error("Get should return false after Remove")
	}
}

func TestState_SetOverwrite(t *testing.T) {
	s := &State{Delegations: make(map[string]Delegation)}

	d1 := Delegation{Sandbox: "sandbox-1", Branch: "test"}
	s.Set("test", d1)

	d2 := Delegation{Sandbox: "sandbox-2", Branch: "test"}
	s.Set("test", d2)

	got, ok := s.Get("test")
	if !ok {
		t.Fatal("Get should return true")
	}
	if got.Sandbox != "sandbox-2" {
		t.Errorf("Sandbox = %q, want %q (should be overwritten)", got.Sandbox, "sandbox-2")
	}
}

func TestState_RemoveNonexistent(t *testing.T) {
	s := &State{Delegations: make(map[string]Delegation)}

	// Should not panic
	s.Remove("nonexistent")

	if len(s.Delegations) != 0 {
		t.Errorf("expected empty delegations, got %d", len(s.Delegations))
	}
}

func TestSaveAndLoad(t *testing.T) {
	setupTestEnv(t)

	original := &State{
		Delegations: map[string]Delegation{
			"feature-a": {
				Sandbox:  "od-feature-a",
				Repo:     "/repos/my-project",
				Worktree: "/repos/my-project/.delegate/feature-a",
				Branch:   "feature-a",
				Created:  time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			"bugfix-b": {
				Sandbox:  "od-bugfix-b",
				Repo:     "/repos/other-project",
				Worktree: "/repos/other-project/.delegate/bugfix-b",
				Branch:   "bugfix-b",
				Created:  time.Date(2025, 2, 20, 14, 0, 0, 0, time.UTC),
			},
		},
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if len(loaded.Delegations) != 2 {
		t.Fatalf("expected 2 delegations, got %d", len(loaded.Delegations))
	}

	d, ok := loaded.Get("feature-a")
	if !ok {
		t.Fatal("expected to find feature-a delegation")
	}
	if d.Sandbox != "od-feature-a" {
		t.Errorf("Sandbox = %q, want %q", d.Sandbox, "od-feature-a")
	}
	if d.Repo != "/repos/my-project" {
		t.Errorf("Repo = %q, want %q", d.Repo, "/repos/my-project")
	}

	d, ok = loaded.Get("bugfix-b")
	if !ok {
		t.Fatal("expected to find bugfix-b delegation")
	}
	if d.Sandbox != "od-bugfix-b" {
		t.Errorf("Sandbox = %q, want %q", d.Sandbox, "od-bugfix-b")
	}
}

func TestLoad_NoFile(t *testing.T) {
	setupTestEnv(t)

	s, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if s == nil {
		t.Fatal("Load() returned nil")
	}
	if s.Delegations == nil {
		t.Fatal("Delegations should be initialized, not nil")
	}
	if len(s.Delegations) != 0 {
		t.Errorf("expected empty delegations, got %d", len(s.Delegations))
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := setupTestEnv(t)

	configDir := filepath.Join(tmpDir, "openshell-delegate")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, stateFile), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error for invalid JSON")
	}
}

func TestLoad_NilDelegationsMap(t *testing.T) {
	tmpDir := setupTestEnv(t)

	configDir := filepath.Join(tmpDir, "openshell-delegate")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// JSON with no delegations field
	if err := os.WriteFile(filepath.Join(configDir, stateFile), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if s.Delegations == nil {
		t.Fatal("Delegations should be initialized even when absent in JSON")
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := setupTestEnv(t)

	s := &State{Delegations: make(map[string]Delegation)}
	if err := Save(s); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	// Check the directory was created
	configDir := filepath.Join(tmpDir, "openshell-delegate")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("expected config directory to be created")
	}

	// Check the file exists
	if _, err := os.Stat(filepath.Join(configDir, stateFile)); os.IsNotExist(err) {
		t.Error("expected state file to be created")
	}
}
