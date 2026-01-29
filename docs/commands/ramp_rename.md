## ramp rename

Set or change the display name of a feature

### Synopsis

Set or change the human-readable display name of a feature.

The display name is shown in status output and the UI as an alternative
to the technical feature identifier (directory/branch name).

Pass an empty string to clear the display name.

Examples:
  ramp rename my-feature "User Authentication Feature"
  ramp rename my-feature ""  # Clear display name

```
ramp rename <feature> <display-name> [flags]
```

### Options

```
  -h, --help   help for rename
```

### Options inherited from parent commands

```
  -v, --verbose   Show detailed output during operations
  -y, --yes       Non-interactive mode: skip prompts and auto-confirm
```

### SEE ALSO

* [ramp](ramp.md)	 - A CLI tool for managing multi-repo development workflows

