## ramp rebase

Switch all source repositories to the specified branch

### Synopsis

Switch all source repositories in the project to the specified branch.
The branch can exist locally, remotely, or both. If the branch doesn't exist 
in any repository, the command will fail.

The operation is atomic - if any repository fails to switch, all repositories 
will be reverted to their original branches.

If there are uncommitted changes, you will be prompted to stash them.

```
ramp rebase <branch-name> [flags]
```

### Options

```
  -h, --help   help for rebase
```

### Options inherited from parent commands

```
  -v, --verbose   Show detailed output during operations
  -y, --yes       Non-interactive mode: skip prompts and auto-confirm
```

### SEE ALSO

* [ramp](ramp.md)	 - A CLI tool for managing multi-repo development workflows

