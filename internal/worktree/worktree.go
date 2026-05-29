package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindRepo walks up from the given directory to find a git repository root.
// Returns the absolute path to the repo root.
func FindRepo(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(abs, ".git")
		if info, err := os.Stat(gitDir); err == nil {
			if info.IsDir() {
				return abs, nil
			}
			// .git can be a file in worktrees, pointing to the real .git dir
			return abs, nil
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("not a git repository (or any parent up to /)")
		}
		abs = parent
	}
}

// RepoName returns the repository name from the remote URL or directory name.
func RepoName(repoPath string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return filepath.Base(repoPath)
	}

	url := strings.TrimSpace(string(out))
	// Handle both HTTPS and SSH URLs
	name := filepath.Base(url)
	name = strings.TrimSuffix(name, ".git")
	return name
}

// WorktreeDir returns the path where the worktree should be created.
// Convention: <repo-parent>/worktrees/<branch>
func WorktreeDir(repoPath, branch string) string {
	parent := filepath.Dir(repoPath)
	return filepath.Join(parent, "worktrees", branch)
}

// Create creates a git worktree on a new branch.
func Create(repoPath, branch string) (string, error) {
	wtDir := WorktreeDir(repoPath, branch)

	if _, err := os.Stat(wtDir); err == nil {
		return "", fmt.Errorf("worktree already exists at %s -- use a different branch name or run cleanup first", wtDir)
	}

	cmd := exec.Command("git", "worktree", "add", wtDir, "-b", branch)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("could not create worktree: %w", err)
	}

	return wtDir, nil
}

// Remove removes a git worktree and deletes the branch.
func Remove(repoPath, wtDir, branch string) error {
	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", wtDir)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Try force remove if normal remove fails
		cmd = exec.Command("git", "worktree", "remove", "--force", wtDir)
		cmd.Dir = repoPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}

	// Delete the branch
	cmd = exec.Command("git", "branch", "-D", branch)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // Best effort, don't fail if branch is already gone

	return nil
}

// IsDirty checks if a worktree has uncommitted changes.
func IsDirty(wtDir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = wtDir
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}
