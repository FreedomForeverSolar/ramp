# CLAUDE.md

This file provides comprehensive guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ramp is a sophisticated CLI tool for managing multi-repository development workflows using git worktrees and automated setup scripts. It enables developers to work on features that span multiple repositories simultaneously by creating isolated working directories, complete with custom setup scripts, port management, and cleanup automation.

## Core Commands

### Build and Development
- `go build -o ramp .` - Builds the ramp binary from Go source code in the current directory
- `./install.sh` - Builds the project and installs the binary to `/usr/local/bin` (requires sudo privileges)
- `go run . --help` - Runs the application without building a binary to show available commands and usage
- `go test ./...` - Runs all Go tests in the project using the standard Go testing framework

### CLI Usage

#### `ramp init`
**Purpose**: Initialize a ramp project by cloning all configured repositories from their remote origins.
**How it works**: 
- Searches upward from current directory to find `.ramp/ramp.yaml` configuration file
- Reads repository configurations and creates the local directory structure
- Clones each repository using `git clone` into the configured paths under the project directory
- Creates parent directories as needed and validates git repository structure
- Provides detailed progress feedback with success/error states

#### `ramp up <feature-name>`
**Purpose**: Create a new feature branch with git worktrees for all configured repositories.
**How it works**:
- Auto-initializes the project if repositories aren't cloned yet (calls `ramp init` internally)
- Creates a `trees/<feature-name>/` directory structure for isolated feature development
- For each configured repository:
  - Detects if branch already exists locally or remotely 
  - Creates git worktree using `git worktree add` with appropriate branch strategy:
    - If local branch exists: uses existing local branch
    - If only remote branch exists: creates local tracking branch
    - If neither exists: creates new branch from default branch
- Allocates a unique port number for the feature (stored in `.ramp/port_allocations.json`)
- Runs optional setup script with environment variables:
  - `RAMP_PROJECT_DIR`: absolute path to project root
  - `RAMP_TREES_DIR`: absolute path to feature's trees directory  
  - `RAMP_WORKTREE_NAME`: the feature name
  - `RAMP_PORT`: allocated port number for this feature
  - `RAMP_REPO_PATH_<REPO>`: path variables for each repository
- Supports `--prefix` flag to override branch naming prefix

#### `ramp down <feature-name>`
**Purpose**: Clean up a feature branch by removing worktrees, branches, and allocated resources.
**How it works**:
- Checks for uncommitted changes across all worktrees and prompts for confirmation if found
- Runs optional cleanup script (if configured) before removal
- For each repository:
  - Detects actual branch name from worktree (handles prefix variations)
  - Removes git worktree using `git worktree remove --force`
  - Deletes local branch using `git branch -D`
- Releases allocated port number and updates port allocations file
- Removes entire `trees/<feature-name>/` directory structure
- Provides detailed progress feedback with warnings for any failures

#### `ramp list`
**Purpose**: List all current feature worktrees and their status, sorted by creation date.
**How it works**:
- Scans `trees/` directory for existing feature directories
- Collects creation timestamps for each feature directory using file system metadata
- Sorts features chronologically by creation date (oldest to newest)
- For each feature, shows which repositories have active worktrees
- Displays tree structure showing feature name and associated repository worktrees
- Handles cases where features exist but may have incomplete worktree setups

#### `ramp refresh`
**Purpose**: Update all source repositories by pulling changes from their remotes.
**How it works**:
- For each configured repository in the source directory:
  - Runs `git fetch --all` to update remote tracking information
  - Detects current branch and checks for remote tracking branch
  - If remote tracking exists, runs `git pull` to merge remote changes
  - If no remote tracking, reports status but skips pull operation
- Provides detailed status for each repository including success/failure states

#### `ramp run <command-name> [feature-name]`
**Purpose**: Execute custom commands defined in the ramp configuration within feature context.
**How it works**:
- Looks up command definition in `.ramp/ramp.yaml` configuration
- If no feature name provided, attempts to auto-detect from current working directory
- Executes the command script from within the feature's trees directory
- Sets up full environment context identical to setup/cleanup scripts
- Provides progress feedback and error handling for command execution

### Global Flags
- `-v, --verbose` - Shows detailed output during all operations, disabling progress spinners for full command visibility

## Architecture

### Command Structure
The application uses the Cobra CLI framework with commands organized in `cmd/`:
- `cmd/root.go` - Main command definition, CLI entry point, and global flag handling
- `cmd/init.go` - Repository initialization logic with auto-initialization support
- `cmd/up.go` - Feature branch and worktree creation with smart branch handling
- `cmd/down.go` - Feature cleanup with safety checks and confirmation prompts
- `cmd/list.go` - Feature discovery and status reporting
- `cmd/refresh.go` - Source repository synchronization
- `cmd/run.go` - Custom command execution with environment context

### Core Internal Packages

#### `internal/config/`
**Purpose**: Configuration file parsing and project discovery.
**Key Types**:
- `Config` - Main configuration structure mapping YAML to Go structs
- `Repo` - Repository configuration with git URL, path, and default branch
- `Command` - Custom command definitions with name and script path

**Key Functions**:
- `FindRampProject(startDir)` - Recursively searches up directory tree for `.ramp/ramp.yaml`
- `LoadConfig(projectDir)` - Parses YAML configuration file and validates structure
- `GetRepos()` - Returns map of repository name to configuration
- `GenerateEnvVarName(repoName)` - Converts repo names to valid environment variable names

#### `internal/git/`
**Purpose**: Git operations and worktree management.
**Key Functions**:
- `Clone(repoURL, destDir)` - Clones repositories with progress feedback
- `CreateWorktree(repoDir, worktreeDir, branchName)` - Intelligent worktree creation with branch detection
- `LocalBranchExists()` / `RemoteBranchExists()` - Branch existence checking
- `RemoveWorktree()` / `DeleteBranch()` - Cleanup operations with force flags
- `HasUncommittedChanges()` - Safety check using `git status --porcelain`
- `GetWorktreeBranch()` - Extracts actual branch name from worktree
- `FetchAll()` / `Pull()` - Repository synchronization operations
- `HasRemoteTrackingBranch()` - Detects if current branch tracks a remote

#### `internal/ports/`
**Purpose**: Port allocation management for features.
**Key Type**: `PortAllocations` - Manages feature-to-port mappings with persistence
**Key Functions**:
- `AllocatePort(featureName)` - Assigns unique port numbers within configured range
- `ReleasePort(featureName)` - Frees port allocation when feature is cleaned up
- `findNextAvailablePort()` - Scans for first available port in range
- Port data persisted in `.ramp/port_allocations.json`

#### `internal/ui/`
**Purpose**: User interface and progress feedback.
**Key Type**: `ProgressUI` - Manages spinner animations and status messages
**Key Functions**:
- `NewProgress()` - Creates progress indicator with cyan spinner
- `Start()` / `Success()` / `Error()` / `Warning()` / `Info()` - Status reporting methods
- `RunCommandWithProgress()` - Executes shell commands with progress feedback
- `OutputCapture` - Captures and conditionally displays command output
- Respects global `--verbose` flag to switch between spinner and direct output modes

### Configuration Schema
Projects require a `.ramp/ramp.yaml` file with complete configuration:

```yaml
name: project-name                    # Display name for the project
repos:                               # Array of repository configurations
  - path: source                     # Local directory path (relative to project root)
    git: git@github.com:owner/repo.git  # Git clone URL
    default_branch: main             # Default branch for new worktrees
  - path: source
    git: https://github.com/owner/other-repo.git
    default_branch: develop

setup: scripts/setup.sh              # Optional: script to run after 'ramp up'
cleanup: scripts/cleanup.sh          # Optional: script to run during 'ramp down'

default-branch-prefix: feature/      # Optional: prefix for new branch names

base_port: 3000                      # Optional: starting port for allocation (default: 3000)
max_ports: 100                       # Optional: maximum ports to allocate (default: 100)

commands:                            # Optional: custom commands for 'ramp run'
  - name: open                       # Command name for 'ramp run open'
    command: scripts/open.sh         # Script path (relative to .ramp/)
  - name: deploy
    command: scripts/deploy.sh
```

### Directory Structure
```
project-root/
├── .ramp/
│   ├── ramp.yaml                    # Main configuration file
│   ├── port_allocations.json       # Auto-generated port tracking
│   └── scripts/                     # Optional setup/cleanup/command scripts
│       ├── setup.sh
│       ├── cleanup.sh
│       └── custom-command.sh
├── source/                          # Source repository clones
│   ├── repo-name/                   # Cloned repositories
│   └── other-repo/
└── trees/                           # Feature worktrees
    ├── feature-name/                # Individual feature directories
    │   ├── repo-name/               # Worktree for each repository
    │   └── other-repo/
    └── other-feature/
```

### Environment Variables for Scripts
All setup, cleanup, and custom command scripts receive these environment variables:

- `RAMP_PROJECT_DIR` - Absolute path to project root directory
- `RAMP_TREES_DIR` - Absolute path to current feature's trees directory
- `RAMP_WORKTREE_NAME` - Name of the current feature
- `RAMP_PORT` - Allocated port number for this feature (if port management enabled)
- `RAMP_REPO_PATH_<REPO_NAME>` - Absolute path to each repository's source directory

Repository names are converted to valid environment variable names by:
1. Converting to uppercase
2. Replacing non-alphanumeric characters with underscores
3. Removing consecutive underscores
4. Prefixing with `RAMP_REPO_PATH_`

### Key Behavioral Features

#### Auto-Initialization
Most commands automatically run `ramp init` if they detect uninitialized repositories, ensuring seamless workflow even when starting from a clean state.

#### Smart Branch Handling
The `up` command intelligently handles various branch scenarios:
- Creates new branches from default branch when needed
- Uses existing local branches without modification
- Creates local tracking branches for existing remote branches
- Provides detailed feedback about branch creation strategy

#### Safety Mechanisms
The `down` command includes multiple safety checks:
- Scans for uncommitted changes before deletion
- Prompts for explicit confirmation when changes would be lost
- Continues cleanup even if individual operations fail
- Provides detailed warning messages for partial failures

#### Port Management
Automatic port allocation ensures each feature gets a unique port number:
- Persisted across ramp sessions
- Automatically allocated on `up` and released on `down`
- Available to setup/cleanup scripts via `RAMP_PORT` environment variable
- Configurable base port and range limits

#### Progress Feedback
Comprehensive progress reporting with two modes:
- **Normal mode**: Animated spinners with status updates and emoji indicators
- **Verbose mode**: Direct command output for debugging and CI environments

This architecture enables Ramp to manage complex multi-repository workflows while providing a smooth developer experience through intelligent automation, safety checks, and comprehensive feedback systems.