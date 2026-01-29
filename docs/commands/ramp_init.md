## ramp init

Initialize a new ramp project with interactive setup

### Synopsis

Initialize a new ramp project by creating the necessary configuration
files and directory structure through an interactive setup process.

This is similar to 'npm init' - it will guide you through creating a
.ramp/ramp.yaml configuration file, .gitignore file, and optional setup scripts.

The .gitignore file is automatically created with entries for ramp-managed files:
repos/, trees/, .ramp/local.yaml, .ramp/port_allocations.json, and .ramp/feature_metadata.json.

After initialization, use 'ramp install' to clone the configured repositories.

```
ramp init [flags]
```

### Options

```
  -h, --help   help for init
```

### Options inherited from parent commands

```
  -v, --verbose   Show detailed output during operations
  -y, --yes       Non-interactive mode: skip prompts and auto-confirm
```

### SEE ALSO

* [ramp](ramp.md)	 - A CLI tool for managing multi-repo development workflows

