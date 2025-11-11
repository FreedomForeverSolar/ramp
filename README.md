# Ramp

**Work on features across multiple repositories simultaneously using git worktrees and automated workflows.**

```bash
ramp init          # Initialize multi-repo project
ramp up my-feature # Create feature branches across all repos
ramp status        # View all active features
ramp down my-feature # Clean up when done
```

## Why Ramp?

Modern applications span multiple repositories (microservices, frontend/backend, libraries). Ramp automates the entire development workflow:

- 🚀 **One command setup** - Create branches across all repos simultaneously
- 🔄 **Git worktrees** - Work on multiple features in parallel without branch switching
- 🎯 **Port management** - Automatic port allocation prevents conflicts
- 📦 **Environment automation** - Custom scripts handle dependencies and services
- 🧹 **Batch cleanup** - Remove all merged features with `ramp prune`

## Quick Start

### Install

**Homebrew** (macOS/Linux):
```bash
brew install freedomforeversolar/tools/ramp
```

**Pre-built binaries**: Download from [releases](https://github.com/FreedomForeverSolar/ramp/releases)

**From source**:
```bash
git clone https://github.com/FreedomForeverSolar/ramp.git
cd ramp
go build -o ramp .
sudo ./install.sh
```

### Try the Demo

```bash
cd demo/demo-microservices-app
ramp install      # Clone demo repositories
ramp up my-feature # Create feature across all repos
ramp status       # View status
ramp down my-feature # Clean up
```

### Create Your Project

```bash
mkdir my-project && cd my-project
ramp init         # Interactive setup
ramp up my-feature # Start coding!
```

## How It Works

Ramp creates isolated workspaces for each feature using git worktrees:

```
my-project/
├── repos/              # Main repository clones
│   ├── frontend/
│   └── api/
└── trees/              # Feature workspaces (where you work)
    ├── feature-a/
    │   ├── frontend/   # Worktree on branch feature/feature-a
    │   └── api/        # Worktree on branch feature/feature-a
    └── feature-b/
        ├── frontend/
        └── api/
```

Each feature gets:
- Dedicated branches in all repositories
- Isolated working directories (git worktrees)
- Unique port allocation
- Automated setup (install deps, start services, run migrations)
- Automated cleanup

## Core Commands

| Command | Description |
|---------|-------------|
| `ramp init` | Initialize a new multi-repo project |
| `ramp install` | Clone all configured repositories |
| `ramp up <feature>` | Create feature branches across all repos |
| `ramp down <feature>` | Remove feature branches and cleanup |
| `ramp prune` | Batch remove all merged features |
| `ramp status` | Show project status and active features |
| `ramp run <cmd>` | Run custom commands (dev, test, etc.) |

See [docs/commands/](docs/commands/) for detailed command reference.

## Configuration

Ramp uses `.ramp/ramp.yaml`:

```yaml
name: my-project

repos:
  - path: repos
    git: git@github.com:org/frontend.git
  - path: repos
    git: git@github.com:org/api.git

setup: scripts/setup.sh      # Run after 'ramp up'
cleanup: scripts/cleanup.sh  # Run before 'ramp down'

default-branch-prefix: feature/

base_port: 3000  # Allocate one unique port per feature
max_ports: 100

commands:
  - name: dev
    command: scripts/dev.sh
```

Scripts receive environment variables for automation:
- `RAMP_PORT` - Unique port for this feature
- `RAMP_TREES_DIR` - Feature workspace path
- `RAMP_REPO_PATH_<NAME>` - Path to each repository

See [docs/configuration.md](docs/configuration.md) for full reference.

## Example Setup Script

```bash
#!/bin/bash
# .ramp/scripts/setup.sh

# Install dependencies
cd "$RAMP_TREES_DIR/frontend"
npm install

cd "$RAMP_TREES_DIR/api"
npm install

# Start database on feature-specific port
docker run -d \
  --name "db-${RAMP_WORKTREE_NAME}" \
  -p "$((RAMP_PORT + 32)):5432" \
  postgres:15

# Run migrations
cd "$RAMP_TREES_DIR/api"
npm run migrate

echo "✅ Ready! Run 'ramp run dev' to start"
```

## Documentation

- **[Getting Started](docs/getting-started.md)** - Your first Ramp project in 5 minutes
- **[Configuration Reference](docs/configuration.md)** - Complete ramp.yaml guide
- **[Command Reference](docs/commands/)** - Detailed command documentation
- **[How-To Guides](docs/guides/)** - Microservices, frontend/backend, custom scripts
- **[Advanced Topics](docs/advanced/)** - Port management, git worktrees, troubleshooting

## Use Cases

**Microservices Development**
Coordinate features across multiple services with shared databases and networking

**Frontend/Backend Projects**
Develop full-stack features requiring changes to both repos simultaneously

**Library Development**
Work on libraries alongside applications that consume them with live linking

**Multi-Environment Testing**
Set up isolated environments for testing features without affecting main development

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Development
go test ./...            # Run tests
go build -o ramp .       # Build binary
./install.sh             # Install locally
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- 📖 [Documentation](docs/)
- 🐛 [Report Issues](https://github.com/FreedomForeverSolar/ramp/issues)
- 💬 [Discussions](https://github.com/FreedomForeverSolar/ramp/discussions)

---

**Get started now**: `brew install freedomforeversolar/tools/ramp` or try the demo in `demo/demo-microservices-app/`
