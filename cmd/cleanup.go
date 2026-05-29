package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ciaranRoche/openshell-delegate/internal/sandbox"
	"github.com/ciaranRoche/openshell-delegate/internal/state"
	"github.com/ciaranRoche/openshell-delegate/internal/worktree"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup <branch>",
	Short: "Remove sandbox and worktree for a delegation",
	Long: `Deletes the OpenShell sandbox, removes the git worktree, and cleans up
the delegation state. Warns if the worktree has uncommitted changes.`,
	Args: cobra.ExactArgs(1),
	RunE: runCleanup,
}

func runCleanup(cmd *cobra.Command, args []string) error {
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

	// Check for uncommitted changes in worktree
	if _, err := os.Stat(d.Worktree); err == nil {
		dirty, err := worktree.IsDirty(d.Worktree)
		if err == nil && dirty {
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

	// Remove state entry
	st.Remove(branch)
	if err := state.Save(st); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	fmt.Printf("\nCleaned up: %s\n", branch)
	return nil
}
