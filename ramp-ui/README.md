# Ramp UI

A native desktop application for Ramp, providing a graphical interface for managing multi-repository development workflows.

## Architecture

The app uses a hybrid architecture:
- **Backend**: Go HTTP server (`cmd/ramp-ui`) that reuses the existing `internal/` packages from the Ramp CLI
- **Frontend**: Electron + React + TypeScript + Vite

This ensures the UI uses the same battle-tested logic as the CLI with zero code duplication.

## Development Setup

### Prerequisites

- Go 1.24+
- Bun 1.0+

### Backend

The backend is part of the main Ramp module to enable direct imports from `internal/` packages.

```bash
# From the ramp root directory
cd /path/to/ramp

# Build the backend binary
go build -o ramp-ui/frontend/resources/ramp-server ./cmd/ramp-ui
```

### Frontend

```bash
cd ramp-ui/frontend

# Install dependencies
bun install

# Build the Electron main process (required once, and after changes to src/main/)
bun run build:electron

# Start development mode (hot reload)
bun run dev
```

This will:
1. Start the Vite dev server on port 5173
2. Launch Electron which spawns the Go backend
3. Connect to the backend API on port 37429

**Note**: The `build:electron` step compiles the Electron main process TypeScript. You only need to re-run it if you modify files in `src/main/`.

### Building for Production

```bash
# From the ramp root directory

# Build backend binary
go build -o ramp-ui/frontend/resources/ramp-server ./cmd/ramp-ui

# Build Electron app
cd ramp-ui/frontend
bun run build
bun run package
```

Distributable files will be in `ramp-ui/frontend/release/`.

## Project Structure

```
ramp/
├── cmd/
│   └── ramp-ui/                # Backend entry point
│       └── main.go             # HTTP server main
│
├── internal/
│   └── uiapi/                  # UI API handlers
│       ├── server.go           # Server struct and setup
│       ├── projects.go         # Project endpoints
│       ├── features.go         # Feature endpoints
│       ├── websocket.go        # Real-time updates
│       ├── models.go           # API data types
│       ├── appconfig.go        # App configuration storage
│       └── utils.go            # Helper functions
│
└── ramp-ui/
    ├── frontend/               # Electron + React app
    │   ├── src/
    │   │   ├── main/           # Electron main process
    │   │   ├── renderer/       # React app
    │   │   │   ├── App.tsx
    │   │   │   ├── components/
    │   │   │   ├── hooks/
    │   │   │   ├── types/
    │   │   │   └── styles/
    │   │   └── preload/        # Preload script
    │   ├── package.json
    │   └── electron-builder.yml
    │
    └── shared/                 # Shared type definitions
        └── types.ts
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/projects` | List all projects |
| POST | `/api/projects` | Add a new project |
| GET | `/api/projects/:id` | Get project details |
| DELETE | `/api/projects/:id` | Remove project from UI |
| GET | `/api/projects/:id/features` | List features |
| POST | `/api/projects/:id/features` | Create feature (ramp up) |
| DELETE | `/api/projects/:id/features/:name` | Delete feature (ramp down) |
| WS | `/ws/logs` | WebSocket for real-time updates |
| GET | `/health` | Health check |

## Configuration

App configuration is stored in platform-specific locations:
- **macOS**: `~/Library/Application Support/ramp-ui/config.json`
- **Linux**: `~/.config/ramp-ui/config.json`
- **Windows**: `%APPDATA%/ramp-ui/config.json`
