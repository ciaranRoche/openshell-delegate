package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ciaranRoche/openshell-delegate/internal/sandbox"
	"github.com/ciaranRoche/openshell-delegate/internal/state"
	"github.com/ciaranRoche/openshell-delegate/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	flagCleanupAll   bool
	flagCleanupForce bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup <branch>",
	Short: "Remove sandbox and worktree for a delegation",
	Long: `Deletes the OpenShell sandbox, removes the git worktree, and cleans up
the delegation state. Warns if the worktree has uncommitted changes.

Use --all to clean up every active delegation at once.
Use --force to skip the confirmation prompt for dirty worktrees.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().BoolVar(&flagCleanupAll, "all", false, "Clean up all active delegations")
	cleanupCmd.Flags().BoolVar(&flagCleanupForce, "force", false, "Skip confirmation for dirty worktrees")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	if !flagCleanupAll && len(args) == 0 {
		return fmt.Errorf("provide a branch name or use --all to clean up everything")
	}
	if flagCleanupAll && len(args) > 0 {
		return fmt.Errorf("cannot use --all with a specific branch")
	}

	st, err := state.Load()
	if err != nil {
		return err
	}

	if flagCleanupAll {
		return cleanupAll(st)
	}

	return cleanupBranch(st, args[0])
}

// cleanupAll removes every active delegation.
func cleanupAll(st *state.State) error {
	if len(st.Delegations) == 0 {
		fmt.Println("No active delegations to clean up.")
		return nil
	}

	// Collect branches to avoid mutating map during iteration.
	branches := make([]string, 0, len(st.Delegations))
	for b := range st.Delegations {
		branches = append(branches, b)
	}

	var firstErr error
	for _, branch := range branches {
		fmt.Printf("--- %s ---\n", branch)
		if err := cleanupBranch(st, branch); err != nil {
			fmt.Fprintf(os.Stderr, "Error cleaning up %s: %v\n", branch, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// cleanupBranch removes a single delegation by branch name.
func cleanupBranch(st *state.State, branch string) error {
	d, ok := st.Get(branch)
	if !ok {
		return fmt.Errorf("no delegation found for branch %q -- check 'openshell-delegate list'", branch)
	}

	// Check for uncommitted changes in worktree
	if _, err := os.Stat(d.Worktree); err == nil {
		dirty, err := worktree.IsDirty(d.Worktree)
		if err == nil && dirty && !flagCleanupForce {
			fmt.Fprintf(os.Stderr, "WARNING: worktree %s has uncommitted changes.\n", d.Worktree)
			fmt.Fprintf(os.Stderr, "You may want to pull first: openshell-delegate pull %s\n\n", branch)
			fmt.Fprint(os.Stderr, "Continue with cleanup? [y/N] ")

			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// Delete sandbox (best effort)
	fmt.Printf("Deleting sandbox: %s\n", d.Sandbox)
	if err := sandbox.Delete(d.Sandbox); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: could not delete sandbox: %v\n", err)
	}

	// Remove worktree (best effort)
	if _, err := os.Stat(d.Worktree); err == nil {
		fmt.Printf("Removing worktree: %s\n", d.Worktree)
		worktree.Remove(d.Repo, d.Worktree, d.Branch)
	}

	// Remove the .delegate directory if it is now empty.
	cleanupDelegateDir(d.Repo)

	// Remove state entry
	st.Remove(branch)
	if err := state.Save(st); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	fmt.Printf("\nCleaned up: %s\n", branch)
	return nil
}

// cleanupDelegateDir removes the .delegate directory inside the repo if it
// exists and is empty (no remaining worktrees).
func cleanupDelegateDir(repoPath string) {
	delegateDir := filepath.Join(repoPath, ".delegate")

	entries, err := os.ReadDir(delegateDir)
	if err != nil {
		return // directory doesn't exist or can't be read
	}

	if len(entries) == 0 {
		if err := os.Remove(delegateDir); err == nil {
			fmt.Printf("Removed empty .delegate directory\n")
		}
	}
}
