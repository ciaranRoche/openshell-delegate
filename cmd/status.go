package cmd

import (
	"fmt"
	"time"

	"github.com/ciaranRoche/openshell-delegate/internal/sandbox"
	"github.com/ciaranRoche/openshell-delegate/internal/state"
	"github.com/ciaranRoche/openshell-delegate/internal/worktree"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <branch>",
	Short: "Show status of a delegation",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	branch := args[0]

	st, err := state.Load()
	if err != nil {
		return err
	}

	d, ok := st.Get(branch)
	if !ok {
		return fmt.Errorf("no delegation found for branch %q", branch)
	}

	status := "Gone"
	if sandbox.IsReady(d.Sandbox) {
		status = "Ready"
	}

	repoName := worktree.RepoName(d.Repo)
	duration := formatDuration(time.Since(d.Created))

	fmt.Printf("Sandbox:   %s\n", d.Sandbox)
	fmt.Printf("State:     %s\n", status)
	fmt.Printf("Running:   %s\n", duration)
	fmt.Printf("Repo:      %s\n", repoName)
	fmt.Printf("Worktree:  %s\n", d.Worktree)

	return nil
}

// formatDuration returns a human-readable duration string.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}
