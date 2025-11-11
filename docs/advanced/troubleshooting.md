# Troubleshooting

Common issues and solutions when using Ramp.

## Installation Issues

### "ramp: command not found"

**Cause**: Binary not in PATH or not installed globally

**Solutions**:

1. Run from project directory:
```bash
./ramp --help
```

2. Install globally:
```bash
sudo ./install.sh
```

3. Add to PATH:
```bash
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

4. Verify installation:
```bash
which ramp
ramp version
```

### Homebrew Installation Fails

**Cause**: Tap not added or outdated

**Solutions**:

```bash
# Update Homebrew
brew update

# Try installing again
brew install freedomforeversolar/tools/ramp

# If still failing, tap explicitly
brew tap freedomforeversolar/tools
brew install ramp
```

### Build from Source Fails

**Cause**: Go version mismatch or missing dependencies

**Solutions**:

1. Check Go version (requires 1.21+):
```bash
go version
```

2. Update Go if needed:
```bash
# macOS
brew upgrade go

# Linux
# Download from https://go.dev/dl/
```

3. Clean and rebuild:
```bash
go clean
go mod download
go build -o ramp .
```

## Configuration Issues

### "No .ramp/ramp.yaml found"

**Cause**: Running command outside a Ramp project

**Solutions**:

1. Navigate to project directory:
```bash
cd /path/to/my-project
ramp status
```

2. Initialize a new project:
```bash
ramp init
```

3. Check current directory:
```bash
pwd
ls -la .ramp/
```

### "Failed to parse ramp.yaml"

**Cause**: Invalid YAML syntax

**Solutions**:

1. Validate YAML:
```bash
# Using yamllint (install with: pip install yamllint)
yamllint .ramp/ramp.yaml

# Or use online validator: https://www.yamllint.com/
```

2. Common syntax errors:
```yaml
# ❌ Bad: inconsistent indentation
repos:
  - path: repos
      git: url  # Too much indentation

# ✅ Good: consistent indentation
repos:
  - path: repos
    git: url
```

3. Check for tabs (YAML requires spaces):
```bash
cat -A .ramp/ramp.yaml | grep $'\t'
```

### Scripts Not Executing

**Cause**: Scripts not executable or invalid shebang

**Solutions**:

1. Make scripts executable:
```bash
chmod +x .ramp/scripts/*.sh
```

2. Verify shebang line:
```bash
head -n 1 .ramp/scripts/setup.sh
# Should be: #!/bin/bash
```

3. Test script manually:
```bash
bash .ramp/scripts/setup.sh
```

## Git Issues

### "fatal: cannot clone repository"

**Cause**: Invalid git URL or authentication failure

**Solutions**:

1. Test clone manually:
```bash
git clone git@github.com:org/repo.git /tmp/test
```

2. Check SSH keys (for SSH URLs):
```bash
ssh -T git@github.com
```

3. Use HTTPS if SSH fails:
```yaml
repos:
  - path: repos
    git: https://github.com/org/repo.git  # Instead of git@github.com:org/repo.git
```

4. For private repos, ensure authentication:
```bash
# GitHub: use personal access token or SSH key
# GitLab: similar authentication options
```

### "fatal: 'branch' is already checked out"

**Cause**: Branch already in use by another worktree

**Solutions**:

1. Check existing worktrees:
```bash
cd repos/frontend
git worktree list
```

2. Remove conflicting worktree:
```bash
ramp down conflicting-feature
```

3. Use a different feature name:
```bash
ramp up feature-v2
```

### "error: cannot delete branch checked out at 'path'"

**Cause**: Trying to delete a branch that's checked out in a worktree

**Solutions**:

1. Use Ramp to clean up:
```bash
ramp down feature-name
```

2. Manual cleanup if needed:
```bash
cd repos/frontend
git worktree remove ../../trees/feature-name/frontend
git branch -D feature/feature-name
```

### Uncommitted Changes Warning

**Cause**: Feature has uncommitted changes that would be lost

**Solutions**:

1. Commit changes:
```bash
cd trees/feature-name/frontend
git add .
git commit -m "WIP: save progress"
```

2. Stash changes:
```bash
git stash push -m "WIP for feature-name"
```

3. Force removal (⚠️ loses changes):
```bash
ramp down feature-name
# Confirm when prompted
```

## Port Issues

### "Port already in use"

**Cause**: Another process using the port

**Solutions**:

1. Find process using port:
```bash
lsof -i :3000
```

2. Kill the process:
```bash
kill $(lsof -ti :3000)
```

3. Check for orphaned Docker containers:
```bash
docker ps -a | grep ramp
docker rm -f $(docker ps -a -q -f "name=ramp-")
```

4. Change base port:
```yaml
# .ramp/ramp.yaml
base_port: 4000  # Use different range
```

### "No available ports"

**Cause**: All ports in range allocated

**Solutions**:

1. Clean up merged features:
```bash
ramp prune
```

2. Remove old features:
```bash
ramp status  # See all features
ramp down old-feature-1
ramp down old-feature-2
```

3. Increase port range:
```yaml
# .ramp/ramp.yaml
max_ports: 200  # Increased from 100
```

4. Reset port allocations (⚠️ may cause conflicts):
```bash
rm .ramp/port_allocations.json
ramp status  # Regenerates
```

## Docker Issues

### "Cannot connect to Docker daemon"

**Cause**: Docker not running

**Solutions**:

1. Start Docker:
```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

2. Verify Docker is running:
```bash
docker ps
```

3. Check Docker permissions (Linux):
```bash
sudo usermod -aG docker $USER
# Log out and back in
```

### "bind: address already in use" (Docker)

**Cause**: Port conflict with Docker container

**Solutions**:

1. Find conflicting container:
```bash
docker ps | grep 3032
```

2. Stop container:
```bash
docker stop <container-id>
```

3. Remove all Ramp containers:
```bash
docker rm -f $(docker ps -a -q -f "name=ramp-")
```

## Script Issues

### Environment Variables Not Set

**Cause**: Script not receiving Ramp environment variables

**Solutions**:

1. Verify variables in script:
```bash
#!/bin/bash
echo "RAMP_PORT: $RAMP_PORT"
echo "RAMP_TREES_DIR: $RAMP_TREES_DIR"
env | grep RAMP
```

2. Test script manually with env vars:
```bash
export RAMP_PROJECT_DIR=/path/to/project
export RAMP_PORT=3000
export RAMP_WORKTREE_NAME=test
bash .ramp/scripts/setup.sh
```

3. Ensure using `ramp run`, not direct execution:
```bash
# ❌ Don't run directly
./ramp/scripts/dev.sh

# ✅ Use ramp run
ramp run dev
```

### "Permission denied" Error

**Cause**: Scripts not executable

**Solutions**:

```bash
chmod +x .ramp/scripts/*.sh
```

### Script Fails Silently

**Cause**: Missing `set -e` or error handling

**Solutions**:

1. Add error handling to scripts:
```bash
#!/bin/bash
set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure
```

2. Run with verbose mode:
```bash
ramp -v up my-feature
ramp -v run dev
```

3. Add debugging:
```bash
#!/bin/bash
set -x  # Print each command
```

## Performance Issues

### Slow `ramp up`

**Cause**: Large repositories or slow network

**Solutions**:

1. Disable auto-refresh for large repos:
```yaml
repos:
  - path: repos
    git: git@github.com:org/large-repo.git
    auto_refresh: false
```

2. Use `--no-refresh` flag:
```bash
ramp up my-feature --no-refresh
```

3. Check network speed:
```bash
git clone --progress git@github.com:org/repo.git /tmp/test
```

### Slow `ramp status`

**Cause**: Many repositories or slow git operations

**Solutions**:

1. Use verbose mode to see what's slow:
```bash
ramp -v status
```

2. Check git remote connectivity:
```bash
cd repos/frontend
git fetch --dry-run
```

## Debugging Tips

### Enable Verbose Mode

See all commands Ramp executes:

```bash
ramp -v up my-feature
ramp -v status
ramp -v down my-feature
```

### Check Ramp Version

```bash
ramp version
```

### Inspect Port Allocations

```bash
cat .ramp/port_allocations.json | jq .
```

### List All Worktrees

```bash
cd repos/frontend
git worktree list
```

### Check Git Status

```bash
cd trees/feature-name/frontend
git status
git log --oneline -10
git remote -v
```

### Verify Directory Structure

```bash
tree -L 3 -a
# Or without tree:
find . -maxdepth 3 -type d
```

## Getting Help

### Check Documentation

- [Getting Started](../getting-started.md) - Basic workflow
- [Configuration Reference](../configuration.md) - Config options
- [Command Reference](../commands/ramp.md) - All commands

### Report Issues

If you've found a bug:

1. Check existing issues: https://github.com/FreedomForeverSolar/ramp/issues
2. Include in your report:
   - Ramp version (`ramp version`)
   - OS and version
   - Go version (`go version`)
   - Error message and steps to reproduce
   - Relevant configuration (`.ramp/ramp.yaml`)

### Emergency Manual Cleanup

If Ramp commands fail and you need to clean up manually:

```bash
# 1. Stop all processes
pkill -f "ramp.*my-feature"

# 2. Stop Docker containers
docker stop $(docker ps -q -f "name=ramp-my-feature")
docker rm $(docker ps -a -q -f "name=ramp-my-feature")

# 3. Remove worktrees
cd repos/frontend
git worktree remove ../../trees/my-feature/frontend --force
git branch -D feature/my-feature
git worktree prune

# Repeat for each repository

# 4. Remove feature directory
rm -rf trees/my-feature

# 5. Edit port allocations
nano .ramp/port_allocations.json
# Remove the feature entry
```

## Prevention

### Best Practices

1. **Always use Ramp commands** - Don't manually manipulate worktrees
2. **Run `ramp status` regularly** - Keep track of active features
3. **Clean up finished features** - Use `ramp down` or `ramp prune`
4. **Use version control for scripts** - Commit `.ramp/` directory
5. **Test scripts in isolation** - Before using with Ramp
6. **Document custom workflows** - Add README in `.ramp/`
7. **Use verbose mode when debugging** - `ramp -v`

### Maintenance

```bash
# Weekly: Clean up merged features
ramp prune

# Monthly: Check for orphaned worktrees
cd repos/frontend
git worktree prune

# As needed: Reset port allocations
rm .ramp/port_allocations.json
ramp status
```

## Next Steps

- [Getting Started](../getting-started.md) - Learn the basics
- [Git Worktrees](worktrees.md) - Understand worktrees
- [Port Management](port-management.md) - Port allocation deep dive
