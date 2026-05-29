package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "openshell-delegate",
	Short: "Delegate tasks to OpenShell sandboxes",
	Long: `openshell-delegate prepares OpenShell sandboxes for task delegation using
the git worktree pattern. It handles worktree creation, sandbox provisioning,
credential injection, and file transfer so you can focus on defining the task
and reviewing the result.

Workflow:
  1. openshell-delegate run --branch fix-scaling --task-file task.md
  2. openshell sandbox connect od-fix-scaling
  3. cd /sandbox/workspace && opencode
  4. openshell-delegate pull fix-scaling
  5. cd worktrees/fix-scaling && git diff
  6. openshell-delegate cleanup fix-scaling`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(cleanupCmd)
}
