# Custom Scripts Guide

This guide covers writing setup, cleanup, and custom command scripts for Ramp.

## Overview

Ramp supports three types of scripts:

1. **Setup scripts** - Run after `ramp up` creates a feature
2. **Cleanup scripts** - Run before `ramp down` removes a feature
3. **Custom commands** - Run on-demand via `ramp run <command>`

All scripts receive the same environment variables and context.

## Environment Variables

Every script receives these variables:

```bash
RAMP_PROJECT_DIR      # Absolute path to project root
RAMP_TREES_DIR        # Path to feature's trees directory
RAMP_WORKTREE_NAME    # Feature name
RAMP_PORT             # Allocated port number (if configured)
RAMP_REPO_PATH_<NAME> # Path to each repository's source
```

### Example Values

```bash
RAMP_PROJECT_DIR=/home/user/my-project
RAMP_TREES_DIR=/home/user/my-project/trees/my-feature
RAMP_WORKTREE_NAME=my-feature
RAMP_PORT=3000
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
  -p "$((RAMP_PORT + 32)):5432" \
  postgres:15

# 3. Generate configuration files
cat > "$RAMP_TREES_DIR/api/.env" <<EOF
PORT=$RAMP_PORT
DATABASE_URL=postgresql://postgres:dev@localhost:$((RAMP_PORT + 32))/myapp
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
echo "🔗 API:      http://localhost:$RAMP_PORT"
echo "🌐 Frontend: http://localhost:$((RAMP_PORT + 1))"
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
DATABASE_URL="postgresql://postgres:dev@localhost:$((RAMP_PORT + 32))/test" \
  npm test

# Frontend tests
cd "$RAMP_TREES_DIR/frontend"
VITE_API_URL="http://localhost:$RAMP_PORT" \
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

check_port "$RAMP_PORT"
check_port "$((RAMP_PORT + 1))"

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

calculate_port() {
  local offset=$1
  echo $((RAMP_PORT + offset))
}
```

```bash
# .ramp/scripts/setup.sh
# Use shared functions

source "$(dirname "$0")/common.sh"

POSTGRES_PORT=$(calculate_port 32)
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
