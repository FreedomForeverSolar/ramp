# Custom Scripts Guide

This guide covers writing setup, cleanup, custom commands, and hooks for Ramp.

## Overview

Ramp supports four types of scripts:

1. **Setup scripts** - Run once after `ramp up` creates a feature
2. **Cleanup scripts** - Run once before `ramp down` removes a feature
3. **Custom commands** - Run on-demand via `ramp run <command>`
4. **Hooks** - Run automatically at lifecycle events (up, down, run)

All scripts receive the same environment variables and context.

## Environment Variables

Every script receives these variables:

```bash
RAMP_PROJECT_DIR      # Absolute path to project root
RAMP_TREES_DIR        # Path to feature's trees directory
RAMP_WORKTREE_NAME    # Feature name
RAMP_COMMAND_NAME     # Custom command name (for run hooks only)
RAMP_PORT             # Allocated port number (if configured)
RAMP_REPO_PATH_<NAME> # Path to each repository (context-dependent)
```

### Example Values

**Feature mode** (running against a feature worktree):
```bash
RAMP_PROJECT_DIR=/home/user/my-project
RAMP_TREES_DIR=/home/user/my-project/trees/my-feature
RAMP_WORKTREE_NAME=my-feature
RAMP_COMMAND_NAME=deploy        # Only set for run hooks
RAMP_PORT=3000
RAMP_REPO_PATH_FRONTEND=/home/user/my-project/trees/my-feature/frontend
RAMP_REPO_PATH_API=/home/user/my-project/trees/my-feature/api
```

**Source mode** (running against source repos, no feature):
```bash
RAMP_PROJECT_DIR=/home/user/my-project
RAMP_REPO_PATH_FRONTEND=/home/user/my-project/repos/frontend
RAMP_REPO_PATH_API=/home/user/my-project/repos/api
```

## Setup Scripts

Setup scripts run **after** worktrees are created but **before** control returns to the user.

### Common Setup Tasks

```bash
#!/bin/bash
# .ramp/scripts/setup.sh

set -e  # Exit on error

echo "🚀 Setting up feature: $RAMP_WORKTREE_NAME"

# 1. Install dependencies
cd "$RAMP_TREES_DIR/frontend"
npm install

cd "$RAMP_TREES_DIR/api"
go mod download

# 2. Start infrastructure
docker run -d \
  --name "db-${RAMP_WORKTREE_NAME}" \
  -e POSTGRES_PASSWORD=dev \
  -p "$RAMP_PORT_2:5432" \
  postgres:15

# 3. Generate configuration files
cat > "$RAMP_TREES_DIR/api/.env" <<EOF
PORT=$RAMP_PORT_1
DATABASE_URL=postgresql://postgres:dev@localhost:$RAMP_PORT_2/myapp
EOF

# 4. Run migrations
cd "$RAMP_TREES_DIR/api"
npm run migrate

# 5. Seed data
npm run seed

echo "✅ Setup complete!"
```

### Setup Script Best Practices

**Use `set -e`**: Exit immediately if any command fails
```bash
#!/bin/bash
set -e
```

**Check for required tools**:
```bash
if ! command -v docker &> /dev/null; then
  echo "❌ Docker is required but not installed"
  exit 1
fi
```

**Provide clear feedback**:
```bash
echo "📦 Installing dependencies..."
npm install
echo "✅ Dependencies installed"
```

**Use absolute paths**:
```bash
# Good
cd "$RAMP_TREES_DIR/frontend"

# Bad
cd ../frontend
```

**Handle idempotency**:
```bash
# Stop existing container if it exists
docker stop "db-${RAMP_WORKTREE_NAME}" 2>/dev/null || true
docker rm "db-${RAMP_WORKTREE_NAME}" 2>/dev/null || true

# Now start fresh
docker run -d --name "db-${RAMP_WORKTREE_NAME}" ...
```

## Cleanup Scripts

Cleanup scripts run **before** worktrees are removed.

### Common Cleanup Tasks

```bash
#!/bin/bash
# .ramp/scripts/cleanup.sh

set -e

echo "🧹 Cleaning up feature: $RAMP_WORKTREE_NAME"

# 1. Stop running processes
pkill -f "node.*$RAMP_WORKTREE_NAME" || true

# 2. Stop Docker containers
docker stop "db-${RAMP_WORKTREE_NAME}" 2>/dev/null || true
docker rm "db-${RAMP_WORKTREE_NAME}" 2>/dev/null || true

# 3. Remove temporary files
rm -rf "$RAMP_TREES_DIR/*/node_modules/.cache"
rm -f "$RAMP_TREES_DIR/*/.env.local"

# 4. Archive logs (optional)
mkdir -p "$RAMP_PROJECT_DIR/.ramp/logs"
tar -czf "$RAMP_PROJECT_DIR/.ramp/logs/${RAMP_WORKTREE_NAME}-$(date +%Y%m%d).tar.gz" \
  "$RAMP_TREES_DIR/*/logs" 2>/dev/null || true

echo "✅ Cleanup complete!"
```

### Cleanup Script Best Practices

**Use `|| true` for non-critical operations**:
```bash
# Don't fail cleanup if container doesn't exist
docker stop "db-${RAMP_WORKTREE_NAME}" || true
```

**Clean up in reverse order of setup**:
```bash
# Setup: install deps → start DB → run migrations
# Cleanup: archive data → stop DB → remove deps
```

**Be conservative with `rm`**:
```bash
# Good - specific paths
rm -rf "$RAMP_TREES_DIR/frontend/node_modules/.cache"

# Dangerous - could delete too much
rm -rf node_modules  # Missing absolute path!
```

## Custom Commands

Custom commands let you create domain-specific workflows.

### Development Command

```bash
#!/bin/bash
# .ramp/scripts/dev.sh

set -e

echo "🚀 Starting development environment..."

# Start all services in background
cd "$RAMP_TREES_DIR/api"
npm run dev &
API_PID=$!

cd "$RAMP_TREES_DIR/frontend"
npm run dev &
FRONTEND_PID=$!

# Show URLs
echo ""
echo "✅ Development servers started!"
echo "🔗 API:      http://localhost:$RAMP_PORT_1"
echo "🌐 Frontend: http://localhost:$RAMP_PORT_2"
echo ""
echo "Press Ctrl+C to stop"

# Cleanup handler
cleanup() {
  echo ""
  echo "🛑 Stopping servers..."
  kill $API_PID $FRONTEND_PID 2>/dev/null || true
  exit 0
}

trap cleanup INT TERM

wait $API_PID $FRONTEND_PID
```

### Test Command

```bash
#!/bin/bash
# .ramp/scripts/test.sh

set -e

echo "🧪 Running tests for feature: $RAMP_WORKTREE_NAME"

# Backend tests
cd "$RAMP_TREES_DIR/api"
DATABASE_URL="postgresql://postgres:dev@localhost:$RAMP_PORT_2/test" \
  npm test

# Frontend tests
cd "$RAMP_TREES_DIR/frontend"
VITE_API_URL="http://localhost:$RAMP_PORT_1" \
  npm test

# Integration tests
cd "$RAMP_TREES_DIR/integration-tests"
npm test

echo "✅ All tests passed!"
```

### Doctor Command (Environment Check)

```bash
#!/bin/bash
# .ramp/scripts/doctor.sh

echo "🏥 Running environment checks..."

ERRORS=0

# Check required tools
check_tool() {
  if command -v "$1" &> /dev/null; then
    echo "✅ $1 installed ($($1 --version | head -n1))"
  else
    echo "❌ $1 not found"
    ERRORS=$((ERRORS + 1))
  fi
}

check_tool node
check_tool npm
check_tool docker
check_tool git

# Check Docker daemon
if docker ps &> /dev/null; then
  echo "✅ Docker daemon running"
else
  echo "❌ Docker daemon not running"
  ERRORS=$((ERRORS + 1))
fi

# Check port availability
check_port() {
  if lsof -Pi :$1 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "⚠️  Port $1 already in use"
    ERRORS=$((ERRORS + 1))
  else
    echo "✅ Port $1 available"
  fi
}

check_port "$RAMP_PORT_1"
check_port "$RAMP_PORT_2"

# Summary
echo ""
if [ $ERRORS -eq 0 ]; then
  echo "✅ All checks passed!"
  exit 0
else
  echo "❌ $ERRORS check(s) failed"
  exit 1
fi
```

### Deploy Command

```bash
#!/bin/bash
# .ramp/scripts/deploy.sh

set -e

echo "🚀 Deploying feature: $RAMP_WORKTREE_NAME"

# Build everything
cd "$RAMP_TREES_DIR/frontend"
npm run build

cd "$RAMP_TREES_DIR/api"
npm run build

# Deploy to preview environment
PREVIEW_URL="https://${RAMP_WORKTREE_NAME}.preview.myapp.com"

echo "📦 Deploying to $PREVIEW_URL..."
# Your deployment logic here

echo "✅ Deployed to $PREVIEW_URL"
```

## Hooks

Hooks are scripts that run automatically at specific lifecycle events. Unlike setup/cleanup scripts (which run once per feature), hooks can be defined at project, local, or user level and execute in sequence.

### Hook Events

- **`up` hooks** - Run after `ramp up` completes (after setup script)
- **`down` hooks** - Run before `ramp down` starts cleanup (before cleanup script)
- **`run` hooks** - Run after `ramp run <command>` completes

### Configuration

```yaml
hooks:
  - event: up
    command: scripts/open-ide.sh

  - event: down
    command: scripts/backup-db.sh

  - event: run
    command: scripts/notify.sh
    for: deploy              # Only after 'ramp run deploy'
```

### Run Hook Filtering

The `for` field filters which commands trigger a `run` hook:

```yaml
hooks:
  # Runs after ANY command
  - event: run
    command: scripts/log-command.sh

  # Runs only after 'ramp run deploy'
  - event: run
    command: scripts/notify-deploy.sh
    for: deploy

  # Runs after any test command (test-unit, test-e2e, test-integration, etc.)
  - event: run
    command: scripts/cleanup-test-data.sh
    for: test-*
```

### Hooks vs Setup/Cleanup

| Feature | Setup/Cleanup | Hooks |
|---------|---------------|-------|
| **When runs** | Once per feature lifecycle | Every time event occurs |
| **Configuration** | Single script per project | Multiple scripts, multi-level |
| **Failure behavior** | Aborts operation | Warns but continues |
| **Use case** | Core feature dependencies | Personal automation, notifications |

**Use setup/cleanup for:**
- Installing dependencies
- Starting/stopping services
- Creating/destroying databases
- Critical operations that must succeed

**Use hooks for:**
- Opening IDEs automatically
- Sending notifications
- Logging operations
- Personal workflow automation
- Non-critical side effects

### Example: IDE Automation

**Personal local hook** (`.ramp/local.yaml`):
```yaml
preferences:
  RAMP_IDE: vscode

hooks:
  - event: up
    command: scripts/open-vscode.sh
```

**Script** (`.ramp/scripts/open-vscode.sh`):
```bash
#!/bin/bash
# Open feature in VSCode after creation

if [ "$RAMP_IDE" = "vscode" ]; then
  code "$RAMP_TREES_DIR"
fi
```

This hook:
- Only runs for team members who prefer VSCode
- Doesn't affect other team members
- Fails gracefully if VSCode isn't installed (warning, not error)

### Example: Deployment Notifications

**Project hook** (`.ramp/ramp.yaml`):
```yaml
hooks:
  - event: run
    command: scripts/notify-slack.sh
    for: deploy
```

**Script** (`.ramp/scripts/notify-slack.sh`):
```bash
#!/bin/bash
# Notify team when deployment completes

WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

curl -X POST "$WEBHOOK_URL" \
  -H 'Content-Type: application/json' \
  -d "{\"text\": \"Deployed feature $RAMP_WORKTREE_NAME via $RAMP_COMMAND_NAME\"}"
```

### Example: Test Cleanup

**Project hook** (`.ramp/ramp.yaml`):
```yaml
hooks:
  - event: run
    command: scripts/cleanup-test-data.sh
    for: test-*
```

**Script** (`.ramp/scripts/cleanup-test-data.sh`):
```bash
#!/bin/bash
# Clean up test artifacts after any test command

rm -rf "$RAMP_TREES_DIR"/*/test-output/
rm -rf "$RAMP_TREES_DIR"/*/coverage/

echo "Cleaned up test artifacts for $RAMP_COMMAND_NAME"
```

### Multi-Level Hooks

Hooks can be defined at three levels:

**Project** (`.ramp/ramp.yaml`) - Team-wide automation:
```yaml
hooks:
  - event: up
    command: scripts/team-setup.sh
```

**Local** (`.ramp/local.yaml`) - Personal project automation:
```yaml
hooks:
  - event: up
    command: scripts/open-ide.sh
```

**User** (`~/.config/ramp/ramp.yaml`) - Personal global automation:
```yaml
hooks:
  - event: up
    command: scripts/start-dashboard.sh  # Runs for ALL projects
```

**Path resolution:**
- Project/local config: relative paths resolve from `.ramp/` (e.g., `scripts/setup.sh` → `.ramp/scripts/setup.sh`)
- User config: relative paths resolve from `~/.config/ramp/` (e.g., `scripts/start-dashboard.sh` → `~/.config/ramp/scripts/start-dashboard.sh`)
- Absolute paths work everywhere

**Execution order:** project → local → user

All matching hooks execute in sequence. This allows personal automation without affecting teammates.

### Hook Environment

Run hooks receive `RAMP_COMMAND_NAME`:

```bash
#!/bin/bash
# Hook that adapts to different commands

case "$RAMP_COMMAND_NAME" in
  deploy)
    echo "Deployment complete for $RAMP_WORKTREE_NAME"
    ;;
  test-*)
    echo "Tests completed: $RAMP_COMMAND_NAME"
    ;;
  *)
    echo "Command completed: $RAMP_COMMAND_NAME"
    ;;
esac
```

## Advanced Patterns

### Parallel Execution

```bash
#!/bin/bash
# Run tasks in parallel

install_deps() {
  cd "$1"
  npm install
}

# Export function for subshells
export -f install_deps

# Run in parallel
for repo in frontend api worker; do
  install_deps "$RAMP_TREES_DIR/$repo" &
done

# Wait for all to complete
wait

echo "✅ All dependencies installed"
```

### Conditional Logic Based on Repositories

```bash
#!/bin/bash
# Only run if specific repo exists

if [ -n "$RAMP_REPO_PATH_MOBILE" ]; then
  echo "📱 Setting up mobile app..."
  cd "$RAMP_TREES_DIR/mobile"
  flutter pub get
fi
```

### Using Configuration Files

```bash
#!/bin/bash
# Read from project-specific config

CONFIG_FILE="$RAMP_PROJECT_DIR/.ramp/config.json"

if [ -f "$CONFIG_FILE" ]; then
  AWS_PROFILE=$(jq -r '.aws.profile' "$CONFIG_FILE")
  AWS_REGION=$(jq -r '.aws.region' "$CONFIG_FILE")

  echo "☁️  Using AWS profile: $AWS_PROFILE in $AWS_REGION"
fi
```

### Logging

```bash
#!/bin/bash
# Log all output to file

LOG_DIR="$RAMP_PROJECT_DIR/.ramp/logs"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/${RAMP_WORKTREE_NAME}-$(date +%Y%m%d-%H%M%S).log"

# Redirect all output to log
exec 1> >(tee -a "$LOG_FILE")
exec 2>&1

echo "📝 Logging to $LOG_FILE"
```

### Shared Functions

```bash
# .ramp/scripts/common.sh
# Shared functions for all scripts

wait_for_port() {
  local port=$1
  local max_wait=${2:-30}
  local waited=0

  while ! nc -z localhost "$port" 2>/dev/null; do
    if [ $waited -ge $max_wait ]; then
      echo "❌ Timeout waiting for port $port"
      return 1
    fi
    sleep 1
    waited=$((waited + 1))
  done

  echo "✅ Port $port is ready"
}

get_port() {
  local service=$1
  case "$service" in
    api)        echo $RAMP_PORT_1 ;;
    db)         echo $RAMP_PORT_2 ;;
    frontend)   echo $RAMP_PORT_3 ;;
    *)          echo "Unknown service: $service" >&2; return 1 ;;
  esac
}
```

```bash
# .ramp/scripts/setup.sh
# Use shared functions

source "$(dirname "$0")/common.sh"

POSTGRES_PORT=$RAMP_PORT_2
docker run -d -p "$POSTGRES_PORT:5432" postgres

wait_for_port "$POSTGRES_PORT" 60
```

## Debugging Scripts

### Enable Verbose Mode

```bash
#!/bin/bash
set -x  # Print each command before executing
```

### Check Environment Variables

```bash
#!/bin/bash
echo "Environment variables:"
env | grep RAMP
```

### Run Scripts Manually

```bash
# Set environment variables manually
export RAMP_PROJECT_DIR=/home/user/my-project
export RAMP_TREES_DIR=/home/user/my-project/trees/test-feature
export RAMP_WORKTREE_NAME=test-feature
export RAMP_PORT=3000

# Run script
./.ramp/scripts/setup.sh
```

## Next Steps

- [Microservices Guide](microservices.md) - Real-world microservices examples
- [Frontend/Backend Guide](frontend-backend.md) - Full-stack development patterns
- [Configuration Reference](../configuration.md) - Configure scripts in ramp.yaml
