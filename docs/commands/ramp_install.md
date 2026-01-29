## ramp install

Clone all configured repositories from ramp.yaml

### Synopsis

Clone all repositories specified in the .ramp/ramp.yaml configuration file
into their configured locations.

This command must be run from within a directory containing a .ramp/ramp.yaml file.

```
ramp install [flags]
```

### Options

```
  -h, --help      help for install
      --shallow   Perform a shallow clone (--depth 1) to reduce clone time and disk usage
```

### Options inherited from parent commands

```
  -v, --verbose   Show detailed output during operations
  -y, --yes       Non-interactive mode: skip prompts and auto-confirm
```

### SEE ALSO

* [ramp](ramp.md)	 - A CLI tool for managing multi-repo development workflows

