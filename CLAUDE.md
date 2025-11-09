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
**Purpose**: Initialize a new ramp project with interactive setup (similar to `npm init`).
**How it works**:
- Checks if `.ramp/ramp.yaml` already exists (errors if found with helpful message)
- Runs interactive prompts using huh forms library:
  - Project name (defaults to current directory name)
  - Branch prefix (defaults to "feature/")
  - Repositories (iterative: asks for Git URL, then "add another?" until done)
  - All repos automatically use `path: repos`
  - Setup script (defaults to Yes)
  - Cleanup script (defaults to Yes)
  - Port management (defaults to No, prompts for base port if Yes)
  - Doctor command for environment checks (defaults to Yes)
  - Clone repositories now (defaults to Yes)
- Creates directory structure: `.ramp/scripts/`, `repos/`, `trees/`
- Generates `ramp.yaml` with proper formatting and `auto_refresh: true` for all repos
- Generates sample scripts with actual repository environment variable names
- Optionally calls `ramp install` to clone repos immediately

#### `ramp install`
**Purpose**: Clone all configured repositories from ramp.yaml into their configured locations.
**How it works**:
- Searches upward from current directory to find `.ramp/ramp.yaml` configuration file
- Reads repository configurations
- Creates parent directories if needed (default: `repos/`)
- For each repository:
  - Checks if already cloned (skips if repository exists)
  - Clones using `git clone` into `repos/<repo-name>/`
  - Validates git repository structure
- Provides detailed progress feedback with success/error states
- Used automatically by other commands via `AutoInstallIfNeeded()` function

#### `ramp up [feature-name]`
**Purpose**: Create a new feature branch with git worktrees for all configured repositories.
**How it works**:
- Auto-installs the project if repositories aren't cloned yet (calls `ramp install` internally)
- Auto-refreshes repositories that have `auto_refresh` enabled (defaults to true if not specified)
- Creates a `trees/<feature-name>/` directory structure for isolated feature development
- For each configured repository:
  - When using `--target`: attempts to resolve target branch/feature, falls back to default behavior if target not found in specific repo
  - Detects if branch already exists locally or remotely
  - Creates git worktree using `git worktree add` with appropriate branch strategy:
    - If `--target` specified and found: creates new branch from target source
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
- Supports `--no-prefix` flag to disable branch prefix entirely (mutually exclusive with --prefix)
- Supports `--target` flag to create feature from existing feature name, local branch, or remote branch
- Supports `--from` flag to create from remote branch with automatic naming (mutually exclusive with --target, --prefix, --no-prefix):
  - `ramp up --from claude/feature-123` creates `trees/feature-123/` with branch `claude/feature-123` tracking `origin/claude/feature-123`
  - `ramp up my-name --from claude/feature-123` creates `trees/my-name/` with branch `claude/feature-123` tracking `origin/claude/feature-123`
  - Automatically derives prefix from path before last "/" and feature name from last segment
  - Always prepends `origin/` to remote branch reference
- Supports `--refresh` flag to force refresh all repositories before creating feature (overrides auto_refresh config)
- Supports `--no-refresh` flag to skip refresh for all repositories (overrides auto_refresh config)

#### `ramp down <feature-name>`
**Purpose**: Clean up a feature branch by removing worktrees, branches, and allocated resources.
**How it works**:
- Checks for uncommitted changes across all worktrees and prompts for confirmation if found
- Runs optional cleanup script (if configured) before removal
- For each repository:
  - Detects actual branch name from worktree (handles prefix variations)
  - Removes git worktree using `git worktree remove --force`
  - Deletes local branch using `git branch -D`
  - Runs `git fetch --prune` to clean up stale remote tracking branches
- Releases allocated port number and updates port allocations file
- Removes entire `trees/<feature-name>/` directory structure
- Provides detailed progress feedback with warnings for any failures

#### `ramp prune`
**Purpose**: Automatically clean up all merged feature branches in one command.
**How it works**:
- Scans all features in `trees/` directory and categorizes them using git merge-base
- Identifies features that have been merged into their default branch (excludes "CLEAN" features that never had commits)
- Displays summary of all merged features that will be removed
- Asks for single confirmation: "Remove all N merged features? (y/N)"
- If confirmed, iterates through each merged feature and performs cleanup:
  - Runs optional cleanup script (if configured)
  - Removes git worktrees using `git worktree remove --force`
  - Deletes local branches using `git branch -D`
  - Runs `git fetch --prune` to clean up stale remote tracking branches
  - Releases allocated port numbers
  - Removes feature directories from `trees/`
- Continues with remaining features if individual cleanups fail (non-blocking errors)
- Displays final summary showing success count and any failures
- Useful for batch cleanup after merging multiple feature branches

#### `ramp refresh`
**Purpose**: Update all source repositories by pulling changes from their remotes.
**How it works**:
- For each configured repository in the source directory:
  - Runs `git fetch --all` to update remote tracking information
  - Detects current branch and checks for remote tracking branch
  - If remote tracking exists, runs `git pull` to merge remote changes
  - If no remote tracking, reports status but skips pull operation
- Provides detailed status for each repository including success/failure states

#### `ramp rebase <branch-name>`
**Purpose**: Switch all source repositories to an existing branch across the multi-repo project.
**How it works**:
- Auto-initializes the project if repositories aren't cloned yet (calls `ramp init` internally)
- Validates that the target branch exists in at least one repository (lenient mode)
- For each configured repository:
  - Checks if branch exists locally or remotely using exact name matching
  - If branch exists: switches to that branch (local checkout or remote tracking)
  - If branch doesn't exist: skips repository and keeps it on current branch
- Handles uncommitted changes by prompting user to stash them before switching
- Implements atomic operations with rollback - if any switch fails, reverts all successful switches
- Provides detailed feedback showing which repositories were switched vs skipped
- Restores stashed changes after successful branch switching

#### `ramp run <command-name> [feature-name]`
**Purpose**: Execute custom commands defined in the ramp configuration within feature context.
**How it works**:
- Looks up command definition in `.ramp/ramp.yaml` configuration
- If no feature name provided, attempts to auto-detect from current working directory
- Executes the command script from within the feature's trees directory
- Sets up full environment context identical to setup/cleanup scripts
- Provides progress feedback and error handling for command execution

#### `ramp status`
**Purpose**: Display comprehensive project and repository status information, including all active feature worktrees.
**How it works**:
- Automatically fetches remote information from all repositories in parallel before displaying status
- Shows project name from configuration
- For each configured source repository:
  - Displays repository name and absolute path
  - Shows current branch with remote tracking status (up to date, ahead/behind commits)
  - Indicates clean vs. uncommitted changes with visual status icons (✅ for clean, ⚠️ for changes)
  - Reports errors for missing or problematic repositories (❌)
- Displays project information:
  - Count of active feature worktrees
  - Port allocation usage (only shown if ports are configured in ramp.yaml)
- Shows detailed list of all active feature worktrees:
  - Scans `trees/` directory for existing feature directories
  - Sorts features chronologically by creation date (oldest to newest)
  - For each feature, shows which repositories have active worktrees
  - Displays tree structure showing feature name and associated repository worktrees
  - Handles cases where features exist but may have incomplete worktree setups
- Provides comprehensive overview of entire project state and all active development branches

#### `ramp version`
**Purpose**: Display the current version of the ramp CLI tool.
**How it works**:
- Shows the version number of the installed ramp binary
- Simple informational command with no side effects

### Global Flags
- `-v, --verbose` - Shows detailed output during all operations, disabling progress spinners for full command visibility

## Architecture

### Command Structure
The application uses the Cobra CLI framework with commands organized in `cmd/`:
- `cmd/root.go` - Main command definition, CLI entry point, and global flag handling
- `cmd/init.go` - Repository initialization logic with auto-initialization support
- `cmd/up.go` - Feature branch and worktree creation with smart branch handling
- `cmd/down.go` - Feature cleanup with safety checks and confirmation prompts
- `cmd/prune.go` - Batch cleanup of merged features with single confirmation prompt
- `cmd/refresh.go` - Source repository synchronization
- `cmd/rebase.go` - Repository branch switching with atomic operations and rollback
- `cmd/run.go` - Custom command execution with environment context
- `cmd/status.go` - Project and repository status display with comprehensive information and feature worktree listing
- `cmd/version.go` - Version display command

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
- `SaveConfig(cfg, projectDir)` - Writes Config to ramp.yaml with custom formatting and spacing
- `GetRepos()` - Returns map of repository name to configuration
- `GenerateEnvVarName(repoName)` - Converts repo names to valid environment variable names

#### `internal/scaffold/`
**Purpose**: Project scaffolding and template generation for ramp init.
**Key Types**:
- `ProjectData` - Holds collected information from interactive init (name, repos, options, commands)
- `RepoData` - Repository URL and path information

**Key Functions**:
- `CreateProject(projectDir, data)` - Orchestrates complete project creation
- `CreateDirectoryStructure(projectDir)` - Creates `.ramp/scripts/`, `repos/`, `trees/` directories
- `GenerateConfigFile(projectDir, data)` - Creates formatted ramp.yaml with auto_refresh enabled
- `GenerateSetupScript(projectDir, repos)` - Creates setup.sh with actual repository env var names
- `GenerateCleanupScript(projectDir, repos)` - Creates cleanup.sh with repo env vars
- `GenerateSampleCommand(projectDir, name, repos)` - Creates custom command scripts (e.g., doctor.sh)
- `extractRepoName(gitURL)` - Extracts repository name from git URL for naming

#### `internal/git/`
**Purpose**: Git operations and worktree management.
**Key Functions**:
- `Clone(repoURL, destDir)` - Clones repositories with progress feedback
- `CreateWorktree(repoDir, worktreeDir, branchName)` - Intelligent worktree creation with branch detection
- `CreateWorktreeFromSource(repoDir, worktreeDir, branchName, sourceBranch, repoName)` - Creates worktree with new branch from specified source branch
- `LocalBranchExists()` / `RemoteBranchExists()` / `BranchExists()` - Branch existence checking with exact name matching
- `RemoveWorktree()` / `DeleteBranch()` - Cleanup operations with force flags
- `HasUncommittedChanges()` - Safety check using `git status --porcelain`
- `GetWorktreeBranch()` - Extracts actual branch name from worktree
- `GetCurrentBranch(repoDir)` - Gets current branch name in a repository
- `FetchAll()` / `FetchAllQuiet()` / `Pull()` - Repository synchronization operations
- `FetchPrune(repoDir)` - Prunes stale remote tracking branches using `git fetch --prune`
- `HasRemoteTrackingBranch()` - Detects if current branch tracks a remote
- `Checkout(repoDir, branchName)` - Switches to existing local branch
- `CheckoutRemoteBranch(repoDir, branchName)` - Creates and switches to remote tracking branch
- `StashChanges(repoDir)` / `PopStash(repoDir)` - Stash management for uncommitted changes
- `FetchBranch(repoDir, branchName)` - Fetches specific branch from remote
- `ResolveSourceBranch(repoDir, target, effectivePrefix)` - Resolves target to actual source branch (feature name, local branch, or remote branch)
- `GetRemoteTrackingStatus(repoDir)` - Gets ahead/behind commit status relative to remote tracking branch
- `IsGitRepo(dir)` - Checks if directory is a git repository

**Internal Helper Functions** (not typically called directly):
- `getRemoteBranchName()` - Internal helper to find matching remote branch name
- `validateSourceBranch()` - Internal validation for source branch existence
- `validateRemoteBranch()` - Internal validation for remote branch references

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
  - path: repos                      # Local directory path (relative to project root)
    git: git@github.com:owner/repo.git  # Git clone URL
    auto_refresh: true               # Optional: auto-refresh before 'ramp up' (default: true)
  - path: repos
    git: https://github.com/owner/other-repo.git
    auto_refresh: true               # Optional: auto-refresh before 'ramp up' (default: true)

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
├── repos/                           # Source repository clones
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

#### Auto-Installation
Most commands automatically run `ramp install` if they detect uninstalled repositories, ensuring seamless workflow even when starting from a clean state.

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

## Testing Infrastructure

Ramp has a comprehensive test suite designed to protect backwards compatibility and enable test-driven development. The test suite uses real git operations for realistic integration testing.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests for specific package
go test ./cmd -v

# Run specific test
go test ./cmd -run TestUpBasic -v

# Run tests with verbose output (disables progress spinners)
go test ./... -v
```

### Test Coverage

Current test coverage (as of latest update):
- **cmd package**: 52.6% (55 tests covering main commands)
- **internal/config**: 96.7% (54 tests for configuration management)
- **internal/git**: 46.1% (58 tests for git operations)
- **internal/scaffold**: 88.9% (23 tests for project scaffolding)
- **internal/ports**: 74.2% (3 tests for port allocation)
- **Total**: 256 tests across all packages

### Test Organization

Tests are colocated with source code following Go conventions:
- `cmd/*_test.go` - Command integration tests
- `internal/*/*.go` - Package unit tests
- `cmd/test_helpers.go` - Shared test infrastructure

### Test Infrastructure Helpers

**TestProject** - Manages complete test project setup:
```go
tp := NewTestProject(t)          // Creates isolated test project
repo1 := tp.InitRepo("repo1")    // Creates repo with bare remote
cleanup := tp.ChangeToProjectDir() // Changes to project directory
defer cleanup()                   // Restores original directory
```

**TestRepo** - Represents a test repository:
- `SourceDir` - Path to cloned source repository
- `RemoteDir` - Path to bare remote repository
- `Name` - Repository name

**Helper Functions**:
- `runGitCmd(t, dir, args...)` - Executes git commands with automatic error handling
- `tp.WorktreeExists(feature, repo)` - Checks if worktree exists
- `tp.FeatureExists(feature)` - Checks if feature directory exists

### Testing Patterns

1. **Table-Driven Tests with Subtests**
```go
tests := []struct {
    name string
    input string
    want string
}{
    {"basic", "input1", "output1"},
    {"edge case", "input2", "output2"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := function(tt.input)
        if got != tt.want {
            t.Errorf("got %v, want %v", got, tt.want)
        }
    })
}
```

2. **Real Git Operations** (No Mocking)
- Tests use actual git repositories for realistic scenarios
- Bare remotes are created for push/pull testing
- Each test gets isolated temporary directories via `t.TempDir()`

3. **Test Isolation**
- Each test creates its own project in a temporary directory
- No shared state between tests
- Automatic cleanup on test completion

4. **Testing Both Success and Failure Paths**
- All commands test happy path scenarios
- Error handling is validated with expected error messages
- Edge cases are explicitly tested (missing repos, empty projects, etc.)

### What's Tested

**Core Commands** (Integration Tests):
- ✅ `ramp up` - Feature creation with various branch scenarios
- ✅ `ramp down` - Feature cleanup with safety checks
- ✅ `ramp install` - Repository cloning and validation
- ✅ `ramp refresh` - Repository synchronization
- ✅ `ramp prune` - Merged feature detection and cleanup
- ✅ `ramp rebase` - Branch switching with rollback
- ✅ `ramp run` - Custom command execution with environment
- ✅ `ramp status` - Comprehensive status display
- ✅ `ramp version` - Version information
- ❌ `ramp init` - Not tested (interactive, uses huh forms library)

**Internal Packages** (Unit Tests):
- ✅ Config loading, saving, and validation
- ✅ Git operations (clone, worktree, branch, merge detection)
- ✅ Project scaffolding and template generation
- ✅ Port allocation and management
- ✅ Environment variable generation

**Key Scenarios Tested**:
- Backwards compatibility (auto_refresh defaults, config migration)
- Multi-repository workflows
- Branch handling (local, remote, non-existent)
- Uncommitted changes detection
- Port allocation conflicts
- Nested feature paths
- Missing repositories
- Empty projects
- Error recovery and rollback

### Known Limitations

1. **Interactive Commands**: `ramp init` is not tested due to interactive stdin requirements
2. **Coverage Gaps**: ~47% of cmd code paths not yet covered
3. **Performance**: No benchmark tests for performance-critical operations
4. **UI Testing**: Progress spinner and output formatting not validated

### Adding New Tests

When adding new functionality:

1. **Write tests first** (TDD approach encouraged)
2. **Use existing helpers** (`TestProject`, `TestRepo`, `runGitCmd`)
3. **Test both success and failure** paths
4. **Use descriptive test names** (`TestUpWithCustomPrefix`)
5. **Add table-driven tests** for multiple scenarios
6. **Check error messages** for user-facing errors
7. **Run tests locally** before committing:
   ```bash
   go test ./... -cover
   ```

### Bug Fixes Found Through Testing

Tests have already caught several bugs:
- **max_ports persistence**: Config was not saving max_ports field (fixed in config.go:208-210)
- **Empty project handling**: Installation check incorrectly failed for projects with no repos
- **Git initialization**: Tests needed `commit.gpgsign false` to work in all environments

This architecture enables Ramp to manage complex multi-repository workflows while providing a smooth developer experience through intelligent automation, safety checks, and comprehensive feedback systems.