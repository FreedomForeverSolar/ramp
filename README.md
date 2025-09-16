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
- 🧹 **Automatic cleanup**: `ramp down` removes all traces of feature branches and environments
- 💾 **State persistence**: Projects remember configuration and active features across sessions

## Installation

### Prerequisites

- **Go** 1.21+ (for building from source)
- **Git** 2.25+ (for worktree support)
- **Node.js** (optional, for demo project)

### Build from Source

1. **Clone the repository:**
   ```bash
   git clone [repository-url]
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
ramp init                    # Clone demo repositories
ramp up my-feature          # Create feature branch across all repos
ramp run dev                # Start simulated development environment
ramp status                 # View project status
ramp down my-feature        # Clean up everything
```

See [demo/demo-microservices-app/README.md](demo/demo-microservices-app/README.md) for detailed walkthrough.

### Create Your Own Project

1. **Create project structure:**
   ```bash
   mkdir my-project && cd my-project
   mkdir -p .ramp/scripts
   ```

2. **Create configuration (`.ramp/ramp.yaml`):**
   ```yaml
   name: my-project                          # Display name for the project
   repos:
     - path: frontend                        # Local directory name
       git: git@github.com:yourorg/frontend.git  # Git clone URL
       auto_refresh: true                    # Optional: auto-refresh before 'ramp up' (default: true)
     - path: backend                         # Local directory name
       git: git@github.com:yourorg/backend.git   # Git clone URL
       auto_refresh: false                   # Optional: disable auto-refresh for this repo

   setup: scripts/setup.sh                  # Optional: script to run after 'ramp up'
   cleanup: scripts/cleanup.sh              # Optional: script to run before 'ramp down'
   default-branch-prefix: feature/          # Optional: prefix for new branch names
   base_port: 3000                          # Optional: starting port for allocation (default: 3000)
   max_ports: 50                            # Optional: maximum ports to allocate (default: 100)

   commands:                                # Optional: custom commands for 'ramp run'
     - name: dev                            # Command name for 'ramp run dev'
       command: scripts/dev.sh              # Script path (relative to .ramp/)
   ```

3. **Create setup script (`.ramp/scripts/setup.sh`):**
   ```bash
   #!/bin/bash
   echo "Setting up $RAMP_WORKTREE_NAME on port $RAMP_PORT"
   # Install dependencies, create config files, etc.
   ```

4. **Use your project:**
   ```bash
   ramp init                # Clone all repositories
   ramp up new-feature     # Create feature branches
   ramp run dev            # Run custom development command
   ramp down new-feature   # Clean up when done
   ```

## Commands Reference

### Core Commands

#### `ramp init`
Initialize project by cloning all configured repositories.
```bash
ramp init
ramp init -v    # Verbose output showing clone operations
```

**Flags:**
- `-v, --verbose`: Show detailed command output instead of progress spinners

#### `ramp up <feature-name>`
Create feature branch with worktrees across all repositories. Automatically refreshes repositories that have `auto_refresh` enabled (defaults to true).
```bash
ramp up user-auth-feature
ramp up --prefix hotfix/ urgent-fix           # Custom branch prefix
ramp up --target existing-feature new-feature # Create from existing feature
ramp up --target feature/my-branch new-feature # Create from specific branch
ramp up --target origin/main new-feature      # Create from remote branch
ramp up -v my-feature                         # Verbose output showing all commands
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
  - path: frontend              # Local directory name
    git: git@github.com:org/frontend.git  # Git clone URL
    auto_refresh: true          # Optional: auto-refresh before 'ramp up' (default: true)

  - path: api
    git: https://github.com/org/api.git
    auto_refresh: false         # Optional: disable auto-refresh for this repo

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
├── source/                          # Source repository clones
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

[License information would go here]

## Support

- **Issues**: Report bugs and request features via GitHub issues
- **Discussions**: Ask questions and share use cases in GitHub discussions
- **Documentation**: Additional examples and guides in the [docs/](docs/) directory

---

**Get started now**: Try the demo in `demo/demo-microservices-app/` to see Ramp in action!