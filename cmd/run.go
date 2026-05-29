package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ciaranRoche/openshell-delegate/internal/config"
	"github.com/ciaranRoche/openshell-delegate/internal/creds"
	"github.com/ciaranRoche/openshell-delegate/internal/sandbox"
	"github.com/ciaranRoche/openshell-delegate/internal/state"
	"github.com/ciaranRoche/openshell-delegate/internal/worktree"
	"github.com/spf13/cobra"
)

const sandboxContext = `
## Sandbox Context

You are running inside an OpenShell sandbox with restricted network access.

If a task file exists at TASK.md in the workspace root, read it for your instructions.

Guidelines:
- Do NOT commit or push -- the developer will review changes locally
- Do NOT ask interactive questions -- make reasonable decisions and proceed
- If uncertain about something, leave a TODO comment in the code
- Run the project's test/verify commands to confirm correctness before finishing
`

var (
	flagBranch   string
	flagTask     string
	flagTaskFile string
	flagName     string
	flagImage    string
	flagPolicy   string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Prepare a sandbox with a worktree and task file",
	Long: `Creates a git worktree, spins up an OpenShell sandbox, uploads the worktree
and optional task file. The sandbox is ready for the developer to connect
and run their preferred agent (opencode, claude, etc.).`,
	RunE: runRun,
}

func init() {
	runCmd.Flags().StringVar(&flagBranch, "branch", "", "Branch name (required)")
	runCmd.Flags().StringVar(&flagTask, "task", "", "Task description (inline, written to TASK.md)")
	runCmd.Flags().StringVar(&flagTaskFile, "task-file", "", "Task from a file (copied as TASK.md)")
	runCmd.Flags().StringVar(&flagName, "name", "", "Sandbox name (default: od-<branch>)")
	runCmd.Flags().StringVar(&flagImage, "image", "", "Override container image")
	runCmd.Flags().StringVar(&flagPolicy, "policy", "", "Override network policy")
	_ = runCmd.MarkFlagRequired("branch")
}

func runRun(cmd *cobra.Command, args []string) error {
	if flagTask != "" && flagTaskFile != "" {
		return fmt.Errorf("cannot use both --task and --task-file")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("could not load config: %w", err)
	}

	// Apply flag overrides
	image := cfg.Image
	if flagImage != "" {
		image = flagImage
	}
	if image == "" {
		return fmt.Errorf("no image configured -- set 'image' in config.yaml or use --image")
	}

	policy := config.ExpandPath(cfg.Policy)
	if flagPolicy != "" {
		policy = flagPolicy
	}
	if policy == "" {
		return fmt.Errorf("no policy configured -- set 'policy' in config.yaml or use --policy")
	}

	sandboxName := flagName
	if sandboxName == "" {
		sandboxName = "od-" + flagBranch
	}

	// Find the git repo from cwd
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	repoPath, err := worktree.FindRepo(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	repoName := worktree.RepoName(repoPath)

	// Check for existing delegation
	st, err := state.Load()
	if err != nil {
		return err
	}
	if _, exists := st.Get(flagBranch); exists {
		return fmt.Errorf("delegation %q already exists -- run 'openshell-delegate cleanup %s' first", flagBranch, flagBranch)
	}

	// Validate credentials
	if errs := creds.Validate(cfg); len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "Credential validation failed:")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		return fmt.Errorf("fix the above issues and retry")
	}

	// Ensure .delegate is gitignored
	ensureDelegateGitignore(repoPath)

	// Create worktree
	fmt.Printf("Creating worktree: %s (branch: %s)\n", worktree.WorktreeDir(repoPath, flagBranch), flagBranch)
	wtDir, err := worktree.Create(repoPath, flagBranch)
	if err != nil {
		return err
	}

	// Write TASK.md if provided
	if flagTask != "" {
		taskPath := filepath.Join(wtDir, "TASK.md")
		if err := os.WriteFile(taskPath, []byte(flagTask+"\n"), 0o644); err != nil {
			return fmt.Errorf("could not write TASK.md: %w", err)
		}
		fmt.Println("Created TASK.md from --task")
	} else if flagTaskFile != "" {
		data, err := os.ReadFile(flagTaskFile)
		if err != nil {
			return fmt.Errorf("could not read task file: %w", err)
		}
		taskPath := filepath.Join(wtDir, "TASK.md")
		if err := os.WriteFile(taskPath, data, 0o644); err != nil {
			return fmt.Errorf("could not write TASK.md: %w", err)
		}
		fmt.Printf("Created TASK.md from %s\n", flagTaskFile)
	}

	// Append sandbox context to AGENTS.md
	agentsPath := filepath.Join(wtDir, "AGENTS.md")
	f, err := os.OpenFile(agentsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("could not open AGENTS.md: %w", err)
	}
	if _, err := f.WriteString(sandboxContext); err != nil {
		f.Close()
		return fmt.Errorf("could not append to AGENTS.md: %w", err)
	}
	f.Close()

	// Build launch script and upload map
	launchScript := creds.BuildLaunchScript(cfg)
	uploads := creds.UploadMap(cfg)

	// Create sandbox
	fmt.Printf("\nCreating sandbox: %s\n", sandboxName)
	fmt.Printf("  Image:  %s\n", image)
	fmt.Printf("  Policy: %s\n", filepath.Base(policy))
	fmt.Println()

	if err := sandbox.Create(sandboxName, image, policy, cfg.Provider, launchScript, uploads); err != nil {
		return fmt.Errorf("could not create sandbox: %w", err)
	}

	// Wait for ready
	fmt.Println("Waiting for sandbox...")
	if err := sandbox.WaitReady(sandboxName, 120*time.Second); err != nil {
		return err
	}
	fmt.Println("Sandbox is ready.")

	// Upload worktree
	fmt.Printf("\nUploading workspace: %s -> /sandbox/workspace\n", wtDir)
	if err := sandbox.Upload(sandboxName, wtDir, "/sandbox/workspace"); err != nil {
		return fmt.Errorf("could not upload workspace: %w", err)
	}

	// Save state
	st.Set(flagBranch, state.Delegation{
		Sandbox:  sandboxName,
		Repo:     repoPath,
		Worktree: wtDir,
		Branch:   flagBranch,
		Created:  time.Now(),
	})
	if err := state.Save(st); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	// Print instructions
	fmt.Println()
	fmt.Printf("Delegation ready: %s\n", sandboxName)
	fmt.Println()
	fmt.Printf("  Repo:      %s\n", repoName)
	fmt.Printf("  Worktree:  %s\n", wtDir)
	fmt.Printf("  Sandbox:   %s\n", sandboxName)
	if flagTask != "" || flagTaskFile != "" {
		fmt.Printf("  Task file: /sandbox/workspace/TASK.md\n")
	}
	fmt.Println()
	fmt.Printf("  Connect:   openshell sandbox connect %s\n", sandboxName)
	fmt.Printf("  Inside:    cd /sandbox/workspace && opencode\n")
	fmt.Println()
	fmt.Printf("  When done: openshell-delegate pull %s\n", flagBranch)
	fmt.Printf("  Cleanup:   openshell-delegate cleanup %s\n", flagBranch)

	return nil
}

// ensureDelegateGitignore adds .delegate to the repo's .gitignore if not already present.
func ensureDelegateGitignore(repoPath string) {
	gitignorePath := filepath.Join(repoPath, ".gitignore")

	// Read existing .gitignore
	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		// Check if .delegate is already ignored
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == ".delegate" || strings.TrimSpace(line) == ".delegate/" {
				return
			}
		}
	}

	// Append .delegate to .gitignore
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return // best effort
	}
	defer f.Close()

	// Add a newline before if file doesn't end with one
	if len(data) > 0 && data[len(data)-1] != '\n' {
		f.WriteString("\n")
	}
	f.WriteString(".delegate/\n")
}
