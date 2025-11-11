# Microservices Development Guide

This guide shows how to use Ramp for coordinating development across multiple microservices.

## Overview

Microservices architectures typically involve:
- Multiple repositories (one per service)
- Shared databases and message queues
- Service-to-service communication
- Port conflicts between features
- Complex setup and teardown

Ramp automates all of this with feature-scoped environments.

## Example Configuration

```yaml
name: my-microservices-app

repos:
  - path: repos
    git: git@github.com:org/auth-service.git
    auto_refresh: true
  - path: repos
    git: git@github.com:org/user-service.git
    auto_refresh: true
  - path: repos
    git: git@github.com:org/payment-service.git
    auto_refresh: true
  - path: repos
    git: git@github.com:org/api-gateway.git
    auto_refresh: true

setup: scripts/setup.sh
cleanup: scripts/cleanup.sh

default-branch-prefix: feature/

base_port: 3000
max_ports: 50

commands:
  - name: dev
    command: scripts/dev.sh
  - name: test
    command: scripts/test.sh
  - name: logs
    command: scripts/logs.sh
```

## Setup Script Example

```bash
#!/bin/bash
# .ramp/scripts/setup.sh

set -e

echo "🚀 Setting up microservices for feature: $RAMP_WORKTREE_NAME"
echo "📍 Base port: $RAMP_PORT"

# Calculate service ports from base port
BASE=$RAMP_PORT
AUTH_PORT=$((BASE + 1))      # 3001
USER_PORT=$((BASE + 2))      # 3002
PAYMENT_PORT=$((BASE + 3))   # 3003
GATEWAY_PORT=$((BASE + 4))   # 3004
POSTGRES_PORT=$((BASE + 32)) # 3032
REDIS_PORT=$((BASE + 79))    # 3079

# Start infrastructure
echo "🐘 Starting PostgreSQL on port $POSTGRES_PORT..."
docker run -d \
  --name "ramp-${RAMP_WORKTREE_NAME}-postgres" \
  -e POSTGRES_PASSWORD=dev \
  -p "$POSTGRES_PORT:5432" \
  postgres:15

echo "🔴 Starting Redis on port $REDIS_PORT..."
docker run -d \
  --name "ramp-${RAMP_WORKTREE_NAME}-redis" \
  -p "$REDIS_PORT:6379" \
  redis:7

# Install dependencies for each service
for service in auth-service user-service payment-service api-gateway; do
  echo "📦 Installing dependencies for $service..."
  cd "$RAMP_TREES_DIR/$service"
  npm install
done

# Generate environment files
cat > "$RAMP_TREES_DIR/auth-service/.env" <<EOF
PORT=$AUTH_PORT
DATABASE_URL=postgresql://postgres:dev@localhost:$POSTGRES_PORT/auth
REDIS_URL=redis://localhost:$REDIS_PORT
JWT_SECRET=dev-secret-$RAMP_WORKTREE_NAME
EOF

cat > "$RAMP_TREES_DIR/user-service/.env" <<EOF
PORT=$USER_PORT
DATABASE_URL=postgresql://postgres:dev@localhost:$POSTGRES_PORT/users
REDIS_URL=redis://localhost:$REDIS_PORT
AUTH_SERVICE_URL=http://localhost:$AUTH_PORT
EOF

cat > "$RAMP_TREES_DIR/payment-service/.env" <<EOF
PORT=$PAYMENT_PORT
DATABASE_URL=postgresql://postgres:dev@localhost:$POSTGRES_PORT/payments
REDIS_URL=redis://localhost:$REDIS_PORT
AUTH_SERVICE_URL=http://localhost:$AUTH_PORT
USER_SERVICE_URL=http://localhost:$USER_PORT
EOF

cat > "$RAMP_TREES_DIR/api-gateway/.env" <<EOF
PORT=$GATEWAY_PORT
AUTH_SERVICE_URL=http://localhost:$AUTH_PORT
USER_SERVICE_URL=http://localhost:$USER_PORT
PAYMENT_SERVICE_URL=http://localhost:$PAYMENT_PORT
EOF

# Run database migrations
echo "🗄️  Running database migrations..."
cd "$RAMP_TREES_DIR/auth-service"
npm run migrate

cd "$RAMP_TREES_DIR/user-service"
npm run migrate

cd "$RAMP_TREES_DIR/payment-service"
npm run migrate

echo "✅ Setup complete!"
echo "📝 Run 'ramp run dev' to start all services"
echo "🌐 API Gateway: http://localhost:$GATEWAY_PORT"
```

## Cleanup Script Example

```bash
#!/bin/bash
# .ramp/scripts/cleanup.sh

set -e

echo "🧹 Cleaning up microservices for feature: $RAMP_WORKTREE_NAME"

# Stop and remove Docker containers
echo "🐘 Stopping PostgreSQL..."
docker stop "ramp-${RAMP_WORKTREE_NAME}-postgres" 2>/dev/null || true
docker rm "ramp-${RAMP_WORKTREE_NAME}-postgres" 2>/dev/null || true

echo "🔴 Stopping Redis..."
docker stop "ramp-${RAMP_WORKTREE_NAME}-redis" 2>/dev/null || true
docker rm "ramp-${RAMP_WORKTREE_NAME}-redis" 2>/dev/null || true

echo "✅ Cleanup complete!"
```

## Development Command

```bash
#!/bin/bash
# .ramp/scripts/dev.sh

set -e

echo "🚀 Starting all services for feature: $RAMP_WORKTREE_NAME"

# Calculate ports
BASE=$RAMP_PORT
AUTH_PORT=$((BASE + 1))
USER_PORT=$((BASE + 2))
PAYMENT_PORT=$((BASE + 3))
GATEWAY_PORT=$((BASE + 4))

# Start all services in background using tmux or separate terminals
cd "$RAMP_TREES_DIR/auth-service"
npm run dev &
AUTH_PID=$!

cd "$RAMP_TREES_DIR/user-service"
npm run dev &
USER_PID=$!

cd "$RAMP_TREES_DIR/payment-service"
npm run dev &
PAYMENT_PID=$!

cd "$RAMP_TREES_DIR/api-gateway"
npm run dev &
GATEWAY_PID=$!

echo "✅ All services started!"
echo ""
echo "🔗 Service URLs:"
echo "  Auth Service:    http://localhost:$AUTH_PORT"
echo "  User Service:    http://localhost:$USER_PORT"
echo "  Payment Service: http://localhost:$PAYMENT_PORT"
echo "  API Gateway:     http://localhost:$GATEWAY_PORT"
echo ""
echo "Press Ctrl+C to stop all services"

# Wait for all background processes
wait $AUTH_PID $USER_PID $PAYMENT_PID $GATEWAY_PID
```

## Workflow

### Creating a Feature

```bash
ramp up add-oauth-login
```

This creates:
- Feature branches in all 4 services
- Isolated Docker containers for Postgres and Redis
- Environment files with feature-specific ports
- Database schemas via migrations

### Development

```bash
ramp run dev
```

Opens all services on unique ports. Each feature gets its own:
- Port range (3000-3004 for services, 3032 for DB, etc.)
- Database instance
- Redis instance

### Testing

```bash
# .ramp/scripts/test.sh
cd "$RAMP_TREES_DIR/auth-service"
npm test

cd "$RAMP_TREES_DIR/user-service"
npm test

# Integration tests against the API gateway
cd "$RAMP_TREES_DIR/api-gateway"
GATEWAY_URL="http://localhost:$((RAMP_PORT + 4))" npm run integration-test
```

### Cleanup

```bash
ramp down add-oauth-login
```

Removes worktrees, branches, and Docker containers.

## Port Allocation Strategy

Since Ramp allocates **one port per feature**, use a deterministic offset strategy:

| Service | Offset | Example (base=3000) |
|---------|--------|---------------------|
| Base | 0 | 3000 |
| Auth Service | +1 | 3001 |
| User Service | +2 | 3002 |
| Payment Service | +3 | 3003 |
| API Gateway | +4 | 3004 |
| PostgreSQL | +32 | 3032 |
| Redis | +79 | 3079 |

This ensures:
- Each feature has unique ports
- Services don't conflict between features
- Ports are predictable and debuggable

## Benefits

✅ **Parallel Development**: Multiple developers work on different features simultaneously without port conflicts

✅ **Isolated Testing**: Each feature has its own database, avoiding test data conflicts

✅ **Simple Cleanup**: `ramp down` removes everything - no manual Docker cleanup

✅ **Consistent Environments**: Every feature gets identical setup

✅ **Fast Context Switching**: Switch between features with `ramp run dev <feature>`

## Best Practices

1. **Use Docker for infrastructure** - Easy to start/stop, isolate between features
2. **Script port calculations** - Use `$RAMP_PORT` + offsets consistently
3. **Generate .env files** - Don't commit them, generate per-feature
4. **Run migrations in setup** - Ensure database schema is ready
5. **Health checks** - Add to dev script to ensure services are ready
6. **Logs command** - Create a custom command to tail all service logs

## Next Steps

- [Custom Scripts Guide](custom-scripts.md) - Advanced scripting techniques
- [Port Management](../advanced/port-management.md) - Deep dive into port allocation
- [Troubleshooting](../advanced/troubleshooting.md) - Common issues
