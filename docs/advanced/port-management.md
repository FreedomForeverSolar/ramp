# Port Management

This guide explains how Ramp's port allocation system works and strategies for managing ports across multiple services.

## Overview

Ramp allocates ports per feature from a configured range. You can configure how many ports each feature receives using `ports_per_feature`. These ports are available to all scripts via the `RAMP_PORT` and `RAMP_PORT_N` environment variables.

## Configuration

```yaml
base_port: 3000        # Starting port (default: 3000)
max_ports: 100         # Maximum ports available (default: 100)
ports_per_feature: 3   # Ports per feature (default: 1)
```

This creates a port range from `3000` to `3099` (base_port + max_ports - 1), with 3 consecutive ports allocated per feature.

## How Port Allocation Works

### Allocation on `ramp up`

When you create a feature (with `ports_per_feature: 3`):

```bash
ramp up feature-a  # Allocated ports: 3000, 3001, 3002
ramp up feature-b  # Allocated ports: 3003, 3004, 3005
ramp up feature-c  # Allocated ports: 3006, 3007, 3008
```

Ramp:
1. Scans `.ramp/port_allocations.json` for the next available consecutive ports
2. Assigns the ports to the feature
3. Persists the allocation to disk
4. Sets `RAMP_PORT`, `RAMP_PORT_1`, `RAMP_PORT_2`, etc. environment variables for scripts

### Deallocation on `ramp down`

When you remove a feature:

```bash
ramp down feature-b  # Ports 3003, 3004, 3005 released
```

Ramp:
1. Removes the port allocation entry
2. Makes all allocated ports available for future features
3. Updates `.ramp/port_allocations.json`

### Port Allocations File

```json
{
  "feature-a": [3000, 3001, 3002],
  "feature-c": [3006, 3007, 3008]
}
```

**Important**: This file is auto-generated. Don't edit manually.

## Multi-Service Strategy

For projects with multiple services (frontend, API, database, etc.), use `ports_per_feature` to allocate dedicated ports for each service.

### Native Multi-Port (Recommended)

Configure `ports_per_feature` in your `ramp.yaml`:

```yaml
base_port: 3000
max_ports: 100
ports_per_feature: 3  # Allocate 3 ports per feature
```

Access ports in your scripts using indexed environment variables:

```bash
#!/bin/bash
# .ramp/scripts/setup.sh

# Each variable is a dedicated port
FRONTEND_PORT=$RAMP_PORT_1    # 3000
API_PORT=$RAMP_PORT_2         # 3001
DB_PORT=$RAMP_PORT_3          # 3002

# Start services
docker run -d -p "$FRONTEND_PORT:3000" frontend-app
docker run -d -p "$API_PORT:8080" api-server
docker run -d -p "$DB_PORT:5432" postgres
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `RAMP_PORT` | First allocated port (backward compatible) |
| `RAMP_PORT_1` | First allocated port |
| `RAMP_PORT_2` | Second allocated port |
| `RAMP_PORT_N` | Nth allocated port (if `ports_per_feature >= N`) |

### How Multi-Port Allocation Works

With `ports_per_feature: 3`:

| Feature | RAMP_PORT_1 | RAMP_PORT_2 | RAMP_PORT_3 |
|---------|-------------|-------------|-------------|
| feature-a | 3000 | 3001 | 3002 |
| feature-b | 3003 | 3004 | 3005 |
| feature-c | 3006 | 3007 | 3008 |

Services never conflict because each feature gets its own consecutive port range.

### Alternative: Offset Pattern

For projects that only need a single allocated port but want to derive additional ports, you can use offset calculations:

```bash
#!/bin/bash
# .ramp/scripts/setup.sh

BASE=$RAMP_PORT

# Derive ports from base
FRONTEND_PORT=$((BASE + 0))
API_PORT=$((BASE + 1))
DB_PORT=$((BASE + 2))
```

**Note:** This approach requires careful coordination to avoid port collisions between features. The native `ports_per_feature` approach is recommended for most multi-service setups.

## Advanced Patterns

### Using Ports in env_files

Instead of shell scripts, you can reference port variables directly in env file templates:

```yaml
# ramp.yaml
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

### Port Helper Function (Legacy)

For projects using the offset pattern, a helper function can centralize port assignments:

```bash
# .ramp/scripts/common.sh
get_port() {
  local service=$1
  case "$service" in
    frontend)   echo $RAMP_PORT_1 ;;
    api)        echo $RAMP_PORT_2 ;;
    db)         echo $RAMP_PORT_3 ;;
    *)          echo "Unknown service: $service" >&2; return 1 ;;
  esac
}
```

### Test-Specific Ports

For test services that need additional ports beyond your configured `ports_per_feature`, increase the value or derive test ports:

```bash
#!/bin/bash
# .ramp/scripts/test.sh

# If ports_per_feature: 5, use ports 4-5 for testing
TEST_DB_PORT=$RAMP_PORT_4
TEST_REDIS_PORT=$RAMP_PORT_5

docker run -d \
  --name "test-db-${RAMP_WORKTREE_NAME}" \
  -p "$TEST_DB_PORT:5432" \
  postgres:15

npm test -- --db-port="$TEST_DB_PORT"
```

## Choosing base_port, max_ports, and ports_per_feature

### Guidelines

**base_port**: Choose a range that doesn't conflict with:
- System ports (< 1024)
- Common application ports (3000, 8000, 8080, 5000, etc.)
- Other development tools

**max_ports**: Based on:
- Team size
- Average active features per developer
- Number of ports per feature (`ports_per_feature` × expected features)

**ports_per_feature**: Based on:
- Number of services in your stack (frontend, API, database, etc.)
- Test service requirements
- Generally 1-5 ports per feature is typical

### Examples

**Single service (default):**
```yaml
base_port: 3000
max_ports: 100
# ports_per_feature defaults to 1
```

**Multi-service stack (frontend + API + DB):**
```yaml
base_port: 3000
max_ports: 300          # 100 features × 3 ports each
ports_per_feature: 3
```

**Large team with microservices:**
```yaml
base_port: 10000
max_ports: 500
ports_per_feature: 5    # More ports for complex stacks
```

**Avoiding conflicts with common ports:**
```yaml
base_port: 30000        # Well above common ports
max_ports: 300
ports_per_feature: 3
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

echo "Port allocation status:"
echo ""

# Check allocated ports
if [ -f ".ramp/port_allocations.json" ]; then
  # Parse JSON and check each feature's ports
  jq -r 'to_entries[] | "\(.key):\(.value | join(","))"' .ramp/port_allocations.json | \
    while IFS=: read feature ports; do
      echo "Feature: $feature"
      for port in ${ports//,/ }; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
          echo "  $port (in use)"
        else
          echo "  $port (allocated but not in use)"
        fi
      done
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

1. **Use `ports_per_feature`** - Configure the number of ports you need upfront rather than using offset calculations
2. **Choose high base_port** - Avoid conflicts with system services (3000+)
3. **Size max_ports appropriately** - Account for `ports_per_feature` × expected concurrent features
4. **Clean up regularly** - Use `ramp prune` to release ports from merged features
5. **Check for conflicts** - Add port checks to `doctor` script
6. **Automate cleanup** - Ensure cleanup script stops all services on allocated ports
7. **Use env_files** - Reference `${RAMP_PORT_N}` in env file templates for cleaner configuration

## Next Steps

- [Custom Scripts Guide](../guides/custom-scripts.md) - Use ports in your scripts
- [Microservices Guide](../guides/microservices.md) - Multi-service port strategies
- [Troubleshooting](troubleshooting.md) - More debugging tips
