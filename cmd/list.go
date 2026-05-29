package cmd

import (
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/ciaranRoche/openshell-delegate/internal/sandbox"
	"github.com/ciaranRoche/openshell-delegate/internal/state"
	"github.com/ciaranRoche/openshell-delegate/internal/worktree"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List active delegations",
	RunE:    runList,
}

func runList(cmd *cobra.Command, args []string) error {
	st, err := state.Load()
	if err != nil {
		return err
	}

	if len(st.Delegations) == 0 {
		fmt.Println("No active delegations.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "BRANCH\tSANDBOX\tREPO\tSTATUS\tCREATED")

	for branch, d := range st.Delegations {
		repoName := worktree.RepoName(d.Repo)

		status := "Gone"
		if sandbox.IsReady(d.Sandbox) {
			status = "Ready"
		}

		created := d.Created.Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", branch, d.Sandbox, repoName, status, created)
	}

	return w.Flush()
}
