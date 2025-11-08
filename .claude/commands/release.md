---
allowed-tools: Bash(git describe:*), Bash(git tag:*), Bash(git push:*)
argument-hint: [major|minor|patch]
description: Create and push a new semver git tag
model: claude-haiku-4-5-20251001
---

## Current State

- Latest git tag: !`git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.0"`

## Your Task

Create a new git tag based on semantic versioning and push it to the remote.

**Increment type**: $1 (defaults to "patch" if not provided)

**Steps**:
1. Parse the latest tag to extract the version numbers (format: vX.Y.Z)
2. Increment the appropriate version component:
   - `major`: increment X, reset Y and Z to 0 (e.g., v1.2.3 � v2.0.0)
   - `minor`: increment Y, reset Z to 0 (e.g., v1.2.3 � v1.3.0)
   - `patch`: increment Z (e.g., v1.2.3 � v1.2.4)
3. Create the new tag: `git tag vX.Y.Z`
4. Push the tag to origin: `git push origin vX.Y.Z`

Confirm the new version before creating and pushing the tag.
