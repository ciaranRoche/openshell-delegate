package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a temporary git repo and returns its path.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git command %v failed: %v\n%s", args, err, out)
		}
	}

	return dir
}

func TestFindRepo_InRoot(t *testing.T) {
	repoDir := initGitRepo(t)

	found, err := FindRepo(repoDir)
	if err != nil {
		t.Fatalf("FindRepo() returned error: %v", err)
	}
	if found != repoDir {
		t.Errorf("FindRepo() = %q, want %q", found, repoDir)
	}
}

func TestFindRepo_InSubdir(t *testing.T) {
	repoDir := initGitRepo(t)

	subDir := filepath.Join(repoDir, "src", "pkg")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := FindRepo(subDir)
	if err != nil {
		t.Fatalf("FindRepo() returned error: %v", err)
	}
	if found != repoDir {
		t.Errorf("FindRepo() = %q, want %q", found, repoDir)
	}
}

func TestFindRepo_NotARepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindRepo(tmpDir)
	if err == nil {
		t.Error("FindRepo() should return error for non-git directory")
	}
}

func TestRepoName_NoRemote(t *testing.T) {
	repoDir := initGitRepo(t)

	name := RepoName(repoDir)
	// When there's no remote, it should fall back to directory name
	expected := filepath.Base(repoDir)
	if name != expected {
		t.Errorf("RepoName() = %q, want %q", name, expected)
	}
}

func TestWorktreeDir(t *testing.T) {
	tests := []struct {
		name     string
		repoPath string
		branch   string
		expected string
	}{
		{
			name:     "simple branch",
			repoPath: "/repos/my-project",
			branch:   "feature-x",
			expected: "/repos/my-project/.delegate/feature-x",
		},
		{
			name:     "branch with hyphens",
			repoPath: "/home/user/code/repo",
			branch:   "fix-scaling-issue",
			expected: "/home/user/code/repo/.delegate/fix-scaling-issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorktreeDir(tt.repoPath, tt.branch)
			if got != tt.expected {
				t.Errorf("WorktreeDir(%q, %q) = %q, want %q", tt.repoPath, tt.branch, got, tt.expected)
			}
		})
	}
}

func TestCreate_And_Remove(t *testing.T) {
	repoDir := initGitRepo(t)

	wtDir, err := Create(repoDir, "test-branch")
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	expectedDir := WorktreeDir(repoDir, "test-branch")
	if wtDir != expectedDir {
		t.Errorf("Create() = %q, want %q", wtDir, expectedDir)
	}

	// Verify worktree directory exists
	if _, err := os.Stat(wtDir); os.IsNotExist(err) {
		t.Error("worktree directory should exist after Create")
	}

	// Verify it's a valid git worktree (has .git file)
	gitFile := filepath.Join(wtDir, ".git")
	if _, err := os.Stat(gitFile); os.IsNotExist(err) {
		t.Error("worktree should have .git file/directory")
	}

	// Remove the worktree
	if err := Remove(repoDir, wtDir, "test-branch"); err != nil {
		t.Fatalf("Remove() returned error: %v", err)
	}

	// Verify worktree directory is gone
	if _, err := os.Stat(wtDir); !os.IsNotExist(err) {
		t.Error("worktree directory should be removed after Remove")
	}
}

func TestCreate_AlreadyExists(t *testing.T) {
	repoDir := initGitRepo(t)

	_, err := Create(repoDir, "dup-branch")
	if err != nil {
		t.Fatalf("first Create() returned error: %v", err)
	}

	_, err = Create(repoDir, "dup-branch")
	if err == nil {
		t.Error("second Create() should return error for existing worktree")
	}
}

func TestIsDirty_Clean(t *testing.T) {
	repoDir := initGitRepo(t)

	dirty, err := IsDirty(repoDir)
	if err != nil {
		t.Fatalf("IsDirty() returned error: %v", err)
	}
	if dirty {
		t.Error("freshly initialized repo should not be dirty")
	}
}

func TestIsDirty_WithChanges(t *testing.T) {
	repoDir := initGitRepo(t)

	// Create an untracked file
	if err := os.WriteFile(filepath.Join(repoDir, "new-file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	dirty, err := IsDirty(repoDir)
	if err != nil {
		t.Fatalf("IsDirty() returned error: %v", err)
	}
	if !dirty {
		t.Error("repo with untracked file should be dirty")
	}
}

func TestIsDirty_WithStagedChanges(t *testing.T) {
	repoDir := initGitRepo(t)

	// Create and stage a file
	filePath := filepath.Join(repoDir, "staged.txt")
	if err := os.WriteFile(filePath, []byte("staged"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "add", "staged.txt")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	dirty, err := IsDirty(repoDir)
	if err != nil {
		t.Fatalf("IsDirty() returned error: %v", err)
	}
	if !dirty {
		t.Error("repo with staged changes should be dirty")
	}
}
