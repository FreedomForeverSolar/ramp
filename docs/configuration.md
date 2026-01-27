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
    env_files:
      - .env.example                        # Simple: copy as-is
      - source: ../configs/frontend.env     # Advanced: copy with templating
        dest: .env
        replace:
          PORT: "${RAMP_PORT_1}"
          API_URL: "http://localhost:${RAMP_PORT_2}"
  - path: repos
    git: https://github.com/org/api.git
    auto_refresh: true
    env_files:
      - source: ../configs/api.env
        dest: .env
        replace:
          PORT: "${RAMP_PORT_2}"
  - path: repos
    git: git@github.com:org/shared-library.git
    auto_refresh: false

# Optional: Interactive prompts for team preferences
prompts:
  - name: RAMP_IDE
    question: "Which IDE do you use?"
    options:
      - value: vscode
        label: Visual Studio Code
      - value: intellij
        label: IntelliJ IDEA
      - value: vim
        label: Vim/Neovim
      - value: none
        label: None
    default: vscode

  - name: RAMP_DATABASE
    question: "Which database for local development?"
    options:
      - value: postgres
        label: PostgreSQL
      - value: mysql
        label: MySQL
    default: postgres

# Optional: Scripts to run during lifecycle events
setup: scripts/setup.sh
cleanup: scripts/cleanup.sh

# Optional: Branch naming
default-branch-prefix: feature/

# Optional: Port management
base_port: 3000
max_ports: 100
ports_per_feature: 3  # Allocate multiple ports per feature (default: 1)

# Optional: Custom commands
commands:
  - name: dev
    command: scripts/dev.sh
    scope: feature              # Only available for features
  - name: test
    command: scripts/test.sh
  - name: deploy
    command: scripts/deploy.sh
    scope: feature
  - name: doctor
    command: scripts/doctor.sh
    scope: source               # Only available for source repos

# Optional: Lifecycle hooks
hooks:
  - event: up
    command: scripts/post-setup-hook.sh
  - event: down
    command: scripts/pre-cleanup-hook.sh
  - event: run
    command: scripts/notify-on-deploy.sh
    for: deploy                 # Only runs after 'ramp run deploy'
  - event: run
    command: scripts/test-hook.sh
    for: test-*                 # Runs after any 'ramp run test-*' command
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

#### `env_files` (optional)

Automatically copy and template environment files when creating feature worktrees. Supports both simple copying and advanced templating with variable substitution.

**Simple Syntax** - Copy file as-is:
```yaml
repos:
  - path: repos
    git: git@github.com:org/app.git
    env_files:
      - .env.example     # Copies .env.example → .env.example
      - .env.local       # Copies .env.local → .env.local
```

**Advanced Syntax** - Copy with templating:
```yaml
repos:
  - path: repos
    git: git@github.com:org/app.git
    env_files:
      - source: .env.example         # Copy from source repo
        dest: .env                   # Save to worktree
      - source: ../configs/app.env   # Can reference files outside repo
        dest: .env.production
        replace:                     # Template variable substitution
          PORT: "${RAMP_PORT_1}"
          API_PORT: "${RAMP_PORT_2}"
          APP_NAME: "myapp-${RAMP_WORKTREE_NAME}"
          DATABASE_URL: "${RAMP_DATABASE_URL}"
```

**How it works:**
1. `source` - Path to file relative to source repository root (can use `../` to reference parent directories)
2. `dest` - Where to copy file in feature worktree (relative to worktree root)
3. `replace` - Key-value pairs for variable substitution (optional)
   - Keys are replaced with values using exact string matching
   - Values can reference any Ramp environment variables (`RAMP_PORT`, `RAMP_WORKTREE_NAME`, etc.)
   - Values can also reference custom prompt variables (see `prompts` section)

**Common Use Cases:**
```yaml
env_files:
  # Development configuration
  - source: .env.example
    dest: .env
    replace:
      PORT: "${RAMP_PORT}"
      NODE_ENV: "development"

  # Team-specific settings
  - source: ../configs/shared.env
    dest: .env.shared
    replace:
      IDE: "${RAMP_IDE}"              # From prompts
      DATABASE: "${RAMP_DATABASE}"    # From prompts

  # Multi-service port allocation (requires ports_per_feature: 3)
  - source: docker-compose.env
    dest: .env
    replace:
      FRONTEND_PORT: "${RAMP_PORT_1}"
      API_PORT: "${RAMP_PORT_2}"
      DB_PORT: "${RAMP_PORT_3}"
```

**Best Practices:**
- Store template files outside repos in `../configs/` to keep them centralized
- Use `ports_per_feature` in your config and reference `${RAMP_PORT_1}`, `${RAMP_PORT_2}`, etc. for multi-service setups
- Reference custom prompt variables for team-specific configurations
- Keep sensitive values in templated files, not committed `.env` files

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

**Important**: Ramp allocates ports **per feature**, not per repository. Use `ports_per_feature` to allocate multiple ports per feature for multi-service setups.

### `max_ports` (optional)

Maximum number of ports to allocate. Defaults to `100` if not specified.

```yaml
max_ports: 100
```

This creates a port range from `base_port` to `base_port + max_ports - 1`.

### `ports_per_feature` (optional)

Number of ports to allocate per feature. Defaults to `1` if not specified.

```yaml
ports_per_feature: 3
```

This is useful for multi-service setups where each feature needs multiple dedicated ports (e.g., frontend, API, database).

**Example Configuration:**
```yaml
base_port: 3000
max_ports: 100
ports_per_feature: 3
```

With this configuration:
- First feature: ports 3000, 3001, 3002
- Second feature: ports 3003, 3004, 3005
- And so on...

**Environment Variables:**

When `ports_per_feature` is set, Ramp provides indexed port variables:

| Variable | Description |
|----------|-------------|
| `RAMP_PORT` | First allocated port (backward compatible) |
| `RAMP_PORT_1` | First allocated port |
| `RAMP_PORT_2` | Second allocated port |
| `RAMP_PORT_3` | Third allocated port (if `ports_per_feature: 3`) |

**Usage in Scripts:**
```bash
#!/bin/bash
# Start multiple services on dedicated ports
docker run -p "$RAMP_PORT_1:3000" frontend-app
docker run -p "$RAMP_PORT_2:8080" api-server
docker run -p "$RAMP_PORT_3:5432" postgres
```

**Usage in env_files:**
```yaml
repos:
  - path: repos
    git: git@github.com:org/app.git
    env_files:
      - source: .env.example
        dest: .env
        replace:
          FRONTEND_PORT: "${RAMP_PORT_1}"
          API_PORT: "${RAMP_PORT_2}"
          DB_PORT: "${RAMP_PORT_3}"
```

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

#### `scope` (optional)

Restricts where the command can be run. If not specified, the command is available in both contexts.

- `source` - Command only available when running against source repositories (`ramp run <cmd>`)
- `feature` - Command only available when running against a feature (`ramp run <cmd> <feature>`)

```yaml
commands:
  - name: doctor
    command: scripts/doctor.sh
    scope: source     # Only: ramp run doctor

  - name: dev
    command: scripts/dev.sh
    scope: feature    # Only: ramp run dev my-feature

  - name: logs
    command: scripts/logs.sh
    # No scope = available everywhere
```

**Why use scope?**
- Source-only commands: environment checks, dependency updates, global setup scripts
- Feature-only commands: dev servers, feature-specific tests, deployment scripts
- In the desktop app, commands are automatically filtered based on context

**CLI Behavior:**
```bash
# If doctor has scope: source
ramp run doctor              # Works
ramp run doctor my-feature   # Error: command 'doctor' can only run against source repos

# If dev has scope: feature
ramp run dev my-feature      # Works
ramp run dev                 # Error: command 'dev' requires a feature name
```

Example custom commands:
```yaml
commands:
  - name: dev
    command: scripts/dev.sh           # Start dev servers
    scope: feature
  - name: test
    command: scripts/test.sh          # Run tests (both contexts)
  - name: deploy
    command: scripts/deploy.sh        # Deploy feature
    scope: feature
  - name: doctor
    command: scripts/doctor.sh        # Check environment
    scope: source
  - name: open
    command: scripts/open.sh          # Open in browser/editor
    scope: feature
```

### `hooks` (optional)

Lifecycle hooks allow you to run scripts at specific points during ramp operations. Hooks receive the same environment variables as commands and can be defined at project, local, or user level.

**When hooks execute:**
- `up` hooks run **after** feature creation (after setup script)
- `down` hooks run **before** feature deletion (before cleanup script)
- `run` hooks run **after** custom command execution

**Hook failure behavior:** If a hook script exits with a non-zero code, ramp shows a warning but continues with the operation.

Each hook has:

#### `event` (required)

The lifecycle event when the hook should execute.

Valid events:
- `up` - Runs after `ramp up` completes (after setup script)
- `down` - Runs before `ramp down` starts cleanup (before cleanup script)
- `run` - Runs after `ramp run <command>` completes

```yaml
hooks:
  - event: up
    command: scripts/open-ide.sh
  - event: down
    command: scripts/backup-db.sh
  - event: run
    command: scripts/notify.sh
```

#### `command` (required)

Path to script file. Relative to `.ramp/` directory (same as commands).

```yaml
hooks:
  - event: up
    command: scripts/post-setup.sh
```

#### `for` (optional, run hooks only)

For `run` hooks, filter which commands trigger the hook:
- **Empty/omitted**: runs after any `ramp run` command
- **Exact match**: runs only after specific command (e.g., `for: deploy`)
- **Glob pattern**: runs after matching commands (e.g., `for: test-*`)

```yaml
hooks:
  - event: run
    command: scripts/log-all-commands.sh
    # No 'for' = runs after ALL commands

  - event: run
    command: scripts/notify-deploy.sh
    for: deploy                       # Only after 'ramp run deploy'

  - event: run
    command: scripts/test-cleanup.sh
    for: test-*                       # After 'ramp run test-unit', 'test-e2e', etc.
```

**Common hook patterns:**

```yaml
hooks:
  # Open IDE after creating feature
  - event: up
    command: scripts/open-vscode.sh

  # Backup databases before deleting feature
  - event: down
    command: scripts/backup-feature-db.sh

  # Send Slack notification after deployment
  - event: run
    command: scripts/notify-slack.sh
    for: deploy

  # Clean up test artifacts after any test command
  - event: run
    command: scripts/cleanup-test-output.sh
    for: test-*
```

**Hooks vs setup/cleanup scripts:**
- `setup` script: Runs once during `ramp up`, typically for installing dependencies
- `cleanup` script: Runs once during `ramp down`, typically for stopping services
- `up` hooks: Additional automation after setup completes (IDE, databases, notifications)
- `down` hooks: Additional automation before cleanup starts (backups, warnings)
- `run` hooks: Automation after custom commands (logging, notifications, cleanup)

See the [Custom Scripts Guide](guides/custom-scripts.md) for detailed examples and patterns.

### `prompts` (optional)

Define interactive prompts to collect team member preferences. Ramp will prompt once per project and store responses in `.ramp/local.yaml` (which is gitignored). These values become environment variables available in scripts and env_files.

**Why use prompts?**
- IDE-agnostic development (VSCode, IntelliJ, Vim, etc.)
- Database preferences (PostgreSQL, MySQL, SQLite)
- Runtime versions (Node 18 vs 20, Python 3.11 vs 3.12)
- Personal tooling choices without committing to repo

Each prompt has:

#### `name` (required)

Environment variable name. Must start with `RAMP_` prefix and use uppercase with underscores.

```yaml
prompts:
  - name: RAMP_IDE          # Available as ${RAMP_IDE}
  - name: RAMP_DATABASE     # Available as ${RAMP_DATABASE}
  - name: RAMP_NODE_VERSION # Available as ${RAMP_NODE_VERSION}
```

#### `question` (required)

Question text shown to user during prompt.

```yaml
prompts:
  - name: RAMP_IDE
    question: "Which IDE do you use?"
```

#### `options` (required)

Array of choices. Each option has `value` (stored value) and `label` (display text).

```yaml
prompts:
  - name: RAMP_IDE
    question: "Which IDE do you use?"
    options:
      - value: vscode
        label: Visual Studio Code
      - value: intellij
        label: IntelliJ IDEA
      - value: vim
        label: Vim/Neovim
      - value: none
        label: None (Terminal only)
```

#### `default` (required)

Default value (must match one of the option values).

```yaml
prompts:
  - name: RAMP_IDE
    question: "Which IDE do you use?"
    options:
      - value: vscode
        label: Visual Studio Code
      - value: vim
        label: Vim
    default: vscode  # Must match an option value
```

**Complete Example:**
```yaml
prompts:
  - name: RAMP_IDE
    question: "Which IDE do you use for development?"
    options:
      - value: vscode
        label: Visual Studio Code
      - value: intellij
        label: IntelliJ IDEA
      - value: vim
        label: Vim/Neovim
      - value: none
        label: None
    default: vscode

  - name: RAMP_DATABASE
    question: "Which database for local development?"
    options:
      - value: postgres
        label: PostgreSQL
      - value: mysql
        label: MySQL
      - value: sqlite
        label: SQLite
    default: postgres

  - name: RAMP_NODE_VERSION
    question: "Which Node.js version?"
    options:
      - value: "18"
        label: Node 18 LTS
      - value: "20"
        label: Node 20 LTS
      - value: "22"
        label: Node 22
    default: "20"
```

**Using Prompt Values:**

In scripts:
```bash
#!/bin/bash
# .ramp/scripts/setup.sh

# Use prompt values as environment variables
echo "Setting up environment for $RAMP_IDE"

if [ "$RAMP_DATABASE" = "postgres" ]; then
  docker run -d -p "$RAMP_PORT:5432" postgres
elif [ "$RAMP_DATABASE" = "mysql" ]; then
  docker run -d -p "$RAMP_PORT:3306" mysql
fi

# Open IDE if configured
if [ "$RAMP_IDE" = "vscode" ]; then
  code "$RAMP_TREES_DIR"
elif [ "$RAMP_IDE" = "intellij" ]; then
  idea "$RAMP_TREES_DIR"
fi
```

In env_files:
```yaml
repos:
  - path: repos
    git: git@github.com:org/app.git
    env_files:
      - source: .env.example
        dest: .env
        replace:
          DATABASE_TYPE: "${RAMP_DATABASE}"
          NODE_VERSION: "${RAMP_NODE_VERSION}"
          IDE: "${RAMP_IDE}"
```

**Managing Preferences:**

Use the `ramp config` command to manage local preferences:

```bash
# View current preferences
ramp config --show

# Re-configure interactively
ramp config

# Reset to defaults (will re-prompt)
ramp config --reset
```

Local preferences are stored in `.ramp/local.yaml`:
```yaml
preferences:
  RAMP_IDE: vscode
  RAMP_DATABASE: postgres
  RAMP_NODE_VERSION: "20"
```

**Important Notes:**
- Prompts appear **once per team member** when they first run `ramp up`
- Responses are stored in `.ramp/local.yaml` (gitignored by default)
- Team members can change preferences anytime with `ramp config`
- Prompt variable names must start with `RAMP_` prefix
- All prompt values are available as environment variables

## Multi-Level Configuration

Ramp supports configuration at three levels, allowing both project-wide and personal customization:

### Configuration Levels

1. **Project Config** - `.ramp/ramp.yaml` (committed to git)
   - Full configuration including repos, setup, cleanup, ports
   - Project-wide commands and hooks
   - Shared by entire team

2. **Local Config** - `.ramp/local.yaml` (gitignored)
   - Personal preferences from prompts
   - Personal commands and hooks
   - Team member-specific customization

3. **User Config** - `~/.config/ramp/ramp.yaml` (global)
   - Personal commands and hooks that apply to ALL ramp projects
   - Cannot define repos or project-specific settings

### Merging Rules

**Commands** - First match wins (precedence: project > local > user)
- If a command named `test` exists in both project and local config, the project version is used
- Allows projects to override user defaults

**Hooks** - All execute in sequence (order: project → local → user)
- All matching hooks from all levels execute
- Project hooks run first, then local hooks, then user hooks
- Allows personal automation without affecting team

### Example: Personal IDE Hook

**Project config** (`.ramp/ramp.yaml`):
```yaml
name: my-project
repos:
  - path: repos
    git: git@github.com:org/api.git

hooks:
  - event: up
    command: scripts/start-services.sh
```

**Local config** (`.ramp/local.yaml`):
```yaml
preferences:
  RAMP_IDE: vscode

hooks:
  - event: up
    command: scripts/open-vscode.sh  # Personal IDE automation
```

**User config** (`~/.config/ramp/ramp.yaml`):
```yaml
hooks:
  - event: up
    command: scripts/start-dashboard.sh  # Personal dashboard for all projects
```

**When running `ramp up my-feature`, hooks execute in order:**
1. `scripts/start-services.sh` (project)
2. `scripts/open-vscode.sh` (local)
3. `scripts/start-dashboard.sh` (user)

### Creating User Config

Create `~/.config/ramp/ramp.yaml` for personal automation:

```bash
mkdir -p ~/.config/ramp
cat > ~/.config/ramp/ramp.yaml << 'EOF'
# Personal hooks that apply to all ramp projects
hooks:
  - event: up
    command: scripts/notify-slack.sh
  - event: down
    command: scripts/cleanup-logs.sh

# Personal commands available in all projects
commands:
  - name: notify
    command: scripts/send-notification.sh
EOF
```

**Notes:**
- Scripts in user config should use absolute paths or be in PATH
- User hooks run last, so they can observe the state after project/local hooks
- Local config merges preferences with commands/hooks (all in `.ramp/local.yaml`)

## Environment Variables

All scripts (setup, cleanup, custom commands, hooks) receive these environment variables:

### Standard Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `RAMP_PROJECT_DIR` | Absolute path to project root | `/home/user/my-project` |
| `RAMP_TREES_DIR` | Path to feature's trees directory | `/home/user/my-project/trees/my-feature` |
| `RAMP_WORKTREE_NAME` | Feature name | `my-feature` |
| `RAMP_COMMAND_NAME` | Custom command name (for `run` hooks only) | `deploy` |
| `RAMP_PORT` | First allocated port (backward compatible) | `3000` |
| `RAMP_PORT_1` | First allocated port | `3000` |
| `RAMP_PORT_2` | Second allocated port (if `ports_per_feature >= 2`) | `3001` |
| `RAMP_PORT_N` | Nth allocated port (if `ports_per_feature >= N`) | `3002` |

### Repository Path Variables

For each repository, a variable is created with the pattern `RAMP_REPO_PATH_<REPO_NAME>`:

```yaml
repos:
  - git: git@github.com:org/frontend.git      # RAMP_REPO_PATH_FRONTEND
  - git: git@github.com:org/api-server.git    # RAMP_REPO_PATH_API_SERVER
  - git: git@github.com:org/shared-lib.git    # RAMP_REPO_PATH_SHARED_LIB
```

The path depends on context:
- **Feature mode**: Points to the worktree path (`trees/<feature>/<repo>`)
- **Source mode**: Points to the source path (`repos/<repo>`)

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
echo "Ports: $RAMP_PORT_1 (frontend), $RAMP_PORT_2 (api), $RAMP_PORT_3 (db)"

# Install frontend dependencies
cd "$RAMP_TREES_DIR/frontend"
npm install

# Install API dependencies
cd "$RAMP_REPO_PATH_API_SERVER"
go mod download

# Start services on feature-specific ports (when using ports_per_feature: 3)
docker run -d -p "$RAMP_PORT_1:3000" frontend-app
docker run -d -p "$RAMP_PORT_2:8080" api-server
docker run -d -p "$RAMP_PORT_3:5432" postgres
```

## Directory Structure

```
my-project/
├── .ramp/
│   ├── ramp.yaml                # This configuration file
│   ├── local.yaml               # Local preferences (gitignored)
│   ├── port_allocations.json    # Auto-generated (DO NOT EDIT)
│   └── scripts/                 # Your scripts
│       ├── setup.sh
│       ├── cleanup.sh
│       ├── dev.sh
│       ├── test.sh
│       └── doctor.sh
├── configs/                     # Optional: Shared env templates
│   ├── frontend.env
│   ├── api.env
│   └── shared.env
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
