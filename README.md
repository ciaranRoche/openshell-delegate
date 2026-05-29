# openshell-delegate

Delegate tasks to [OpenShell](https://github.com/NVIDIA/OpenShell) sandboxes using the git worktree pattern.

The CLI handles the plumbing -- worktree creation, sandbox provisioning, credential injection, file transfer -- so you can focus on defining the task and reviewing the result. It doesn't run the agent for you. You connect to the sandbox, run your preferred agent (opencode, claude, etc.), and pull the results back when done.

## Install

```bash
go install github.com/ciaranRoche/openshell-delegate@latest
```

Or build from source:

```bash
git clone https://github.com/ciaranRoche/openshell-delegate.git
cd openshell-delegate
make build
# Binary at ./bin/openshell-delegate
```

## Quick Start

```bash
# From your repo, delegate a task
cd ~/Projects/myproject
openshell-delegate run --branch fix-scaling --task "Fix the scaling bug. Run make test."

# Connect to the sandbox
openshell sandbox connect od-fix-scaling

# Inside the sandbox: your repo is at /sandbox/workspace
cd /sandbox/workspace
cat TASK.md     # Read the task
opencode        # Run your agent

# When done, exit the sandbox and pull results
openshell-delegate pull fix-scaling

# Review locally
cd ../worktrees/fix-scaling
git diff
git add -A && git commit -m "fix scaling"
git push origin fix-scaling

# Clean up
openshell-delegate cleanup fix-scaling
```

## Commands

### `run`

Prepare a sandbox with a worktree and optional task file.

```
openshell-delegate run --branch <name> [--task <string> | --task-file <path>] [flags]

Flags:
  --branch <name>        Branch name (required)
  --task <string>        Task description (written to TASK.md in workspace)
  --task-file <path>     Task from a file (copied as TASK.md)
  --name <name>          Sandbox name (default: od-<branch>)
  --image <image>        Override container image
  --policy <file>        Override network policy
```

What it does:

1. Creates a git worktree at `<repo-parent>/worktrees/<branch>`
2. Writes `TASK.md` if `--task` or `--task-file` provided
3. Appends sandbox context to `AGENTS.md`
4. Creates an OpenShell sandbox from the configured image
5. Injects credentials into the sandbox
6. Uploads the worktree to `/sandbox/workspace`
7. Prints connection instructions

### `pull`

Download the workspace from a sandbox back to the local worktree.

```
openshell-delegate pull <branch>
```

### `list`

List active delegations and their status.

```
openshell-delegate list
```

### `cleanup`

Remove a sandbox and its worktree.

```
openshell-delegate cleanup <branch>
```

Warns if the worktree has uncommitted changes before removing.

## Configuration

Create `~/.config/openshell-delegate/config.yaml`:

```yaml
# Container image for sandboxes
image: quay.io/myorg/my-sandbox:latest

# Network policy YAML (required)
policy: ~/.config/openshell-delegate/policy.yaml

# OpenShell provider name (for LLM credential injection)
provider: opencode-vertex

# Environment variables to forward from host into sandbox
env:
  - GITHUB_TOKEN
  - JIRA_USERNAME
  - JIRA_API_TOKEN

# Fixed environment variables (not read from host)
static_env:
  GOOGLE_CLOUD_PROJECT: my-gcp-project
  VERTEX_LOCATION: us-east5
  VERTEXAI_PROJECT: my-gcp-project
  VERTEXAI_LOCATION: us-east5
  CLAUDE_CODE_USE_VERTEX: "1"
  ANTHROPIC_VERTEX_PROJECT_ID: my-gcp-project
  CLOUD_ML_REGION: us-east5

# Files to upload into sandbox at creation time
uploads:
  - src: ~/.config/gcloud/application_default_credentials.json
    dst: /sandbox/.config/gcloud/application_default_credentials.json
```

All config values can be overridden by CLI flags.

## Prerequisites

- [OpenShell](https://github.com/NVIDIA/OpenShell) installed and gateway running
- Podman or Docker for sandbox containers
- Git
- rsync (for pulling results)

## How It Works

```
Host                                    Sandbox (OpenShell container)
----                                    -------
~/Projects/myproject/                   /sandbox/workspace/
  (your working copy, untouched)          (worktree snapshot)
                                          TASK.md (your task description)
~/Projects/worktrees/fix-scaling/         AGENTS.md (project rules + sandbox context)
  (git worktree on branch fix-scaling)
  (results pulled back here)
```

The worktree pattern keeps your working copy untouched. The agent works on an isolated branch inside the sandbox. When you pull results back, the worktree is a regular git checkout you can diff, commit, and push.

## Shorthand Alias

Add to your shell profile if you want a shorter command:

```bash
alias osd='openshell-delegate'
```

## License

MIT
