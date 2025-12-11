# CLAUDE.md

AI assistant guidance for working with the Ramp CLI codebase.

## Project Overview

Ramp is a CLI tool for managing multi-repository development workflows using git worktrees. It enables developers to work on features spanning multiple repositories simultaneously by creating isolated working directories with automated setup scripts, port management, and cleanup.

## Quick Reference

### Build & Test
- `go build -o ramp .` - Build CLI binary
- `./install.sh` - Build and install CLI to `/usr/local/bin`
- `go test ./...` - Run all tests

### Desktop App (ramp-ui)
- `go build -o ramp-ui/frontend/resources/ramp-server ./cmd/ramp-ui` - Build backend
- `cd ramp-ui/frontend && bun run dev` - Start dev mode with hot reload
- `cd ramp-ui/frontend && bun run build && bun run package` - Build distributable

### Key Commands
- `ramp init` - Interactive project setup (uses huh forms library)
- `ramp install` - Clone all configured repositories
- `ramp up <feature>` - Create feature worktrees across all repos
- `ramp down <feature>` - Clean up feature worktrees and branches
- `ramp config` - Manage local preferences
- `ramp status` - Show project and worktree status
- `ramp refresh` - Update all source repositories
- `ramp prune` - Clean up merged features
- `ramp run <cmd>` - Execute custom commands

For detailed usage, see README or use `--help` flag.

## Architecture

### Project Structure
```
cmd/              # Cobra CLI commands (root.go, up.go, down.go, etc.)
cmd/ramp-ui/      # HTTP server entry point for desktop app
internal/
  config/         # YAML parsing, project discovery
  git/            # Git operations and worktree management
  scaffold/       # Project initialization templates
  envfile/        # Environment file processing
  ports/          # Port allocation management
  ui/             # Progress spinners and feedback
  autoupdate/     # Homebrew auto-update system
  operations/     # Shared operation logic (up, down, refresh, install)
  uiapi/          # REST API handlers for desktop app
ramp-ui/          # Electron + React desktop app (see ramp-ui/README.md)
```

### Configuration
Projects use `.ramp/ramp.yaml`:
```yaml
name: project-name
repos:
  - path: repos
    git: git@github.com:owner/repo.git
    auto_refresh: true  # Auto-refresh before 'ramp up' (default: true)
    env_files:          # Optional: copy/template env files
      - .env.example
      - source: scripts/fetch-secrets.sh
        dest: .env
        cache: 24h      # Cache script output
setup: scripts/setup.sh     # Optional
cleanup: scripts/cleanup.sh # Optional
default-branch-prefix: feature/
base_port: 3000            # Optional port management
commands:                  # Custom commands for 'ramp run'
  - name: open
    command: scripts/open.sh
```

### Directory Layout
```
.ramp/
  ├── ramp.yaml              # Main config
  ├── local.yaml             # Local preferences (gitignored)
  └── scripts/               # Setup/cleanup scripts
repos/                       # Source clones (gitignored)
trees/                       # Feature worktrees (gitignored)
  └── feature-name/
      ├── repo1/
      └── repo2/
```

## Critical Patterns

### Nested Spinner Anti-Pattern

**NEVER create nested spinners** - causes visual flashing and terminal conflicts.

❌ **BAD:**
```go
progress := ui.NewProgress()
progress.Start("Processing repos")
for name, repo := range repos {
    git.CreateWorktree(...)  // Creates its own spinner!
}
progress.Success("Done")
```

✅ **GOOD:**
```go
progress := ui.NewProgress()
progress.Start("Processing repos")
for name, repo := range repos {
    git.CreateWorktreeQuiet(...)  // No spinner
    progress.Update(fmt.Sprintf("Processed %s", name))
}
progress.Success("Done")
```

**Rule:** Inside loops with an active spinner, ALWAYS use "Quiet" versions of git operations:
- `CreateWorktreeQuiet()`, `RemoveWorktreeQuiet()`, `DeleteBranchQuiet()`, etc.
- All git functions that use `ui.RunCommandWithProgress()` must have a `Quiet` variant

## Key Packages

### `internal/config/`
Configuration management and project discovery.
- `Config`, `Repo`, `EnvFile`, `Prompt`, `LocalConfig` types
- `FindRampProject()` - Recursively searches for `.ramp/ramp.yaml`
- `LoadConfig()`, `SaveConfig()` - YAML persistence
- `DetectFeatureFromWorkingDir()` - Auto-detect current feature

### `internal/git/`
Git operations with two variants for each operation:
- **Regular** (with spinner): `CreateWorktree()`, `RemoveWorktree()`, etc.
- **Quiet** (no spinner): `CreateWorktreeQuiet()`, `RemoveWorktreeQuiet()`, etc.
- **Helpers**: `BranchExists()`, `HasUncommittedChanges()`, `GetCurrentBranch()`, etc.

### `internal/envfile/`
Environment file processing with script execution support:
- Detects executable scripts vs regular files (via execute bit)
- Executes scripts and captures stdout as env file content
- Optional caching with TTL (e.g., `cache: 24h`)
- Variable replacement: `${RAMP_PORT}`, `${RAMP_WORKTREE_NAME}`, etc.

### `internal/ui/`
Progress feedback respecting `--verbose` flag:
- `NewProgress()`, `Start()`, `Success()`, `Error()`, `Warning()`
- `RunCommandWithProgress()` - Executes commands with spinner
- `RunCommandWithProgressQuiet()` - Executes without showing output on success

### `internal/operations/`
Shared operation logic used by both CLI and desktop app:
- `ProgressReporter` interface - Abstracts CLI spinners vs WebSocket broadcasting
- `OutputStreamer` interface - Line-by-line command output streaming
- `ConfirmationHandler` interface - User confirmation abstraction
- `Up()`, `Down()`, `Refresh()`, `Install()`, `Run()` - Core operations accepting `ProgressReporter`

### `internal/uiapi/`
REST API and WebSocket handlers for the desktop app:
- `Server` struct - Manages WebSocket connections, routes, and per-project locks
- `projects.go` - Project CRUD, reorder, favorites
- `features.go` - Feature create/delete/prune (ramp up/down/prune)
- `commands.go` - Custom command execution endpoints
- `source_repos.go` - Source repository status and refresh
- `terminal.go` - Open terminal at project/feature paths
- `websocket.go` - Real-time progress updates via WebSocket
- `appconfig.go` - Persistent app configuration and settings
- `config.go` - Project-level local preferences (prompts)
- `models.go` - API request/response types

## Environment Variables

Scripts receive these variables:
- `RAMP_PROJECT_DIR` - Project root
- `RAMP_TREES_DIR` - Feature trees directory
- `RAMP_WORKTREE_NAME` - Feature name
- `RAMP_PORT` - Allocated port (if configured)
- `RAMP_REPO_PATH_<REPO>` - Path to each repo (uppercase, underscores)
- Custom variables from `prompts` configuration

## Testing

Run tests with `go test ./...` or `go test ./... -cover`.

**Test Helpers:**
- `NewTestProject(t)` - Creates isolated test project
- `tp.InitRepo("name")` - Creates repo with bare remote
- `runGitCmd(t, dir, args...)` - Executes git commands

**Testing Pattern:**
- Uses real git operations (no mocking)
- Table-driven tests with subtests
- Tests both success and failure paths
- Each test gets isolated temp directories

## Important Behaviors

- **Auto-installation**: Most commands auto-run `ramp install` if repos not cloned
- **Auto-refresh**: Repos with `auto_refresh: true` (default) refresh before `ramp up`
- **Smart branching**: Intelligently handles local/remote/new branches
- **Safety checks**: `ramp down` warns about uncommitted changes
- **Port management**: Unique ports allocated per feature (persisted in `.ramp/port_allocations.json`)
- **Git stash caveat**: Stashes are shared across all worktrees of the same repo

## Auto-Update System

Homebrew installs get automatic background updates:
- Spawns detached background process on every command
- Checks `~/.ramp/settings.yaml` for config (default: `check_interval: 12h`)
- Uses file locking to prevent concurrent updates
- Manual installs (non-Homebrew) auto-disable updates

For more details, see source code or run commands with `--help`.
