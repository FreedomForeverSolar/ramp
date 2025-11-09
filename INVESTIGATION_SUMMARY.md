# Investigation Summary: Stash Behavior in Ramp Refresh

## The Reported Issue

> "I had a stash in a feature tree then called ramp refresh and the repo inside my .repos folder had pulled in the stash after pulling in changes from main."

## What I Did

I created comprehensive tests to try to reproduce this issue, testing multiple scenarios:

1. **Basic stash + refresh** - Stash in worktree, then refresh
2. **Stash + autostash config** - With `pull.rebase=true` and `rebase.autoStash=true`  
3. **Stash + uncommitted changes** - Source repo has uncommitted changes during refresh
4. **Stash sharing verification** - Documented that stashes ARE shared across worktrees

## Test Results

**I could NOT reproduce the reported issue** in any of my test scenarios. The current `ramp refresh` implementation appears to be working correctly:

✅ All 10 refresh tests pass  
✅ Stashes from worktrees do NOT leak into source repos during refresh  
✅ Git's autostash behavior works correctly (LIFO - pops only the stash it creates)

## Key Finding: Stashes ARE Shared

While I couldn't reproduce the issue, I **confirmed an important fact**:

**Git stashes are shared across all worktrees** because they're stored in `.git/refs/stash` which is shared, not per-worktree.

### What This Means

```bash
# In a worktree:
cd trees/my-feature/repo1
echo "test" > file.txt
git add .
git stash push -m "worktree changes"

# In the source repo:
cd repos/repo1
git stash list
# Output: stash@{0}: On feature/my-feature: worktree changes
# ☝️ The stash is visible here!

git stash pop  # ⚠️ This would apply the worktree's stash!
```

This is a potential "footgun" - users could accidentally apply worktree stashes if they run `git stash pop` in the source repo.

## How This Could Happen

Since I couldn't reproduce it with `ramp refresh`, here are scenarios where the reported issue might occur:

1. **User error**: Accidentally running `git stash pop` in source repo
2. **Git aliases/hooks**: Custom git configuration that auto-applies stashes
3. **Manual intervention**: User manually resolving a failed pull with stash commands
4. **Specific git version**: Possible bug in older git versions with worktree + autostash

## Current `ramp refresh` Implementation

The current code is safe:

```go
func Pull(repoDir string) error {
	cmd := exec.Command("git", "pull")  // Uses default git behavior
	// ...
}
```

This respects user's git config and git's autostash behavior works correctly (LIFO stash stack).

## Recommendations

### Option 1: Keep Current Implementation (✅ Recommended)

**What to do:**
- Keep the current implementation as-is
- Document the shared stash behavior
- Add a note to README/docs about this limitation

**Rationale:**
- Current implementation is working correctly
- All tests pass
- Respects user's git configuration
- No reproducible bug

### Option 2: Make Refresh More Defensive

If this issue gets reported again with reproducible steps, consider:

```go
func Pull(repoDir string) error {
	// Check for uncommitted changes first
	hasChanges, err := HasUncommittedChanges(repoDir)
	if err != nil {
		return err
	}
	if hasChanges {
		return fmt.Errorf("repository has uncommitted changes, please commit or stash them first")
	}
	
	// Use explicit fetch + ff-only merge instead of pull
	if err := FetchAll(repoDir); err != nil {
		return err
	}
	
	cmd := exec.Command("git", "merge", "--ff-only", "@{u}")
	// ...
}
```

**Pros:**
- Forces clean state
- No autostash surprises
- Explicit about what's happening

**Cons:**
- Changes existing behavior
- Less flexible for users who work with WIP in source repos

### Option 3: Add `--no-autostash` Flag

```go
func Pull(repoDir string) error {
	cmd := exec.Command("git", "pull", "--no-autostash")
	// ...
}
```

**Pros:**
- Prevents any autostash-related edge cases

**Cons:**
- Breaks workflows for users who rely on autostash
- Doesn't solve the root issue (stashes are still shared)

## Documentation to Add

Consider adding to `CLAUDE.md` or README:

```markdown
### Important: Git Stashes and Worktrees

Git stashes are **shared across all worktrees** of the same repository. 
When you create a stash in a feature tree, it's visible in the source 
repository and other worktrees.

Be careful when using `git stash pop` in source repositories - you might 
accidentally apply stashes created in feature trees.

Best practice: Use `git stash push -m "descriptive message"` and check 
`git stash list` before popping to ensure you're applying the right stash.
```

## Conclusion

**The issue is NOT reproducible** with the current implementation. The `ramp refresh` 
command behaves correctly and safely. 

**However, git stashes ARE shared**, which is a potential footgun users should be 
aware of.

**My recommendation**: Keep the current implementation and add documentation about 
the shared stash limitation. If users report this again, ask for specific reproduction 
steps to identify the actual cause.

## Test Coverage

Added comprehensive test coverage in `cmd/refresh_test.go`:

- `TestRefreshWithStashInWorktree` - Basic scenario
- `TestRefreshWithStashAndAutoStashConfig` - With autostash enabled  
- `TestStashesAreSharedAcrossWorktrees` - Documents the sharing behavior

All tests pass ✅
