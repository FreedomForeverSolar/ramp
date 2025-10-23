# Ramp

A sophisticated CLI tool for managing multi-repository development workflows using git worktrees and automated setup scripts.

## Overview

**Ramp** enables developers to work on features that span multiple repositories simultaneously by creating isolated working directories, complete with custom setup scripts, port management, and cleanup automation. Instead of manually managing multiple git repositories, switching branches, and coordinating development environments, Ramp automates the entire workflow.

### Why Ramp?

**Problem**: Modern applications often consist of multiple repositories (microservices, frontend/backend, libraries). Developing features that span multiple repos requires:
- Cloning and managing multiple repositories
- Creating and switching feature branches across repos
- Coordinating development environments and ports
- Managing dependencies between services
- Cleaning up when features are complete

**Solution**: Ramp automates all of this with a single command. Create feature branches across all repositories, set up development environments, allocate ports, and clean everything up when done.

### Key Benefits

- 🚀 **One-command setup**: `ramp up feature-name` creates branches and sets up environment across all repos
- 🔄 **Git worktrees**: Work on multiple features simultaneously without branch switching
- 🎯 **Port management**: Automatic port allocation (one per feature) prevents conflicts
- 📦 **Environment automation**: Custom scripts handle dependencies, databases, and service startup
- 🧹 **Automatic cleanup**: `ramp down` removes individual features, `ramp prune` batch-removes all merged features
- 💾 **State persistence**: Projects remember configuration and active features across sessions

## Installation

### Prerequisites

- **Go** 1.21+ (for building from source)
- **Git** 2.25+ (for worktree support)
- **Node.js** (optional, for demo project)

### Homebrew (macOS/Linux)

Install Ramp using Homebrew:

```bash
brew install freedomforeversolar/tools/ramp
```

Verify installation:
```bash
ramp --help
```

**Upgrading:**
```bash
brew update                                  # Update Homebrew and all taps
brew upgrade freedomforeversolar/tools/ramp  # Upgrade ramp to latest version
```

### Download Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/FreedomForeverSolar/ramp/releases):

1. Download the appropriate archive for your OS and architecture
2. Extract the binary
3. Move it to a directory in your PATH (e.g., `/usr/local/bin`)
4. Make it executable: `chmod +x ramp`

### Build from Source

1. **Clone the repository:**
   ```bash
   git clone https://github.com/FreedomForeverSolar/ramp.git
   cd ramp
   ```

2. **Build the binary:**
   ```bash
   go build -o ramp .
   ```

3. **Install globally (optional):**
   ```bash
   sudo ./install.sh
   ```
   This installs to `/usr/local/bin/ramp`

4. **Verify installation:**
   ```bash
   ./ramp --help  # If running locally
   # OR
   ramp --help    # If installed globally
   ```

### Alternative: Local Development

For development or testing, run without building:
```bash
go run . --help
```

## Quick Start

### Try the Demo

The easiest way to understand Ramp is to run the included demo:

```bash
cd demo/demo-microservices-app
ramp install                # Clone demo repositories (config already exists)
ramp up my-feature          # Create feature branch across all repos
ramp run dev                # Start simulated development environment
ramp status                 # View project status
ramp down my-feature        # Clean up single feature
ramp prune                  # Or batch-remove all merged features
```

See [demo/demo-microservices-app/README.md](demo/demo-microservices-app/README.md) for detailed walkthrough.

### Create Your Own Project

1. **Initialize a new Ramp project:**
   ```bash
   mkdir my-project && cd my-project
   ramp init
   ```

   This interactive setup will guide you through:
   - Project name (defaults to directory name)
   - Branch prefix (defaults to "feature/")
   - Repository URLs (add as many as needed)
   - Setup/cleanup scripts (optional, defaults to Yes)
   - Port management (optional)
   - Custom commands like 'doctor' for environment checks

2. **Customize your scripts:**
   Edit the generated files in `.ramp/scripts/` to match your workflow:
   - `setup.sh` - Runs after creating feature branches
   - `cleanup.sh` - Runs before tearing down features
   - `doctor.sh` - Environment validation (if added)

3. **Install repositories:**
   ```bash
   ramp install  # Clone all configured repositories
   ```
   (This happens automatically if you chose "Yes" during init)

4. **Start working on features:**
   ```bash
   ramp up new-feature     # Create feature branches
   ramp run dev            # Run custom development command (if configured)
   ramp status             # View all active features and their merge status
   ramp down new-feature   # Clean up individual feature when done
   ramp prune              # Or batch-remove all merged features
   ```

## Commands Reference

### Core Commands

#### `ramp init`
Initialize a new ramp project with interactive setup (similar to `npm init`).
```bash
ramp init       # Interactive project scaffolding
ramp init -v    # Verbose output
```

Creates `.ramp/ramp.yaml`, directory structure (`repos/`, `trees/`), and optional setup scripts.
After initialization, run `ramp install` to clone repositories.

**Flags:**
- `-v, --verbose`: Show detailed output

#### `ramp install`
Clone all configured repositories from ramp.yaml.
```bash
ramp install
ramp install -v    # Verbose output showing clone operations
```

**Flags:**
- `-v, --verbose`: Show detailed command output instead of progress spinners

#### `ramp up <feature-name>`
Create feature branch with worktrees across all repositories. Automatically refreshes repositories that have `auto_refresh` enabled (defaults to true).
```bash
ramp up user-auth-feature
ramp up urgent-fix --prefix hotfix/           # Custom branch prefix
ramp up new-feature --target existing-feature # Create from existing feature
ramp up new-feature --target feature/my-branch # Create from specific branch
ramp up new-feature --target origin/main      # Create from remote branch
ramp up my-feature -v                         # Verbose output showing all commands
```

**Flags:**
- `--prefix <prefix>`: Override the branch prefix from config (e.g., `--prefix hotfix/`)
- `--target <target>`: Create feature from existing feature name, local branch, or remote branch
- `-v, --verbose`: Show detailed command output instead of progress spinners

#### `ramp down <feature-name>`
Clean up feature branch, worktrees, and allocated resources.
```bash
ramp down user-auth-feature  # Prompts for confirmation if uncommitted changes
ramp down -v my-feature      # Verbose output showing cleanup steps
```

**Flags:**
- `-v, --verbose`: Show detailed command output instead of progress spinners

#### `ramp prune`
Automatically clean up all merged feature branches in one command.
```bash
ramp prune        # Shows summary, asks for confirmation, then removes all merged features
ramp prune -v     # Verbose output showing detailed cleanup operations
```

Scans all features in the `trees/` directory, identifies features that have been merged into their default branch (using `git merge-base`), and removes them after confirmation. Features categorized as "CLEAN" (never had any commits) are excluded from pruning.

**What gets removed:**
- Git worktrees for each repository
- Local feature branches
- Allocated port numbers
- Feature directories in `trees/`

**Behavior:**
- Shows summary of all merged features before proceeding
- Asks for single confirmation to remove all
- Continues with remaining features if one fails (non-blocking errors)
- Displays final summary with success count and any failures

**Flags:**
- `-v, --verbose`: Show detailed command output instead of progress spinners

#### `ramp status`
Show comprehensive project status, including active features.
```bash
ramp status
ramp status -v      # Verbose output with additional repository details
```

**Flags:**
- `-v, --verbose`: Show detailed command output instead of progress spinners

### Repository Management

#### `ramp refresh`
Update all source repositories by pulling from remotes.
```bash
ramp refresh
ramp refresh -v     # Verbose output showing git operations
```

**Flags:**
- `-v, --verbose`: Show detailed command output instead of progress spinners

#### `ramp rebase <branch-name>`
Switch all repositories to specified branch.
```bash
ramp rebase develop        # Switch all repos to develop branch
ramp rebase feature/shared # Switch to shared feature branch
ramp rebase -v main       # Verbose output showing branch switching
```

**Flags:**
- `-v, --verbose`: Show detailed command output instead of progress spinners

### Custom Commands

#### `ramp run <command> [feature]`
Execute custom commands defined in configuration.
```bash
ramp run dev               # Auto-detect current feature
ramp run dev my-feature    # Specify feature explicitly
ramp run test my-feature   # Run custom test command
ramp run -v dev my-feature # Verbose output showing script execution
```

**Flags:**
- `-v, --verbose`: Show detailed command output instead of progress spinners

### Global Options

- `-v, --verbose`: Show detailed output and disable progress spinners
- `-h, --help`: Show help information

## Configuration

### Project Configuration (`.ramp/ramp.yaml`)

```yaml
# Project name (displayed in status)
name: my-project

# Repository configurations
repos:
  - path: repos                 # Local directory name
    git: git@github.com:org/frontend.git  # Git clone URL
    auto_refresh: true          # Optional: auto-refresh before 'ramp up' (default: true)

  - path: repos
    git: https://github.com/org/api.git
    auto_refresh: true          # Optional: auto-refresh before 'ramp up' (default: true)

# Optional: Scripts to run during lifecycle events
setup: scripts/setup.sh       # After 'ramp up'
cleanup: scripts/cleanup.sh   # Before 'ramp down'

# Optional: Branch naming
default-branch-prefix: feature/  # Prefix for new branches

# Optional: Port management (allocates ONE port per feature)
base_port: 3000              # Starting port number
max_ports: 100              # Maximum ports to allocate

# Optional: Custom commands
commands:
  - name: dev                # 'ramp run dev'
    command: scripts/dev.sh
  - name: test
    command: scripts/test.sh
  - name: deploy
    command: scripts/deploy.sh
```

### Environment Variables for Scripts

All scripts receive these environment variables:

- `RAMP_PROJECT_DIR`: Absolute path to project root
- `RAMP_TREES_DIR`: Path to current feature's trees directory
- `RAMP_WORKTREE_NAME`: Feature name
- `RAMP_PORT`: Allocated port number (if port management enabled)
- `RAMP_REPO_PATH_<REPO>`: Path to each repository's source directory

Repository names are converted to valid environment variable names (uppercase, underscores for non-alphanumeric characters).

### Directory Structure

```
my-project/
├── .ramp/
│   ├── ramp.yaml                    # Main configuration
│   ├── port_allocations.json       # Auto-generated port tracking
│   └── scripts/                     # Custom scripts
│       ├── setup.sh
│       ├── cleanup.sh
│       └── dev.sh
├── repos/                           # Source repository clones
│   ├── frontend/                    # Cloned repositories
│   └── api/
└── trees/                           # Feature worktrees
    ├── feature-a/                   # Individual feature directories
    │   ├── frontend/                # Worktree for each repository
    │   └── api/
    └── feature-b/
        ├── frontend/
        └── api/
```

## Advanced Features

### Smart Branch Handling

Ramp intelligently handles various branch scenarios:

- **New branches**: Creates from default branch (or specified target with `--target`)
- **Existing local branches**: Uses without modification
- **Remote-only branches**: Creates local tracking branch
- **Target branching**: When using `--target`, creates new branches from specified source; gracefully falls back to default behavior if target doesn't exist in some repositories
- **Conflicting names**: Provides clear error messages

### Safety Mechanisms

- **Uncommitted changes**: Warns before destructive operations
- **Atomic operations**: Rolls back on failure during multi-repo operations
- **Port conflicts**: Automatically allocates available ports
- **Missing repositories**: Auto-initializes on first use

### Port Management

**Important**: Ramp allocates **exactly one port per feature** from the configured range.

- Each feature gets one unique port (e.g., 3000, 3001, 3002...)
- Persistent across sessions in `.ramp/port_allocations.json`
- Released automatically on cleanup
- Available to scripts via `RAMP_PORT` environment variable

**Multi-Service Strategy**: If your project needs multiple ports per feature, implement a port range strategy in your setup scripts:
- Use `RAMP_PORT` as base (e.g., 3000)
- Append digits for additional services (e.g., 30001, 30002)
- This ensures different features don't conflict

### Progress Feedback

- **Normal mode**: Animated progress spinners with status updates
- **Verbose mode** (`-v`): Full command output for debugging

## Use Cases

### Microservices Development
Coordinate feature development across multiple microservices with shared databases and networking.

### Frontend/Backend Projects
Develop full-stack features that require changes to both frontend and backend simultaneously.

### Library Development
Work on libraries alongside applications that consume them, with live linking during development.

### Multi-Environment Testing
Set up isolated environments for testing features without affecting main development.

## Troubleshooting

### Common Issues

**Q: "ramp: command not found"**
A: Either run `./ramp` from the project directory or install globally with `sudo ./install.sh`

**Q: "No .ramp/ramp.yaml found"**
A: Run commands from a directory containing `.ramp/ramp.yaml` or a subdirectory

**Q: "Port already in use"**
A: Ramp automatically allocates available ports. Check `ramp status` to see allocations

**Q: Git worktree errors**
A: Ensure Git 2.25+ is installed and repositories are properly initialized

**Q: Permission denied on scripts**
A: Make scripts executable: `chmod +x .ramp/scripts/*.sh`

### Debug Mode

Run with verbose flag to see detailed output:
```bash
ramp -v up my-feature
ramp -v status
```

### Batch Cleanup

To clean up multiple merged features at once:
```bash
ramp prune        # Removes all merged features after confirmation
ramp status       # Verify cleanup completed successfully
```

### Manual Cleanup

If Ramp cleanup fails, manually remove:
```bash
# Remove worktrees
git worktree remove trees/feature-name/repo-name --force

# Delete branches
git branch -D feature/feature-name

# Remove port allocations
rm .ramp/port_allocations.json
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o ramp .
```

### Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run tests: `go test ./...`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: Report bugs and request features via GitHub issues
- **Discussions**: Ask questions and share use cases in GitHub discussions
- **Documentation**: Additional examples and guides in the [docs/](docs/) directory

---

**Get started now**: Try the demo in `demo/demo-microservices-app/` to see Ramp in action!