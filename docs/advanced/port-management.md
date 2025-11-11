# Port Management

This guide explains how Ramp's port allocation system works and strategies for managing ports across multiple services.

## Overview

Ramp allocates **exactly one port per feature** from a configured range. This unique port is available to all scripts via the `RAMP_PORT` environment variable.

## Configuration

```yaml
base_port: 3000   # Starting port (default: 3000)
max_ports: 100    # Maximum features (default: 100)
```

This creates a port range from `3000` to `3099` (base_port + max_ports - 1).

## How Port Allocation Works

### Allocation on `ramp up`

When you create a feature:

```bash
ramp up feature-a  # Allocated port: 3000
ramp up feature-b  # Allocated port: 3001
ramp up feature-c  # Allocated port: 3002
```

Ramp:
1. Scans `.ramp/port_allocations.json` for the next available port
2. Assigns the port to the feature
3. Persists the allocation to disk
4. Sets `RAMP_PORT` environment variable for scripts

### Deallocation on `ramp down`

When you remove a feature:

```bash
ramp down feature-b  # Port 3001 released
```

Ramp:
1. Removes the port allocation entry
2. Makes the port available for future features
3. Updates `.ramp/port_allocations.json`

### Port Allocations File

```json
{
  "base_port": 3000,
  "max_ports": 100,
  "allocations": {
    "feature-a": 3000,
    "feature-c": 3002
  }
}
```

**Important**: This file is auto-generated. Don't edit manually.

## Multi-Service Strategy

Since each feature gets **one** port, use a deterministic offset strategy for multiple services.

### Simple Offset Pattern

```bash
#!/bin/bash
# .ramp/scripts/setup.sh

BASE=$RAMP_PORT

# Service ports (offset by 1-10)
FRONTEND_PORT=$((BASE + 1))      # 3001
API_PORT=$((BASE + 2))           # 3002
WORKER_PORT=$((BASE + 3))        # 3003

# Infrastructure ports (offset by 30+)
POSTGRES_PORT=$((BASE + 32))     # 3032
REDIS_PORT=$((BASE + 79))        # 3079
RABBITMQ_PORT=$((BASE + 72))     # 3072
```

### Why This Works

Each feature uses the same offset pattern:

| Feature | RAMP_PORT | Frontend | API | Postgres | Redis |
|---------|-----------|----------|-----|----------|-------|
| feature-a | 3000 | 3001 | 3002 | 3032 | 3079 |
| feature-b | 3100 | 3101 | 3102 | 3132 | 3179 |
| feature-c | 3200 | 3201 | 3202 | 3232 | 3279 |

Services never conflict because each feature has a unique base port.

### Recommended Offsets

Choose offsets that won't collide:

```bash
# Application services: +1 to +9
FRONTEND_PORT=$((RAMP_PORT + 1))
API_PORT=$((RAMP_PORT + 2))
ADMIN_PORT=$((RAMP_PORT + 3))
WORKER_PORT=$((RAMP_PORT + 4))

# Databases: +30 to +39
POSTGRES_PORT=$((RAMP_PORT + 32))
MYSQL_PORT=$((RAMP_PORT + 33))
MONGO_PORT=$((RAMP_PORT + 34))

# Caches: +70 to +79
REDIS_PORT=$((RAMP_PORT + 79))
MEMCACHED_PORT=$((RAMP_PORT + 78))

# Message queues: +40 to +49
RABBITMQ_PORT=$((RAMP_PORT + 72))
KAFKA_PORT=$((RAMP_PORT + 73))
```

## Advanced Patterns

### Port Range Helper Function

```bash
# .ramp/scripts/common.sh

get_port() {
  local service=$1
  local base=$RAMP_PORT

  case "$service" in
    frontend)   echo $((base + 1)) ;;
    api)        echo $((base + 2)) ;;
    admin)      echo $((base + 3)) ;;
    worker)     echo $((base + 4)) ;;
    postgres)   echo $((base + 32)) ;;
    redis)      echo $((base + 79)) ;;
    *)          echo "Unknown service: $service" >&2; return 1 ;;
  esac
}
```

```bash
# .ramp/scripts/setup.sh
source "$(dirname "$0")/common.sh"

FRONTEND_PORT=$(get_port frontend)
API_PORT=$(get_port api)
```

### Configuration File Approach

```bash
# .ramp/port-map.env
FRONTEND_OFFSET=1
API_OFFSET=2
WORKER_OFFSET=3
POSTGRES_OFFSET=32
REDIS_OFFSET=79
```

```bash
# .ramp/scripts/setup.sh
source "$(dirname "$0")/../port-map.env"

FRONTEND_PORT=$((RAMP_PORT + FRONTEND_OFFSET))
API_PORT=$((RAMP_PORT + API_OFFSET))
```

### Dynamic Port Allocation for Tests

```bash
#!/bin/bash
# .ramp/scripts/test.sh

# Use higher offsets for test services to avoid conflicts
TEST_DB_PORT=$((RAMP_PORT + 100))
TEST_REDIS_PORT=$((RAMP_PORT + 179))

docker run -d \
  --name "test-db-${RAMP_WORKTREE_NAME}" \
  -p "$TEST_DB_PORT:5432" \
  postgres:15

npm test -- --db-port="$TEST_DB_PORT"
```

## Choosing base_port and max_ports

### Guidelines

**base_port**: Choose a range that doesn't conflict with:
- System ports (< 1024)
- Common application ports (3000, 8000, 8080, 5000, etc.)
- Other development tools

**max_ports**: Based on:
- Team size
- Average active features per developer
- Port offset strategy (how many ports each feature uses)

### Examples

**Small team (1-3 developers):**
```yaml
base_port: 3000
max_ports: 30    # 30 features max
```

**Medium team (5-10 developers):**
```yaml
base_port: 4000
max_ports: 100   # 100 features max
```

**Large team (10+ developers) or heavy port usage:**
```yaml
base_port: 10000
max_ports: 500   # 500 features max, ports 10000-10499
```

**Avoiding conflicts with common ports:**
```yaml
base_port: 30000  # Well above common ports
max_ports: 1000   # Ports 30000-30999
```

## Port Conflict Resolution

### Detecting Conflicts

```bash
# Check if port is in use
lsof -i :3000
```

### Handling Conflicts in Scripts

```bash
#!/bin/bash

is_port_in_use() {
  lsof -Pi :$1 -sTCP:LISTEN -t >/dev/null 2>&1
}

start_service() {
  local port=$1

  if is_port_in_use "$port"; then
    echo "⚠️  Port $port is already in use"
    echo "This may be from a previous feature. Trying to clean up..."

    # Try to find and stop the conflicting process
    local pid=$(lsof -ti :$port)
    if [ -n "$pid" ]; then
      kill "$pid" 2>/dev/null || true
      sleep 2
    fi
  fi

  # Start service
  npm run dev -- --port="$port"
}
```

### Checking Available Ports

Add to `ramp status` workflow:

```bash
#!/bin/bash
# .ramp/scripts/doctor.sh

echo "📊 Port allocation status:"
echo ""

# Check allocated ports
if [ -f ".ramp/port_allocations.json" ]; then
  # Parse JSON and check each port
  jq -r '.allocations | to_entries[] | "\(.key): \(.value)"' .ramp/port_allocations.json | \
    while IFS=: read feature port; do
      if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "✅ $feature: $port (in use)"
      else
        echo "⚠️  $feature: $port (allocated but not in use)"
      fi
    done
fi
```

## Troubleshooting

### Port Already in Use

**Symptom**: Service fails to start with "port already in use" error

**Solutions**:

1. Check what's using the port:
```bash
lsof -i :3000
```

2. Check if another Ramp feature is using it:
```bash
ramp status
```

3. Kill the process:
```bash
kill $(lsof -ti :3000)
```

4. Check for orphaned Docker containers:
```bash
docker ps | grep ramp
docker stop <container-id>
```

### Port Allocations Out of Sync

**Symptom**: `.ramp/port_allocations.json` shows features that don't exist

**Solution**: Clean up manually:

```bash
# Remove the file
rm .ramp/port_allocations.json

# Next ramp command will regenerate it
ramp status
```

Or edit the JSON file to remove invalid entries.

### Running Out of Ports

**Symptom**: `ramp up` fails with "no available ports"

**Solutions**:

1. Clean up merged features:
```bash
ramp prune
```

2. Remove old features:
```bash
ramp down old-feature-1
ramp down old-feature-2
```

3. Increase `max_ports` in `ramp.yaml`:
```yaml
max_ports: 200  # Increased from 100
```

### Docker Port Binding Fails

**Symptom**: Docker container fails to start with "bind: address already in use"

**Solution**:

```bash
# List all Docker containers (including stopped)
docker ps -a | grep ramp

# Remove orphaned containers
docker rm $(docker ps -a -q -f "name=ramp-")

# Or force remove all
docker rm -f $(docker ps -a -q -f "name=ramp-")
```

## Best Practices

1. **Document your offset strategy** - Add comments to setup scripts
2. **Use consistent offsets** - Don't change offsets between features
3. **Choose high base_port** - Avoid conflicts with system services
4. **Clean up regularly** - Use `ramp prune` to release ports
5. **Check for conflicts** - Add port checks to `doctor` script
6. **Automate cleanup** - Ensure cleanup script stops all services
7. **Test port allocation** - Verify scripts work with different RAMP_PORT values

## Next Steps

- [Custom Scripts Guide](../guides/custom-scripts.md) - Use ports in your scripts
- [Microservices Guide](../guides/microservices.md) - Multi-service port strategies
- [Troubleshooting](troubleshooting.md) - More debugging tips
