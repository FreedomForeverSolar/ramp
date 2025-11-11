# Frontend/Backend Development Guide

This guide shows how to use Ramp for full-stack development with separate frontend and backend repositories.

## Overview

Full-stack applications typically have:
- Separate frontend (React, Vue, Angular, etc.) and backend (Node, Go, Python, etc.) repositories
- API integration between frontend and backend
- Shared development environment (database, auth, etc.)
- Coordinated feature development across both codebases

Ramp makes it easy to work on features that span both frontend and backend.

## Example Configuration

```yaml
name: my-fullstack-app

repos:
  - path: repos
    git: git@github.com:org/web-frontend.git
    auto_refresh: true
  - path: repos
    git: git@github.com:org/api-backend.git
    auto_refresh: true

setup: scripts/setup.sh
cleanup: scripts/cleanup.sh

default-branch-prefix: feature/

base_port: 3000
max_ports: 30

commands:
  - name: dev
    command: scripts/dev.sh
  - name: build
    command: scripts/build.sh
  - name: test
    command: scripts/test.sh
```

## Setup Script Example

```bash
#!/bin/bash
# .ramp/scripts/setup.sh

set -e

echo "🚀 Setting up full-stack feature: $RAMP_WORKTREE_NAME"

# Calculate ports
BACKEND_PORT=$RAMP_PORT              # 3000
FRONTEND_PORT=$((RAMP_PORT + 1))     # 3001
POSTGRES_PORT=$((RAMP_PORT + 32))    # 3032

echo "📦 Installing backend dependencies..."
cd "$RAMP_TREES_DIR/api-backend"
npm install  # or: go mod download, pip install -r requirements.txt, etc.

echo "📦 Installing frontend dependencies..."
cd "$RAMP_TREES_DIR/web-frontend"
npm install

echo "🐘 Starting PostgreSQL..."
docker run -d \
  --name "ramp-${RAMP_WORKTREE_NAME}-db" \
  -e POSTGRES_PASSWORD=dev \
  -e POSTGRES_DB=myapp \
  -p "$POSTGRES_PORT:5432" \
  postgres:15

# Wait for database
echo "⏳ Waiting for database..."
sleep 3

echo "🗄️  Running migrations..."
cd "$RAMP_TREES_DIR/api-backend"
DATABASE_URL="postgresql://postgres:dev@localhost:$POSTGRES_PORT/myapp" npm run migrate

# Generate environment files
cat > "$RAMP_TREES_DIR/api-backend/.env" <<EOF
PORT=$BACKEND_PORT
DATABASE_URL=postgresql://postgres:dev@localhost:$POSTGRES_PORT/myapp
JWT_SECRET=dev-secret-$RAMP_WORKTREE_NAME
CORS_ORIGIN=http://localhost:$FRONTEND_PORT
NODE_ENV=development
EOF

cat > "$RAMP_TREES_DIR/web-frontend/.env" <<EOF
VITE_API_URL=http://localhost:$BACKEND_PORT
VITE_APP_NAME=MyApp ($RAMP_WORKTREE_NAME)
PORT=$FRONTEND_PORT
EOF

echo "✅ Setup complete!"
echo "📝 Run 'ramp run dev' to start frontend and backend"
echo "🌐 Frontend: http://localhost:$FRONTEND_PORT"
echo "🔗 Backend:  http://localhost:$BACKEND_PORT"
```

## Cleanup Script Example

```bash
#!/bin/bash
# .ramp/scripts/cleanup.sh

set -e

echo "🧹 Cleaning up feature: $RAMP_WORKTREE_NAME"

echo "🐘 Stopping database..."
docker stop "ramp-${RAMP_WORKTREE_NAME}-db" 2>/dev/null || true
docker rm "ramp-${RAMP_WORKTREE_NAME}-db" 2>/dev/null || true

echo "✅ Cleanup complete!"
```

## Development Command

```bash
#!/bin/bash
# .ramp/scripts/dev.sh

set -e

echo "🚀 Starting development servers..."

# Start backend in background
cd "$RAMP_TREES_DIR/api-backend"
npm run dev &
BACKEND_PID=$!

# Start frontend in background
cd "$RAMP_TREES_DIR/web-frontend"
npm run dev &
FRONTEND_PID=$!

BACKEND_PORT=$RAMP_PORT
FRONTEND_PORT=$((RAMP_PORT + 1))

echo "✅ Servers started!"
echo ""
echo "🌐 Frontend: http://localhost:$FRONTEND_PORT"
echo "🔗 Backend:  http://localhost:$BACKEND_PORT"
echo ""
echo "Press Ctrl+C to stop both servers"

# Cleanup function
cleanup() {
  echo ""
  echo "🛑 Stopping servers..."
  kill $BACKEND_PID $FRONTEND_PID 2>/dev/null || true
  exit 0
}

trap cleanup INT TERM

# Wait for both processes
wait $BACKEND_PID $FRONTEND_PID
```

## Build Command Example

```bash
#!/bin/bash
# .ramp/scripts/build.sh

set -e

echo "🔨 Building feature: $RAMP_WORKTREE_NAME"

echo "📦 Building backend..."
cd "$RAMP_TREES_DIR/api-backend"
npm run build

echo "📦 Building frontend..."
cd "$RAMP_TREES_DIR/web-frontend"
npm run build

echo "✅ Build complete!"
echo "📁 Backend build: $RAMP_TREES_DIR/api-backend/dist"
echo "📁 Frontend build: $RAMP_TREES_DIR/web-frontend/dist"
```

## Test Command Example

```bash
#!/bin/bash
# .ramp/scripts/test.sh

set -e

echo "🧪 Running tests for feature: $RAMP_WORKTREE_NAME"

# Backend tests
echo "📡 Testing backend..."
cd "$RAMP_TREES_DIR/api-backend"
DATABASE_URL="postgresql://postgres:dev@localhost:$((RAMP_PORT + 32))/myapp" npm test

# Frontend tests
echo "🎨 Testing frontend..."
cd "$RAMP_TREES_DIR/web-frontend"
npm test

echo "✅ All tests passed!"
```

## Workflow Example

### Creating a Feature

```bash
ramp up user-dashboard
```

Creates:
- `feature/user-dashboard` branch in frontend repo
- `feature/user-dashboard` branch in backend repo
- Isolated database instance
- Environment files with correct API URLs

### Development

```bash
cd trees/user-dashboard
ramp run dev
```

Frontend automatically connects to backend via environment variable.

### Making Changes

**Backend changes:**
```bash
cd trees/user-dashboard/api-backend
# Add new API endpoint
vim src/routes/dashboard.js
git add .
git commit -m "Add dashboard endpoint"
```

**Frontend changes:**
```bash
cd trees/user-dashboard/web-frontend
# Consume new endpoint
vim src/pages/Dashboard.jsx
git add .
git commit -m "Add dashboard page"
```

### Testing Integration

```bash
ramp run test user-dashboard
```

Runs both backend and frontend tests with correct database connection.

### Cleanup

```bash
ramp down user-dashboard
```

Removes branches, worktrees, database, and feature directory.

## Port Allocation

| Component | Offset | Example (base=3000) |
|-----------|--------|---------------------|
| Backend | 0 | 3000 |
| Frontend | +1 | 3001 |
| PostgreSQL | +32 | 3032 |
| Redis (optional) | +79 | 3079 |

## Common Patterns

### Proxy Configuration (Vite/Webpack)

If you prefer to proxy API calls through the frontend dev server:

```javascript
// vite.config.js
export default {
  server: {
    port: process.env.PORT || 3001,
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.BACKEND_PORT || 3000}`,
        changeOrigin: true,
      },
    },
  },
}
```

Then in setup.sh:
```bash
cat > "$RAMP_TREES_DIR/web-frontend/.env" <<EOF
PORT=$FRONTEND_PORT
BACKEND_PORT=$BACKEND_PORT
EOF
```

### Database Seeding

Add to setup script:
```bash
echo "🌱 Seeding database..."
cd "$RAMP_TREES_DIR/api-backend"
npm run seed
```

### Opening in Browser

```bash
#!/bin/bash
# .ramp/scripts/open.sh

FRONTEND_PORT=$((RAMP_PORT + 1))

# macOS
open "http://localhost:$FRONTEND_PORT"

# Linux
xdg-open "http://localhost:$FRONTEND_PORT"

# Windows (WSL)
explorer.exe "http://localhost:$FRONTEND_PORT"
```

Then run:
```bash
ramp run open user-dashboard
```

## Benefits

✅ **Synchronized Branches**: Frontend and backend always on matching feature branches

✅ **Isolated Environments**: Each feature has its own database and ports

✅ **No Configuration Drift**: Setup scripts ensure consistent environment

✅ **Easy Context Switching**: Switch between features without manual reconfiguration

✅ **Parallel Development**: Multiple features in development simultaneously

## Troubleshooting

**CORS Errors**: Ensure backend `.env` has `CORS_ORIGIN` set to frontend URL

**Database Connection Failed**: Check that Docker container is running:
```bash
docker ps | grep ramp-${FEATURE_NAME}
```

**Port Already in Use**: Check port allocations:
```bash
ramp status
```

**Frontend Can't Reach Backend**: Verify environment variables:
```bash
cat trees/user-dashboard/web-frontend/.env
```

## Next Steps

- [Custom Scripts Guide](custom-scripts.md) - Advanced automation
- [Microservices Guide](microservices.md) - Multi-service architectures
- [Port Management](../advanced/port-management.md) - Port allocation strategies
