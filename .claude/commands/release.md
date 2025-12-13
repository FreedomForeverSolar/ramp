---
allowed-tools: Bash(git describe:*), Bash(git tag:*), Bash(git push:*)
argument-hint: <cli|desktop> [major|minor|patch]
description: Create and push a new semver git tag
model: claude-haiku-4-5-20251001
---

## Current State

- Latest CLI tag: !`git describe --tags --abbrev=0 --match "v[0-9]*" 2>/dev/null || echo "v1.0.0"`
- Latest Desktop tag: !`git describe --tags --abbrev=0 --match "ui-v[0-9]*" 2>/dev/null || echo "ui-v0.0.0"`

## Your Task

Create a new git tag based on semantic versioning and push it to the remote.

**App**: $1 (required - must be "cli" or "desktop")
**Increment type**: $2 (defaults to "patch" if not provided)

**Tag formats**:
- **cli** - Uses format `vX.Y.Z` (e.g., v1.2.3)
- **desktop** - Uses format `ui-vX.Y.Z` (e.g., ui-v0.0.3)

**Steps**:
1. Validate that $1 is either "cli" or "desktop". If not provided or invalid, ask the user.
2. Parse the latest tag for the selected app to extract the version numbers
3. Increment the appropriate version component:
   - `major`: increment X, reset Y and Z to 0 (e.g., v1.2.3 -> v2.0.0)
   - `minor`: increment Y, reset Z to 0 (e.g., v1.2.3 -> v1.3.0)
   - `patch`: increment Z (e.g., v1.2.3 -> v1.2.4)
4. Create the new tag: `git tag <prefix>vX.Y.Z`
5. Push the tag to origin: `git push origin <prefix>vX.Y.Z`

Confirm the new version before creating and pushing the tag.
