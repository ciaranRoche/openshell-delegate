package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDelegateGitignore_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	ensureDelegateGitignore(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("expected .gitignore to be created, got error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, ".delegate/") {
		t.Errorf("expected .gitignore to contain '.delegate/', got %q", content)
	}
}

func TestEnsureDelegateGitignore_ExistingWithoutDelegate(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("bin/\n*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ensureDelegateGitignore(tmpDir)

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "bin/") {
		t.Error("original content should be preserved")
	}
	if !strings.Contains(content, ".delegate/") {
		t.Error("should append .delegate/")
	}
}

func TestEnsureDelegateGitignore_AlreadyPresent(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	original := "bin/\n.delegate/\n*.log\n"
	if err := os.WriteFile(gitignorePath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	ensureDelegateGitignore(tmpDir)

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	// Should not duplicate .delegate/
	content := string(data)
	count := strings.Count(content, ".delegate")
	if count != 1 {
		t.Errorf("expected .delegate to appear once, appeared %d times in %q", count, content)
	}
}

func TestEnsureDelegateGitignore_AlreadyPresentWithoutSlash(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	original := "bin/\n.delegate\n*.log\n"
	if err := os.WriteFile(gitignorePath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	ensureDelegateGitignore(tmpDir)

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	// Should not duplicate
	content := string(data)
	count := strings.Count(content, ".delegate")
	if count != 1 {
		t.Errorf("expected .delegate to appear once, appeared %d times in %q", count, content)
	}
}

func TestEnsureDelegateGitignore_NoTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	// File without trailing newline
	if err := os.WriteFile(gitignorePath, []byte("bin/"), 0o644); err != nil {
		t.Fatal(err)
	}

	ensureDelegateGitignore(tmpDir)

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// Should have a newline between existing content and .delegate/
	if !strings.Contains(content, "bin/\n.delegate/") {
		t.Errorf("expected newline before .delegate/, got %q", content)
	}
}

func TestRootCommand_HasSubcommands(t *testing.T) {
	// Verify all expected subcommands are registered
	subcommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		subcommands[cmd.Name()] = true
	}

	expected := []string{"run", "pull", "list", "cleanup"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("expected subcommand %q to be registered", name)
		}
	}
}

func TestRootCommand_VersionSet(t *testing.T) {
	if rootCmd.Version == "" {
		t.Error("rootCmd.Version should not be empty")
	}
}

func TestRunCmd_RequiresBranch(t *testing.T) {
	// The run command should have --branch as required
	flag := runCmd.Flags().Lookup("branch")
	if flag == nil {
		t.Fatal("run command should have --branch flag")
	}
}

func TestRunCmd_HasFlags(t *testing.T) {
	flags := []string{"branch", "task", "task-file", "name", "image", "policy"}
	for _, name := range flags {
		if runCmd.Flags().Lookup(name) == nil {
			t.Errorf("run command should have --%s flag", name)
		}
	}
}

func TestCleanupCmd_RequiresArg(t *testing.T) {
	if cleanupCmd.Args == nil {
		t.Error("cleanup command should have Args validation")
	}
}

func TestCleanupCmd_HasFlags(t *testing.T) {
	flags := []string{"all", "force"}
	for _, name := range flags {
		if cleanupCmd.Flags().Lookup(name) == nil {
			t.Errorf("cleanup command should have --%s flag", name)
		}
	}
}

func TestRunCleanup_NoBranchNoAll(t *testing.T) {
	origAll := flagCleanupAll
	origForce := flagCleanupForce
	defer func() {
		flagCleanupAll = origAll
		flagCleanupForce = origForce
	}()

	flagCleanupAll = false
	flagCleanupForce = false

	err := runCleanup(nil, []string{})
	if err == nil {
		t.Fatal("expected error when no branch and no --all")
	}
	if !strings.Contains(err.Error(), "provide a branch name") {
		t.Errorf("expected 'provide a branch name' error, got %q", err.Error())
	}
}

func TestRunCleanup_AllWithBranch(t *testing.T) {
	origAll := flagCleanupAll
	origForce := flagCleanupForce
	defer func() {
		flagCleanupAll = origAll
		flagCleanupForce = origForce
	}()

	flagCleanupAll = true
	flagCleanupForce = false

	err := runCleanup(nil, []string{"some-branch"})
	if err == nil {
		t.Fatal("expected error when --all used with a branch")
	}
	if !strings.Contains(err.Error(), "cannot use --all") {
		t.Errorf("expected 'cannot use --all' error, got %q", err.Error())
	}
}

func TestCleanupDelegateDir_RemovesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	delegateDir := filepath.Join(tmpDir, ".delegate")
	if err := os.MkdirAll(delegateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cleanupDelegateDir(tmpDir)

	if _, err := os.Stat(delegateDir); !os.IsNotExist(err) {
		t.Error("empty .delegate directory should be removed")
	}
}

func TestCleanupDelegateDir_KeepsNonEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	delegateDir := filepath.Join(tmpDir, ".delegate")
	if err := os.MkdirAll(filepath.Join(delegateDir, "other-branch"), 0o755); err != nil {
		t.Fatal(err)
	}

	cleanupDelegateDir(tmpDir)

	if _, err := os.Stat(delegateDir); os.IsNotExist(err) {
		t.Error("non-empty .delegate directory should NOT be removed")
	}
}

func TestCleanupDelegateDir_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not panic when .delegate doesn't exist
	cleanupDelegateDir(tmpDir)
}

func TestPullCmd_RequiresArg(t *testing.T) {
	if pullCmd.Args == nil {
		t.Error("pull command should have Args validation")
	}
}

func TestListCmd_HasAlias(t *testing.T) {
	if len(listCmd.Aliases) == 0 {
		t.Fatal("list command should have aliases")
	}
	found := false
	for _, alias := range listCmd.Aliases {
		if alias == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("list command should have 'ls' alias, got %v", listCmd.Aliases)
	}
}

func TestSandboxContextConstant(t *testing.T) {
	if !strings.Contains(sandboxContext, "TASK.md") {
		t.Error("sandboxContext should reference TASK.md")
	}
	if !strings.Contains(sandboxContext, "Sandbox Context") {
		t.Error("sandboxContext should have header")
	}
}

func TestRunRun_MutuallyExclusiveFlags(t *testing.T) {
	// Save original values and restore after test
	origTask := flagTask
	origTaskFile := flagTaskFile
	origBranch := flagBranch
	defer func() {
		flagTask = origTask
		flagTaskFile = origTaskFile
		flagBranch = origBranch
	}()

	flagTask = "inline task"
	flagTaskFile = "task.md"
	flagBranch = "test-branch"

	err := runRun(nil, nil)
	if err == nil {
		t.Fatal("expected error when both --task and --task-file are set")
	}
	if !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("expected 'cannot use both' error, got %q", err.Error())
	}
}
