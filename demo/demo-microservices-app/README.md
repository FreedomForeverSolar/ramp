# Demo Microservices App

This is a demonstration project for **Ramp**, showing how to manage multi-repository development workflows using git worktrees.

## Overview

This demo simulates a modern development environment with three popular tools:

- **Frontend**: JSON Server (72k+ stars) - Fake REST API for frontend development
- **API Gateway**: Node TypeScript Boilerplate (2.6k+ stars) - Modern Node.js backend starter
- **Auth Service**: GitHub's Hello-World (Official example repository)

Each service runs on its own port and demonstrates how Ramp manages:
- Multi-repository cloning and worktree creation
- Environment variable injection
- Port allocation with our improved port range strategy
- Setup and cleanup automation

## Prerequisites

Before running this demo, ensure you have:

- Git installed and configured
- Ramp CLI built and available (see main README)
- Basic familiarity with command line

## Quick Start

1. **Navigate to the demo directory:**
   ```bash
   cd demo/demo-microservices-app
   ```

2. **Install repositories:**
   ```bash
   ramp install
   ```
   This clones all three repositories into the `repos/` directory.

3. **Create a feature branch:**
   ```bash
   ramp up my-feature
   ```
   This creates worktrees for all repositories with branch `feature/my-feature`.

4. **Start the development environment:**
   ```bash
   ramp run dev
   ```
   This simulates starting all microservices.

5. **Check the status:**
   ```bash
   ramp status
   ```
   Shows project status and active feature branches.

6. **Open in VS Code (optional):**
   ```bash
   ramp run open
   ```

7. **View logs:**
   ```bash
   ramp run logs
   ```

8. **Clean up when done:**
   ```bash
   ramp down my-feature
   ```
   This removes worktrees, branches, and cleans up all resources.

## What Happens Behind the Scenes

### During `ramp up my-feature`:

1. **Port Allocation**: Assigns one base port (e.g., 3000) to your feature
2. **Worktree Creation**: Creates isolated git worktrees in `trees/my-feature/`
3. **Branch Management**: Creates `feature/my-feature` branch in each repo
4. **Setup Script**: Runs `.ramp/scripts/setup.sh` which:
   - Creates `.env` files for each service
   - Sets up port configurations
   - Creates demo documentation

### During `ramp down my-feature`:

1. **Cleanup Script**: Runs `.ramp/scripts/cleanup.sh` which:
   - Stops any running services
   - Removes environment files
   - Cleans up temporary files
2. **Worktree Removal**: Removes all worktrees
3. **Branch Cleanup**: Deletes feature branches
4. **Port Release**: Frees allocated ports for reuse

## Port Allocation Strategy

**Important**: Ramp allocates **exactly one port per feature**, not multiple ports. For multi-service applications, this demo uses a port range strategy:

### How It Works
- **Base Port**: Ramp assigns one port (e.g., `RAMP_PORT=3000`)
- **Service Ports**: Additional services append digits to avoid conflicts:
  - Frontend: `3000` (base port)
  - API Gateway: `30001` (base + "1")
  - Auth Service: `30002` (base + "2")

### Why This Strategy?
- **Avoids Conflicts**: Multiple features can run simultaneously
- **Predictable**: Easy to calculate service ports from base port
- **Safe**: Wide port spacing prevents overlap

### Examples
- **Feature A** (port 3000): Frontend=3000, API=30001, Auth=30002
- **Feature B** (port 3001): Frontend=3001, API=30011, Auth=30012
- **Feature C** (port 3002): Frontend=3002, API=30021, Auth=30022

### Alternative Approaches
In production environments, you might use:
- Docker Compose with internal networking
- Service mesh (Istio, Linkerd)
- Reverse proxy (nginx, Traefik)
- Custom port allocation logic in your setup scripts

## Directory Structure After Setup

```
demo-microservices-app/
├── .ramp/
│   ├── ramp.yaml              # Project configuration
│   └── scripts/               # Setup, cleanup, and custom commands
├── repos/                     # Source repositories (after ramp install)
│   ├── json-server/           # JSON Server repo clone
│   ├── node-typescript-boilerplate/  # Node TypeScript boilerplate clone
│   └── Hello-World/           # Hello-World repo clone
└── trees/                     # Feature worktrees (after ramp up)
    └── my-feature/
        ├── json-server/       # JSON Server worktree
        ├── node-typescript-boilerplate/  # Node TypeScript worktree
        └── Hello-World/       # Hello-World worktree
```

## Environment Variables

When scripts run, they receive these environment variables:

- `RAMP_PROJECT_DIR`: Absolute path to project root
- `RAMP_TREES_DIR`: Path to current feature's trees directory
- `RAMP_WORKTREE_NAME`: Feature name (e.g., "my-feature")
- `RAMP_PORT`: Base port number (e.g., 3000)
- `RAMP_REPO_PATH_JSON_SERVER`: Path to json-server repository
- `RAMP_REPO_PATH_NODE_TYPESCRIPT_BOILERPLATE`: Path to node-typescript-boilerplate repository
- `RAMP_REPO_PATH_HELLO_WORLD`: Path to Hello-World repository

## Custom Commands

The demo includes several custom commands:

- `ramp run dev`: Start all development services
- `ramp run open`: Open the project in VS Code
- `ramp run logs`: View service logs
- `ramp run --help`: List all available commands

## Learning Exercises

Try these exercises to understand Ramp better:

1. **Multiple Features**: Create multiple features and see port allocation
   ```bash
   ramp up feature-a
   ramp up feature-b
   ramp status  # See both features with different ports
   ```

2. **Branch Switching**: Switch source repositories to different branches
   ```bash
   ramp rebase canary  # Switch to canary branch
   ramp status         # See updated branch info
   ```

3. **Repository Updates**: Pull latest changes
   ```bash
   ramp refresh
   ```

4. **Custom Scripts**: Examine and modify the scripts in `.ramp/scripts/`

## Real-World Usage

This demo uses popular open-source repositories for demonstration. In a real project, you would:

- Use your actual project repositories (microservices, frontend/backend, etc.)
- Implement real setup scripts that install dependencies and configure services
- Configure actual service startup in custom commands
- Add database setup, API keys, and other environment configuration
- The repos we chose represent common development tools:
  - **JSON Server**: Mock APIs for frontend development
  - **TypeScript Boilerplate**: Modern backend service templates
  - **Hello-World**: Simple service examples

## Next Steps

- Explore the main Ramp CLI documentation in the root README
- Try creating your own multi-repository project
- Customize the scripts for your development workflow