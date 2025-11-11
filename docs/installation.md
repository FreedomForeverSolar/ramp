# Installation

This guide covers all the ways to install Ramp on your system.

## Prerequisites

- **Go** 1.21+ (for building from source)
- **Git** 2.25+ (for worktree support)

## Homebrew (macOS/Linux)

The easiest way to install Ramp on macOS or Linux:

```bash
brew install freedomforeversolar/tools/ramp
```

Verify installation:
```bash
ramp --help
```

### Upgrading

```bash
brew update                                  # Update Homebrew and all taps
brew upgrade freedomforeversolar/tools/ramp  # Upgrade ramp to latest version
```

## Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/FreedomForeverSolar/ramp/releases):

1. Download the appropriate archive for your OS and architecture
2. Extract the binary
3. Move it to a directory in your PATH (e.g., `/usr/local/bin`)
4. Make it executable: `chmod +x ramp`

### Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Build from Source

### Clone and Build

```bash
git clone https://github.com/FreedomForeverSolar/ramp.git
cd ramp
go build -o ramp .
```

### Install Globally

```bash
sudo ./install.sh
```

This installs the binary to `/usr/local/bin/ramp`.

### Local Development

For development or testing without installing:

```bash
go run . --help
```

## Verification

After installation, verify Ramp is working:

```bash
ramp version
ramp --help
```

## Next Steps

- [Getting Started Guide](getting-started.md) - Create your first Ramp project
- [Configuration Reference](configuration.md) - Learn about ramp.yaml
