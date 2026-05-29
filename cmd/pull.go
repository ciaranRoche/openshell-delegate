package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ciaranRoche/openshell-delegate/internal/sandbox"
	"github.com/ciaranRoche/openshell-delegate/internal/state"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <branch>",
	Short: "Download workspace from sandbox back to worktree",
	Long: `Downloads the workspace from a running sandbox back to the local git worktree.
The worktree is then a regular git checkout you can diff, commit, and push.`,
	Args: cobra.ExactArgs(1),
	RunE: runPull,
}

func runPull(cmd *cobra.Command, args []string) error {
	branch := args[0]

	// Load state
	st, err := state.Load()
	if err != nil {
		return err
	}

	d, ok := st.Get(branch)
	if !ok {
		return fmt.Errorf("no delegation found for branch %q -- check 'openshell-delegate list'", branch)
	}

	// Verify sandbox is running
	if !sandbox.IsReady(d.Sandbox) {
		return fmt.Errorf("sandbox %q is not running -- it may have been deleted", d.Sandbox)
	}

	// Verify worktree exists
	if _, err := os.Stat(d.Worktree); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found at %s", d.Worktree)
	}

	// Download to temp dir
	tmpDir, err := os.MkdirTemp("", "od-pull-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("Pulling: %s:/sandbox/workspace -> %s\n", d.Sandbox, d.Worktree)

	if err := sandbox.Download(d.Sandbox, "/sandbox/workspace", tmpDir); err != nil {
		return fmt.Errorf("could not download workspace: %w", err)
	}

	// Find the downloaded content -- openshell may put it under a subdirectory
	src := filepath.Join(tmpDir, "workspace")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		src = tmpDir
	}

	// Rsync into worktree, excluding .git
	rsyncCmd := exec.Command("rsync", "-a", "--delete", "--exclude=.git", src+"/", d.Worktree+"/")
	rsyncCmd.Stdout = os.Stdout
	rsyncCmd.Stderr = os.Stderr
	if err := rsyncCmd.Run(); err != nil {
		return fmt.Errorf("rsync failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("Pulled to: %s\n", d.Worktree)
	fmt.Println()

	// Show git status
	fmt.Println("Changes:")
	gitCmd := exec.Command("git", "status", "--short")
	gitCmd.Dir = d.Worktree
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	_ = gitCmd.Run()

	fmt.Println()
	fmt.Printf("Review: cd %s && git diff\n", d.Worktree)

	return nil
}
