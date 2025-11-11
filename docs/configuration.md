# Configuration Reference

This document provides a complete reference for the `.ramp/ramp.yaml` configuration file.

## Complete Example

```yaml
# Project name (displayed in status)
name: my-project

# Repository configurations
repos:
  - path: repos
    git: git@github.com:org/frontend.git
    auto_refresh: true
  - path: repos
    git: https://github.com/org/api.git
    auto_refresh: true
  - path: repos
    git: git@github.com:org/shared-library.git
    auto_refresh: false

# Optional: Scripts to run during lifecycle events
setup: scripts/setup.sh
cleanup: scripts/cleanup.sh

# Optional: Branch naming
default-branch-prefix: feature/

# Optional: Port management (one port per feature)
base_port: 3000
max_ports: 100

# Optional: Custom commands
commands:
  - name: dev
    command: scripts/dev.sh
  - name: test
    command: scripts/test.sh
  - name: deploy
    command: scripts/deploy.sh
  - name: doctor
    command: scripts/doctor.sh
```

## Configuration Fields

### `name` (required)

The display name for your project. Used in status output and messages.

```yaml
name: my-awesome-project
```

### `repos` (required)

Array of repository configurations. Each repository must have:

#### `path` (required)

The local directory where repositories will be cloned. Typically `repos`.

```yaml
repos:
  - path: repos
```

#### `git` (required)

The git clone URL. Supports both SSH and HTTPS:

```yaml
repos:
  - git: git@github.com:org/repo.git           # SSH
  - git: https://github.com/org/repo.git       # HTTPS
  - git: git@gitlab.com:org/repo.git           # GitLab
  - git: https://bitbucket.org/org/repo.git    # Bitbucket
```

#### `auto_refresh` (optional)

Whether to automatically fetch and pull this repository before `ramp up`. Defaults to `true` if not specified.

```yaml
repos:
  - path: repos
    git: git@github.com:org/frontend.git
    auto_refresh: true   # Auto-refresh before 'ramp up'

  - path: repos
    git: git@github.com:org/legacy.git
    auto_refresh: false  # Skip auto-refresh for this repo
```

**Why disable auto_refresh?**
- Large repositories that take time to fetch
- Rarely-changing repositories
- Repositories where you want manual control over updates

You can override this setting per-command:
```bash
ramp up my-feature --refresh      # Force refresh all repos
ramp up my-feature --no-refresh   # Skip refresh for all repos
```

### `setup` (optional)

Path to script that runs after `ramp up` creates a new feature. Relative to `.ramp/` directory.

```yaml
setup: scripts/setup.sh
```

Use for:
- Installing dependencies
- Starting databases
- Initializing development environment
- Creating symlinks

See [Custom Scripts Guide](guides/custom-scripts.md) for details.

### `cleanup` (optional)

Path to script that runs before `ramp down` removes a feature. Relative to `.ramp/` directory.

```yaml
cleanup: scripts/cleanup.sh
```

Use for:
- Stopping services
- Cleaning up temporary files
- Backing up data
- Resetting state

### `default-branch-prefix` (optional)

Prefix for new branch names. Defaults to `feature/` if not specified.

```yaml
default-branch-prefix: feature/
```

Examples:
- `feature/` → `feature/my-branch`
- `dev/` → `dev/my-branch`
- `""` (empty) → `my-branch`

Override per-command:
```bash
ramp up my-branch --prefix hotfix/   # hotfix/my-branch
ramp up my-branch --no-prefix        # my-branch
```

### `base_port` (optional)

Starting port number for allocation. Defaults to `3000` if not specified.

```yaml
base_port: 3000
```

**Important**: Ramp allocates **one port per feature**, not per repository.

### `max_ports` (optional)

Maximum number of ports to allocate. Defaults to `100` if not specified.

```yaml
max_ports: 100
```

This creates a port range from `base_port` to `base_port + max_ports - 1`.

Example with `base_port: 3000` and `max_ports: 100`:
- First feature: port 3000
- Second feature: port 3001
- Last available: port 3099

See [Port Management Guide](advanced/port-management.md) for multi-service strategies.

### `commands` (optional)

Custom commands for `ramp run`. Each command has:

#### `name` (required)

Command name used with `ramp run <name>`.

```yaml
commands:
  - name: dev       # Run with: ramp run dev
```

#### `command` (required)

Path to script file. Relative to `.ramp/` directory.

```yaml
commands:
  - name: dev
    command: scripts/dev.sh
```

Example custom commands:
```yaml
commands:
  - name: dev
    command: scripts/dev.sh           # Start dev servers
  - name: test
    command: scripts/test.sh          # Run tests
  - name: deploy
    command: scripts/deploy.sh        # Deploy feature
  - name: doctor
    command: scripts/doctor.sh        # Check environment
  - name: open
    command: scripts/open.sh          # Open in browser/editor
```

## Environment Variables

All scripts (setup, cleanup, custom commands) receive these environment variables:

### Standard Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `RAMP_PROJECT_DIR` | Absolute path to project root | `/home/user/my-project` |
| `RAMP_TREES_DIR` | Path to feature's trees directory | `/home/user/my-project/trees/my-feature` |
| `RAMP_WORKTREE_NAME` | Feature name | `my-feature` |
| `RAMP_PORT` | Allocated port number | `3000` |

### Repository Path Variables

For each repository, a variable is created with the pattern `RAMP_REPO_PATH_<REPO_NAME>`:

```yaml
repos:
  - git: git@github.com:org/frontend.git      # RAMP_REPO_PATH_FRONTEND
  - git: git@github.com:org/api-server.git    # RAMP_REPO_PATH_API_SERVER
  - git: git@github.com:org/shared-lib.git    # RAMP_REPO_PATH_SHARED_LIB
```

Repository names are converted to valid environment variable names:
1. Extract name from git URL (last path segment before `.git`)
2. Convert to uppercase
3. Replace non-alphanumeric characters with underscores
4. Remove consecutive underscores

### Using in Scripts

```bash
#!/bin/bash
# .ramp/scripts/setup.sh

echo "Setting up feature: $RAMP_WORKTREE_NAME"
echo "Port: $RAMP_PORT"

# Install frontend dependencies
cd "$RAMP_TREES_DIR/frontend"
npm install

# Install API dependencies
cd "$RAMP_REPO_PATH_API_SERVER"
go mod download

# Start database on feature-specific port
docker run -p "$RAMP_PORT:5432" postgres
```

## Directory Structure

```
my-project/
├── .ramp/
│   ├── ramp.yaml                # This configuration file
│   ├── port_allocations.json    # Auto-generated (DO NOT EDIT)
│   └── scripts/                 # Your scripts
│       ├── setup.sh
│       ├── cleanup.sh
│       ├── dev.sh
│       ├── test.sh
│       └── doctor.sh
├── repos/                       # Source repositories (path from config)
│   ├── frontend/
│   ├── api-server/
│   └── shared-lib/
└── trees/                       # Feature worktrees
    ├── my-feature/
    │   ├── frontend/
    │   ├── api-server/
    │   └── shared-lib/
    └── other-feature/
        ├── frontend/
        ├── api-server/
        └── shared-lib/
```

## Best Practices

### Repository Configuration

- Use SSH URLs for private repositories (avoid password prompts)
- Set `auto_refresh: false` for large/slow repositories
- Keep all repos in the same `path` directory for simplicity

### Scripts

- Make scripts executable: `chmod +x .ramp/scripts/*.sh`
- Add error handling and validation
- Use absolute paths from environment variables
- Log operations for debugging

### Port Management

- Choose `base_port` that doesn't conflict with common services
- Set `max_ports` based on team size and feature count
- Document port allocation strategy in your scripts

### Branch Naming

- Use consistent prefixes (`feature/`, `bugfix/`, `hotfix/`)
- Keep feature names short and descriptive
- Use kebab-case for feature names

## Migration

### Adding auto_refresh to Existing Config

If your `ramp.yaml` doesn't have `auto_refresh` settings, they default to `true`. To disable for specific repos:

```yaml
repos:
  - path: repos
    git: git@github.com:org/repo.git
    auto_refresh: false  # Add this line
```

### Changing Port Range

Edit `base_port` and `max_ports`, then:

```bash
rm .ramp/port_allocations.json  # Reset allocations
ramp status                     # Regenerate on next command
```

## Next Steps

- [Custom Scripts Guide](guides/custom-scripts.md) - Write powerful automation
- [Getting Started](getting-started.md) - Create your first project
- [Command Reference](commands/ramp.md) - Explore all commands
