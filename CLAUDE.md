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
- Creates `.gitignore` at project root with entries for ramp-managed files:
  - `repos/` (source repository clones)
  - `trees/` (feature worktrees)
  - `.ramp/local.yaml` (local preferences)
  - `.ramp/port_allocations.json` (port allocations)
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
- Auto-prompts for local config if prompts are defined in `ramp.yaml` (calls `EnsureLocalConfig()` internally)
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
  - Processes `env_files` configuration to copy and template environment files into the worktree
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

#### `ramp config`
**Purpose**: Manage local preferences defined in project's `prompts` configuration.
**How it works**:
- Without flags: Interactively prompts user to set/update preferences
- `--show`: Displays current local preference values from `.ramp/local.yaml`
- `--reset`: Deletes local preferences file (will re-prompt on next `ramp up`)
- Preferences are stored in `.ramp/local.yaml` (gitignored)
- Preference values become environment variables (e.g., `RAMP_IDE`, `RAMP_DATABASE`)
- Available in setup/cleanup scripts and env_files templating
- Enables IDE-agnostic and tool-agnostic team workflows

#### `ramp down [feature-name]`
**Purpose**: Clean up a feature branch by removing worktrees, branches, and allocated resources.
**How it works**:
- If no feature name provided, attempts to auto-detect from current working directory
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
- `cmd/config.go` - Local preference management for IDE-agnostic workflows
- `cmd/down.go` - Feature cleanup with safety checks and confirmation prompts
- `cmd/prune.go` - Batch cleanup of merged features with single confirmation prompt
- `cmd/refresh.go` - Source repository synchronization
- `cmd/rebase.go` - Repository branch switching with atomic operations and rollback
- `cmd/run.go` - Custom command execution with environment context
- `cmd/status.go` - Project and repository status display with comprehensive information and feature worktree listing
- `cmd/version.go` - Version display command

### Core Internal Packages

#### `internal/config/`
**Purpose**: Configuration file parsing, project discovery, and local preferences management.
**Key Types**:
- `Config` - Main configuration structure mapping YAML to Go structs
- `Repo` - Repository configuration with git URL, path, default branch, and env_files
- `EnvFile` - Environment file configuration with source, destination, and variable replacements
- `Prompt` - Interactive prompt definition for collecting team preferences
- `PromptOption` - Individual option for a prompt (value and label)
- `LocalConfig` - Local user preferences stored in `.ramp/local.yaml`
- `Command` - Custom command definitions with name and script path

**Key Functions**:
- `FindRampProject(startDir)` - Recursively searches up directory tree for `.ramp/ramp.yaml`
- `LoadConfig(projectDir)` - Parses YAML configuration file and validates structure
- `SaveConfig(cfg, projectDir)` - Writes Config to ramp.yaml with custom formatting and spacing
- `LoadLocalConfig(projectDir)` - Loads local preferences from `.ramp/local.yaml`
- `SaveLocalConfig(localCfg, projectDir)` - Saves local preferences to `.ramp/local.yaml`
- `GetRepos()` - Returns map of repository name to configuration
- `GenerateEnvVarName(repoName)` - Converts repo names to valid environment variable names
- `HasPrompts()` - Checks if configuration defines interactive prompts
- `DetectFeatureFromWorkingDir(projectDir)` - Auto-detects feature name from current working directory by checking if inside `trees/<feature-name>/`

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

#### `internal/envfile/`
**Purpose**: Environment file copying and templating for feature worktrees, with support for executable scripts and caching.

**Key Functions**:
- `ProcessEnvFiles(repoName, envFiles, sourceDir, destDir, envVars, shouldRefresh)` - Processes all env_files for a repository
- `ProcessEnvFilesWithProjectDir(...)` - Internal version with explicit projectDir (useful for testing)
- `processEnvFile(...)` - Processes a single env file (file or script)
- `getContent(sourcePath, cacheTTL, envVars, shouldRefresh, projectDir)` - Gets content from file or script
- `isExecutable(info)` - Checks if file has execute permissions
- `executeScript(scriptPath, cacheTTL, envVars, shouldRefresh, projectDir)` - Executes script and caches output
- `checkCache(scriptPath, cacheTTL, projectDir)` - Checks if valid cache exists
- `cacheOutput(scriptPath, output, projectDir)` - Saves script output to cache
- `buildScriptEnv(envVars)` - Builds environment variables for script execution
- `replaceExplicitKeys(content, replacements, envVars)` - Performs explicit key replacements
- `replaceEnvVars(content, envVars)` - Expands ${VARIABLE_NAME} patterns

**How it works**:
1. **Source Detection**: Auto-detects whether source is a regular file or executable script
   - Regular files (no execute permission): Read directly
   - Executable scripts: Execute and capture stdout
2. **Script Execution** (if source is executable):
   - Passes all RAMP environment variables to script
   - Captures stdout as the env file content
   - Returns detailed error messages on script failure
3. **Caching** (optional, for scripts only):
   - If `cache` field specified (e.g., "24h", "1h", "30m"), caches script output
   - Cache stored in `.ramp/cache/env_files/<hash>.cache`
   - Cache key is SHA256 hash of script path
   - Respects `--refresh` and `--no-refresh` flags
   - Respects `auto_refresh` config per repository
4. **Variable Replacement** (after content retrieval):
   - If `replace` map specified: only replaces those keys
   - If no `replace` map: auto-replaces all ${RAMP_*} variables
   - Replacements happen on destination file (after script execution)
5. **Destination**: Writes final content to worktree destination

**Common Patterns**:
- **Static files**: Simple copy with variable replacement
- **Secret manager integration**: Script fetches secrets, output cached for TTL
- **Merge pattern**: Script combines .env.example + secrets, replacements applied after
- **Dynamic generation**: Script uses RAMP vars to generate environment-specific content

#### `internal/git/`
**Purpose**: Git operations and worktree management.

**Key Functions (with progress spinners - for standalone use):**
- `Clone(repoURL, destDir)` - Clones repositories with progress feedback
- `CreateWorktree(repoDir, worktreeDir, branchName)` - Intelligent worktree creation with branch detection
- `CreateWorktreeFromSource(repoDir, worktreeDir, branchName, sourceBranch, repoName)` - Creates worktree with new branch from specified source branch
- `RemoveWorktree(repoDir, worktreeDir)` - Removes worktree with force flag and progress spinner
- `DeleteBranch(repoDir, branchName)` - Deletes branch with force flag and progress spinner
- `Checkout(repoDir, branchName)` - Switches to existing local branch with progress spinner
- `CheckoutRemoteBranch(repoDir, branchName)` - Creates and switches to remote tracking branch with progress spinner
- `StashChanges(repoDir)` - Stashes uncommitted changes with progress spinner
- `PopStash(repoDir)` - Restores stashed changes with progress spinner
- `FetchBranch(repoDir, branchName)` - Fetches specific branch from remote with progress spinner
- `FetchAll(repoDir)` - Fetches all remotes with progress spinner
- `Pull(repoDir)` - Pulls changes with progress spinner
- `FetchPrune(repoDir)` - Prunes stale remote tracking branches with progress spinner

**Quiet Functions (without spinners - for use in loops with active parent spinner):**
- `CreateWorktreeQuiet()` - Worktree creation without spinner (use inside loops)
- `CreateWorktreeFromSourceQuiet()` - Worktree creation from source without spinner (use inside loops)
- `RemoveWorktreeQuiet()` - Worktree removal without spinner (use inside loops)
- `DeleteBranchQuiet()` - Branch deletion without spinner (use inside loops)
- `CheckoutQuiet()` - Branch checkout without spinner (use inside loops)
- `CheckoutRemoteBranchQuiet()` - Remote branch checkout without spinner (use inside loops)
- `StashChangesQuiet()` - Stash without spinner (use inside loops)
- `PopStashQuiet()` - Stash pop without spinner (use inside loops)
- `FetchBranchQuiet()` - Branch fetch without spinner (use inside loops)
- `FetchAllQuiet()` - Fetch all without spinner (use inside loops)
- `PullQuiet()` - Pull without spinner (use inside loops)
- `FetchPruneQuiet()` - Prune without spinner (use inside loops)

**Helper Functions (no spinners):**
- `LocalBranchExists()` / `RemoteBranchExists()` / `BranchExists()` - Branch existence checking with exact name matching
- `HasUncommittedChanges()` - Safety check using `git status --porcelain`
- `GetWorktreeBranch()` - Extracts actual branch name from worktree
- `GetCurrentBranch(repoDir)` - Gets current branch name in a repository
- `HasRemoteTrackingBranch()` - Detects if current branch tracks a remote
- `ResolveSourceBranch(repoDir, target, effectivePrefix)` - Resolves target to actual source branch (feature name, local branch, or remote branch)
- `GetRemoteTrackingStatus(repoDir)` - Gets ahead/behind commit status relative to remote tracking branch
- `IsGitRepo(dir)` - Checks if directory is a git repository
- `PruneWorktrees(repoDir)` - Prunes orphaned worktree registrations (no spinner)

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
- `RunCommandWithProgress()` - Executes shell commands with progress feedback, displaying output on completion
- `RunCommandWithProgressQuiet()` - Executes shell commands with progress feedback, hiding output on success (shows only on error)
- `OutputCapture` - Captures and conditionally displays command output
- Respects global `--verbose` flag to switch between spinner and direct output modes

**CRITICAL: Nested Spinner Anti-Pattern**

**NEVER create nested spinners** - this causes visual flashing and conflicts for terminal control.

**❌ BAD - Nested Spinner Anti-Pattern:**
```go
progress := ui.NewProgress()
progress.Start("Processing repositories")

for name, repo := range repos {
    // BAD: git.SomeOperation() creates its own spinner via ui.RunCommandWithProgress()
    // This creates a nested spinner while parent is still active!
    git.SomeOperation(repoDir)  // ⚡ CAUSES FLASHING
}

progress.Success("Processing complete")
```

**✅ GOOD - Use Quiet Versions in Loops:**
```go
progress := ui.NewProgress()
progress.Start("Processing repositories")

for name, repo := range repos {
    // GOOD: Use quiet version that runs git command without creating spinner
    git.SomeOperationQuiet(repoDir)  // ✓ No nested spinner
    progress.Update(fmt.Sprintf("Processed %s", name))  // Update existing spinner
}

progress.Success("Processing complete")
```

**Pattern to follow:**
1. **Create ONE spinner** at the start of an operation
2. **Use `Update()`** to change the message for each iteration
3. **Inside loops, ALWAYS use "Quiet" versions** of git operations:
   - `CreateWorktreeQuiet()` instead of `CreateWorktree()`
   - `RemoveWorktreeQuiet()` instead of `RemoveWorktree()`
   - `DeleteBranchQuiet()` instead of `DeleteBranch()`
   - `FetchPruneQuiet()` instead of `FetchPrune()`
   - `StashChangesQuiet()` instead of `StashChanges()`
   - `PopStashQuiet()` instead of `PopStash()`
   - `CheckoutQuiet()` instead of `Checkout()`
   - `CheckoutRemoteBranchQuiet()` instead of `CheckoutRemoteBranch()`
   - `FetchBranchQuiet()` instead of `FetchBranch()`
   - `FetchAllQuiet()` instead of `FetchAll()`
   - `PullQuiet()` instead of `Pull()`
4. **Complete with `Success()` or `Error()`** after the loop

**All git functions in `internal/git/` that call `ui.RunCommandWithProgress()` MUST have a corresponding "Quiet" version that calls `cmd.Run()` directly for use in loops.**

**When adding new git operations:**
1. Create both regular version (with progress) and Quiet version (without progress)
2. Regular version uses `ui.RunCommandWithProgress()` for standalone use
3. Quiet version uses `cmd.Run()` directly for use inside loops with active spinners

**Detection pattern:**
Look for: `for ... range repos` combined with git operations that might create spinners.
If found, ensure quiet versions are being used.

### Configuration Schema
Projects require a `.ramp/ramp.yaml` file with complete configuration:

```yaml
name: project-name                    # Display name for the project
repos:                               # Array of repository configurations
  - path: repos                      # Local directory path (relative to project root)
    git: git@github.com:owner/repo.git  # Git clone URL
    auto_refresh: true               # Optional: auto-refresh before 'ramp up' (default: true)
    env_files:                       # Optional: environment file copying and templating
      - .env.example                 # Simple: copy as-is
      - source: ../configs/app.env   # Advanced: copy with variable substitution
        dest: .env
        replace:
          PORT: "${RAMP_PORT}"
          APP_NAME: "myapp-${RAMP_WORKTREE_NAME}"
      - source: scripts/fetch-secrets.sh  # Script: execute and use output
        dest: .env.secrets
        cache: 24h                   # Cache script output for 24 hours
        replace:
          PORT: "${RAMP_PORT}"       # Replacements applied after script execution
  - path: repos
    git: https://github.com/owner/other-repo.git
    auto_refresh: true               # Optional: auto-refresh before 'ramp up' (default: true)

prompts:                             # Optional: interactive prompts for team preferences
  - name: RAMP_IDE                   # Environment variable name
    question: "Which IDE do you use?"
    options:
      - value: vscode
        label: Visual Studio Code
      - value: vim
        label: Vim/Neovim
    default: vscode                  # Must match an option value

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
├── .gitignore                       # Auto-generated by ramp init
├── .ramp/
│   ├── ramp.yaml                    # Main configuration file
│   ├── local.yaml                   # Local preferences (gitignored)
│   ├── port_allocations.json       # Auto-generated port tracking
│   └── scripts/                     # Optional setup/cleanup/command scripts
│       ├── setup.sh
│       ├── cleanup.sh
│       └── custom-command.sh
├── configs/                         # Optional: shared env file templates
│   ├── app.env
│   └── shared.env
├── repos/                           # Source repository clones (gitignored)
│   ├── repo-name/                   # Cloned repositories
│   └── other-repo/
└── trees/                           # Feature worktrees (gitignored)
    ├── feature-name/                # Individual feature directories
    │   ├── repo-name/               # Worktree for each repository
    │   └── other-repo/
    └── other-feature/
```

### Environment Variables for Scripts
All setup, cleanup, and custom command scripts receive these environment variables:

**Standard Variables:**
- `RAMP_PROJECT_DIR` - Absolute path to project root directory
- `RAMP_TREES_DIR` - Absolute path to current feature's trees directory
- `RAMP_WORKTREE_NAME` - Name of the current feature
- `RAMP_PORT` - Allocated port number for this feature (if port management enabled)
- `RAMP_REPO_PATH_<REPO_NAME>` - Absolute path to each repository's source directory

**Prompt Variables (if configured):**
- Custom environment variables from `prompts` configuration (e.g., `RAMP_IDE`, `RAMP_DATABASE`)
- Values stored in `.ramp/local.yaml` and loaded automatically
- Available in scripts and env_files templating

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

#### Git Stashes and Worktrees
**Important limitation**: Git stashes are **shared across all worktrees** of the same repository because they're stored in `.git/refs/stash` which is shared, not per-worktree.

**What this means**:
- When you create a stash in a feature worktree, it's visible in the source repository and all other worktrees
- Running `git stash pop` in the source repository might accidentally apply stashes created in feature worktrees
- This is a potential "footgun" that users should be aware of

**Best practice**:
- Use `git stash push -m "descriptive message"` to clearly identify stashes
- Check `git stash list` before popping to ensure you're applying the correct stash
- Consider using commits or branches instead of stashes for longer-lived work

## Environment Files with Script Execution

Ramp supports using executable scripts as sources for environment files, enabling integration with secret managers and dynamic configuration generation.

### Basic Concepts

**Source Types**:
- **Regular files**: Copied as-is (requires read permission only)
- **Executable scripts**: Executed, stdout captured as content (requires execute permission)

**Detection**: Automatic based on file execute bit (`chmod +x`)

**Caching**: Optional TTL-based caching for script outputs (reduces API calls to secret managers)

**Refresh Control**: Respects `--refresh`, `--no-refresh` flags and `auto_refresh` config

### Example 1: AWS Secrets Manager Integration

**Configuration** (`.ramp/ramp.yaml`):
```yaml
repos:
  - path: repos
    git: git@github.com:org/app.git
    env_files:
      - source: scripts/fetch-aws-secrets.sh
        dest: .env
        cache: 24h
        replace:
          PORT: "${RAMP_PORT}"
          APP_NAME: "myapp-${RAMP_WORKTREE_NAME}"
```

**Script** (`.ramp/scripts/fetch-aws-secrets.sh`):
```bash
#!/bin/bash
# Fetches secrets from AWS Secrets Manager
# Uses AWS CLI (assumes AWS credentials configured)

aws secretsmanager get-secret-value \
    --secret-id "myapp/${RAMP_ENV:-dev}-secrets" \
    --query SecretString \
    --output text
```

**How it works**:
1. Script executes on first `ramp up` (or when cache expires)
2. Output cached for 24 hours in `.ramp/cache/env_files/`
3. Cached content written to worktree's `.env`
4. `PORT` and `APP_NAME` replaced with dynamic values
5. Subsequent `ramp up` uses cache (no AWS call) unless `--refresh` specified

### Example 2: Merging .env.example with Secrets

**Problem**: You want non-sensitive defaults from `.env.example` plus sensitive keys from a secret manager.

**Configuration**:
```yaml
repos:
  - path: repos
    git: git@github.com:org/app.git
    env_files:
      - source: scripts/merge-env.sh
        dest: .env
        cache: 12h
        replace:
          PORT: "${RAMP_PORT}"
```

**Script** (`.ramp/scripts/merge-env.sh`):
```bash
#!/bin/bash
# Merge .env.example with secrets

# Start with non-sensitive defaults
cat "${RAMP_REPO_PATH_APP}/.env.example"

echo ""
echo "# Secrets from AWS"

# Append secrets (parsed from JSON)
aws secretsmanager get-secret-value \
    --secret-id "myapp/dev-secrets" \
    --query SecretString \
    --output text | \
    jq -r 'to_entries[] | "\(.key)=\(.value)"'
```

**Result** (worktree `.env`):
```
# From .env.example
DEBUG=true
LOG_LEVEL=info
PORT=4000  # Replaced by ramp

# Secrets from AWS
DATABASE_URL=postgres://...
API_KEY=sk_live_...
JWT_SECRET=...
```

### Example 3: Environment-Specific Configuration

**Use Case**: Different secrets per environment (dev/staging/prod)

**Configuration** (`.ramp/ramp.yaml`):
```yaml
prompts:
  - name: RAMP_ENV
    question: "Which environment?"
    options:
      - value: dev
        label: Development
      - value: staging
        label: Staging
      - value: prod
        label: Production
    default: dev

repos:
  - path: repos
    git: git@github.com:org/app.git
    env_files:
      - source: scripts/fetch-env-secrets.sh
        dest: .env
        cache: 1h  # Shorter cache for production
```

**Script** (`.ramp/scripts/fetch-env-secrets.sh`):
```bash
#!/bin/bash
# Uses RAMP_ENV to fetch environment-specific secrets

SECRET_ID="myapp/${RAMP_ENV}-secrets"

echo "Fetching secrets for: ${RAMP_ENV}" >&2  # Logged to stderr (not captured)

aws secretsmanager get-secret-value \
    --secret-id "$SECRET_ID" \
    --query SecretString \
    --output text
```

**Workflow**:
1. User runs `ramp config` (or prompted on first `ramp up`)
2. Selects environment (e.g., "dev")
3. Preference saved to `.ramp/local.yaml` (gitignored)
4. Script receives `RAMP_ENV=dev` and fetches appropriate secrets

### Example 4: Multiple Secret Sources

**Configuration**:
```yaml
repos:
  - path: repos
    git: git@github.com:org/app.git
    env_files:
      # Base configuration
      - source: .env.example
        dest: .env

      # Database credentials from Secrets Manager
      - source: scripts/fetch-db-secrets.sh
        dest: .env.database
        cache: 24h

      # API keys from Vault
      - source: scripts/fetch-api-keys.sh
        dest: .env.api
        cache: 12h
```

**Result**: Three separate env files in worktree, each sourced differently

### Refresh Behavior

**Cache Refresh Rules**:

| Flag | `cache` Field | Behavior |
|------|---------------|----------|
| `--refresh` | Any | Always execute script (ignore cache) |
| `--no-refresh` | Any | Use cache if exists (ignore TTL) |
| None | `24h` | Use cache if < 24h old, else execute |
| None | Not set | Always execute (no caching) |

**Examples**:
```bash
# Force refresh (re-fetch secrets)
ramp up my-feature --refresh

# Use cache even if expired (faster, but stale)
ramp up my-feature --no-refresh

# Respect auto_refresh config and cache TTL (default)
ramp up my-feature
```

### Security Best Practices

1. **Script Permissions**: Only make scripts executable if they should run
   ```bash
   chmod +x .ramp/scripts/fetch-secrets.sh  # Will execute
   chmod 644 .ramp/scripts/template.env     # Will copy as-is
   ```

2. **Never Commit Secrets**: Use scripts to fetch, not hardcode
   ```yaml
   # ✅ Good - fetches secrets at runtime
   - source: scripts/fetch-secrets.sh

   # ❌ Bad - secrets in git
   - source: .env.production
   ```

3. **Cache Location**: Add to `.gitignore`
   ```
   .ramp/cache/
   .ramp/local.yaml
   ```

4. **Authentication**: Scripts inherit shell environment
   - AWS CLI: Uses `~/.aws/credentials` or environment variables
   - Vault: Uses `VAULT_TOKEN` environment variable
   - GCP: Uses `gcloud auth` credentials

5. **Error Handling**: Scripts should exit with non-zero on failure
   ```bash
   #!/bin/bash
   set -e  # Exit on any error

   aws secretsmanager get-secret-value ... || {
       echo "Failed to fetch secrets" >&2
       exit 1
   }
   ```

### Troubleshooting

**Script not executing (copied as text)**:
```bash
# Check execute permission
ls -l .ramp/scripts/fetch-secrets.sh

# Should show: -rwxr-xr-x (note the 'x' bits)
# If not: chmod +x .ramp/scripts/fetch-secrets.sh
```

**Cache not working**:
- Check `.ramp/cache/env_files/` directory exists
- Verify `cache` field format (e.g., "24h", "1h", "30m")
- Use `--refresh` to force bypass cache

**Script fails silently**:
- Check script output: run manually
  ```bash
  cd .ramp/scripts
  ./fetch-secrets.sh
  ```
- Check stderr output (script errors shown to user)
- Verify RAMP environment variables available:
  ```bash
  #!/bin/bash
  echo "RAMP_PORT=${RAMP_PORT}" >&2
  echo "RAMP_ENV=${RAMP_ENV}" >&2
  ```

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
- **cmd package**: 61.2% (covering main commands)
- **internal/config**: 95.7% (configuration management)
- **internal/git**: 75.0% (git operations)
- **internal/scaffold**: 88.9% (project scaffolding)
- **internal/ports**: 74.2% (port allocation)
- **internal/ui**: 77.6% (UI and progress feedback)
- **Total**: 362 tests across all packages

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
2. **Coverage Gaps**: ~39% of cmd code paths not yet covered
3. **Performance**: No benchmark tests for performance-critical operations

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