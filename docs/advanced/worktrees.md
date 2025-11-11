# Git Worktrees

This guide explains how Ramp uses git worktrees to enable working on multiple features simultaneously.

## What are Git Worktrees?

Git worktrees allow you to check out multiple branches from the same repository at once. Each worktree is a separate working directory linked to the main repository.

### Traditional Git Workflow

```bash
git checkout feature-a    # Switch to feature-a
# Work on feature-a
git checkout feature-b    # Switch away (must commit or stash)
# Work on feature-b
```

**Problems:**
- Only one branch active at a time
- Must commit or stash changes before switching
- Slow context switching
- Can't run two features simultaneously

### Worktree Workflow

```bash
git worktree add ../feature-a feature-a  # Checkout feature-a
git worktree add ../feature-b feature-b  # Checkout feature-b
# Both branches available simultaneously
```

**Benefits:**
- Multiple branches checked out at once
- No need to commit/stash when switching
- Fast context switching (just `cd`)
- Can run multiple features in parallel

## How Ramp Uses Worktrees

### Directory Structure

```
my-project/
├── repos/              # Main repository clones
│   ├── frontend/       # Main working directory
│   └── api/
└── trees/              # Feature worktrees
    ├── feature-a/
    │   ├── frontend/   # Worktree linked to repos/frontend
    │   └── api/        # Worktree linked to repos/api
    └── feature-b/
        ├── frontend/
        └── api/
```

### What Happens During `ramp up`

```bash
ramp up feature-a
```

For each repository, Ramp executes:

```bash
cd repos/frontend
git worktree add ../../trees/feature-a/frontend feature/feature-a

cd repos/api
git worktree add ../../trees/feature-a/api feature/feature-a
```

This creates:
- `trees/feature-a/frontend/` - Working directory for frontend on branch `feature/feature-a`
- `trees/feature-a/api/` - Working directory for api on branch `feature/feature-a`

### What Happens During `ramp down`

```bash
ramp down feature-a
```

For each repository, Ramp executes:

```bash
cd repos/frontend
git worktree remove ../../trees/feature-a/frontend --force
git branch -D feature/feature-a

cd repos/api
git worktree remove ../../trees/feature-a/api --force
git branch -D feature/feature-a
```

## How Worktrees Share Data

### Shared Between Worktrees

All worktrees of the same repository share:

- **Git object database** (`.git/objects/`) - All commits, trees, blobs
- **Git configuration** (`.git/config`)
- **Remotes** (`.git/refs/remotes/`)
- **Stashes** (`.git/refs/stash`) - ⚠️ **Important limitation!**
- **Hooks** (`.git/hooks/`)

### Separate Per Worktree

Each worktree has its own:

- **Working directory** - Files on disk
- **Index** (staging area)
- **HEAD** (current branch/commit)
- **Checked out branch**

### The Stash Gotcha

**⚠️ Stashes are shared across all worktrees!**

```bash
# In trees/feature-a/frontend
git stash push -m "WIP feature-a changes"

# In repos/frontend
git stash list
# Shows: stash@{0}: On feature/feature-a: WIP feature-a changes

git stash pop  # ⚠️ Accidentally applies feature-a changes to main!
```

**Best practices:**

1. **Use descriptive stash messages**:
```bash
git stash push -m "feature-a: WIP authentication"
```

2. **Always check `git stash list` before popping**:
```bash
git stash list
# Verify you're popping the right stash
git stash pop stash@{0}
```

3. **Prefer commits over stashes**:
```bash
git add .
git commit -m "WIP: work in progress"
# Later: git reset HEAD~1
```

## Working with Worktrees

### Making Changes

Each worktree is a fully functional git repository:

```bash
cd trees/feature-a/frontend

# Normal git operations
git status
git add .
git commit -m "Add feature"
git push origin feature/feature-a
git pull
```

### Switching Between Features

```bash
# Work on feature-a
cd trees/feature-a/frontend
npm run dev

# Switch to feature-b (no git checkout needed!)
cd trees/feature-b/frontend
npm run dev  # Both can run simultaneously!
```

### Checking Worktree Status

```bash
# In main repository
cd repos/frontend
git worktree list

# Output:
# /path/to/repos/frontend              abc123 [main]
# /path/to/trees/feature-a/frontend    def456 [feature/feature-a]
# /path/to/trees/feature-b/frontend    789abc [feature/feature-b]
```

### Syncing with Remote

Changes in worktrees are immediately visible in the main repository:

```bash
# In worktree
cd trees/feature-a/frontend
git commit -m "Add feature"

# In main repo
cd repos/frontend
git log feature/feature-a  # Shows the commit
```

## Advanced Worktree Operations

### Creating Worktrees from Specific Branches

Ramp's `--target` flag uses this:

```bash
ramp up new-feature --target existing-feature
```

This runs:
```bash
git worktree add trees/new-feature/frontend feature/new-feature feature/existing-feature
#                                           ↑ new branch       ↑ source branch
```

### Pruning Stale Worktrees

If a worktree directory is deleted manually (outside of Ramp), git keeps a reference:

```bash
cd repos/frontend
git worktree prune  # Clean up orphaned worktree registrations
```

Ramp does this automatically during `ramp down`.

### Moving Worktrees

You can move worktree directories:

```bash
mv trees/feature-a trees/feature-a-backup

cd repos/frontend
git worktree repair  # Update worktree paths
```

**Better approach**: Use Ramp commands instead of manual operations.

## Limitations and Considerations

### Cannot Checkout Same Branch Twice

```bash
# This fails:
git worktree add trees/test-1/frontend feature/shared
git worktree add trees/test-2/frontend feature/shared
# Error: 'feature/shared' is already checked out at 'trees/test-1/frontend'
```

**Solution**: Create separate branches:
```bash
ramp up test-1 --target shared-feature
ramp up test-2 --target shared-feature
# Creates feature/test-1 and feature/test-2 from shared-feature
```

### Disk Space Usage

Each worktree requires disk space for its working directory, but:
- Git objects (commits, blobs) are shared (deduplicated)
- Only one copy of each file version is stored
- Working directories contain actual files

**Example**:
- Main repo: 100 MB
- 3 worktrees: ~300 MB total (3x working directories)
- Shared objects: still only 100 MB

### Performance

Worktrees are very fast because:
- No need to checkout/switch branches
- Shared git object database
- No copying of git history

### Hooks Run Per-Worktree

Git hooks run in the context of each worktree:

```bash
# .git/hooks/pre-commit runs separately in:
# - repos/frontend
# - trees/feature-a/frontend
# - trees/feature-b/frontend
```

## Integration with Ramp Features

### Auto-Refresh and Worktrees

When `auto_refresh: true`, Ramp updates the main repository before creating worktrees:

```bash
cd repos/frontend
git fetch --all
git pull  # Update main repo

# Then create worktrees
git worktree add trees/feature-a/frontend feature/feature-a
```

### Branch Detection

Ramp uses worktrees intelligently:

1. **Local branch exists**: Uses existing branch
```bash
git worktree add trees/feature-a/frontend feature/feature-a
```

2. **Remote branch exists**: Creates tracking branch
```bash
git worktree add trees/feature-a/frontend -b feature/feature-a origin/feature/feature-a
```

3. **No branch exists**: Creates new branch
```bash
git worktree add trees/feature-a/frontend -b feature/feature-a
```

### Cleanup and Worktrees

`ramp down` ensures proper cleanup:

```bash
# 1. Remove worktree
git worktree remove trees/feature-a/frontend --force

# 2. Delete branch
git branch -D feature/feature-a

# 3. Prune stale references
git fetch --prune

# 4. Clean up worktree metadata
git worktree prune
```

## Troubleshooting

### "fatal: 'path' is already locked"

**Cause**: Git lock file exists from a crashed operation

**Solution**:
```bash
rm repos/frontend/.git/worktrees/*/locked
```

### "fatal: cannot stat 'path': No such file or directory"

**Cause**: Worktree directory deleted manually

**Solution**:
```bash
cd repos/frontend
git worktree prune
```

### "error: cannot delete branch checked out at 'path'"

**Cause**: Trying to delete a branch that's checked out in a worktree

**Solution**:
```bash
# Remove the worktree first
git worktree remove path/to/worktree

# Then delete the branch
git branch -D branch-name
```

### Changes Not Showing in Main Repo

**Cause**: Need to refresh

**Solution**:
```bash
cd repos/frontend
git fetch .  # Fetch from local worktrees
```

## Best Practices

1. **Always use Ramp commands** - Don't manually create/remove worktrees
2. **Use descriptive stash messages** - Remember stashes are shared
3. **Commit before switching contexts** - Easier than managing stashes
4. **Run `ramp status` regularly** - See all active worktrees
5. **Clean up finished features** - Use `ramp down` or `ramp prune`
6. **Avoid manual worktree operations** - Let Ramp handle it
7. **Use `--target` for feature branches** - Creates proper worktree relationships

## Further Reading

- [Official Git Worktree Documentation](https://git-scm.com/docs/git-worktree)
- [Git Worktree Tutorial](https://git-scm.com/book/en/v2/Git-Tools-Advanced-Merging)

## Next Steps

- [Getting Started](../getting-started.md) - Try worktrees with Ramp
- [Troubleshooting](troubleshooting.md) - Solve common issues
- [Configuration](../configuration.md) - Configure your project
