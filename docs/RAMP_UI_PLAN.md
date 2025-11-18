# Ramp UI - Native Desktop App Plan

## Overview

A native desktop application that provides a graphical interface for interacting with the Ramp CLI, built with Electron, React, and TypeScript while reusing the existing Go codebase.

## Target Audience

Users who prefer visual interfaces over command-line tools, making Ramp more accessible to developers less familiar with CLI workflows.

## Architecture

### Hybrid Approach: Go Backend + Electron Frontend

**Why This Architecture?**
- Maximum code reuse via direct imports from existing `internal/` packages
- Real-time updates via WebSocket for command execution feedback
- Clean separation of concerns
- Type-safe API contracts
- Better error handling and progress feedback
- No code duplication - the same battle-tested logic powers both CLI and UI

### Components

**Backend: Go HTTP Server**
- Lightweight Go HTTP/WebSocket server
- Imports and reuses all existing `internal/` packages
- Exposes REST API endpoints for all Ramp operations
- Runs as a subprocess managed by Electron
- Streams real-time command output via WebSocket

**Frontend: Electron + React + TypeScript**
- Modern React UI with TypeScript for type safety
- Electron main process spawns and manages the Go backend
- Communicates with backend via HTTP/WebSocket
- Bundled as a single native app for macOS/Windows/Linux

## Directory Structure

```
ramp-ui/
├── backend/                    # Go HTTP server
│   ├── main.go                 # HTTP server entry point
│   ├── api/                    # API handlers
│   │   ├── projects.go         # List/add/remove projects
│   │   ├── features.go         # Up/down/list features
│   │   ├── commands.go         # Run custom commands
│   │   ├── git.go              # Git status operations
│   │   └── websocket.go        # Real-time updates
│   ├── models/                 # API request/response types
│   └── go.mod                  # References ../internal
│
├── frontend/                   # Electron + React app
│   ├── public/
│   ├── src/
│   │   ├── main/               # Electron main process
│   │   │   ├── index.ts        # Main entry, spawns Go backend
│   │   │   └── ipc.ts          # IPC handlers
│   │   ├── renderer/           # React app
│   │   │   ├── App.tsx
│   │   │   ├── components/
│   │   │   │   ├── ProjectList.tsx
│   │   │   │   ├── ProjectView.tsx
│   │   │   │   ├── FeatureList.tsx
│   │   │   │   ├── FeatureView.tsx
│   │   │   │   ├── CommandButton.tsx
│   │   │   │   └── Terminal.tsx  # Optional: xterm.js
│   │   │   ├── hooks/
│   │   │   │   └── useRampAPI.ts
│   │   │   ├── types/          # TypeScript types
│   │   │   └── styles/         # CSS/styled-components
│   │   └── preload/            # Preload script
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts          # Use Vite for fast builds
│   └── electron-builder.yml    # Build configuration
│
├── shared/                     # Shared type definitions
│   └── types.ts               # Generated from Go types
│
└── README.md
```

## Technology Stack

### Backend
- **Go 1.24+** (existing version)
- **gorilla/mux** or **chi** for HTTP routing
- **gorilla/websocket** for real-time updates
- Direct imports from existing `internal/*` packages

### Frontend
- **Electron 28+** (latest stable)
- **React 18** with TypeScript
- **Vite** (fast build tool, hot module replacement)
- **TanStack Query** (React Query) for data fetching and caching
- **Zustand** for lightweight state management
- **Tailwind CSS** for styling
- **xterm.js** for embedded terminal (optional nice-to-have)
- **electron-builder** for cross-platform packaging

## API Design

### REST Endpoints

```
# Project Management
GET    /api/projects                           # List all projects in app config
POST   /api/projects                           # Add new project (select directory)
GET    /api/projects/:id                       # Get project details
DELETE /api/projects/:id                       # Remove project from app

# Feature Management
GET    /api/projects/:id/features              # List features/trees
POST   /api/projects/:id/features              # Create feature (ramp up)
DELETE /api/projects/:id/features/:name        # Delete feature (ramp down)

# Custom Commands
GET    /api/projects/:id/commands              # List custom commands from config
POST   /api/projects/:id/commands/:name/run    # Execute custom command

# Git Operations
GET    /api/projects/:id/features/:name/status # Get git status for feature

# Maintenance
POST   /api/projects/:id/refresh               # Run ramp refresh
POST   /api/projects/:id/prune                 # Run ramp prune

# Real-time Updates
WS     /ws/logs                                 # WebSocket for streaming output
```

### Example API Responses

```json
// GET /api/projects
{
  "projects": [
    {
      "id": "abc123",
      "name": "my-app",
      "path": "/Users/rob/projects/my-app",
      "repos": [...],
      "features": [...]
    }
  ]
}

// GET /api/projects/:id/features
{
  "features": [
    {
      "name": "user-auth",
      "repos": ["frontend", "backend"],
      "created": "2025-01-15T10:30:00Z",
      "hasUncommittedChanges": false
    }
  ]
}

// WebSocket message format
{
  "type": "progress",
  "operation": "up",
  "message": "Creating worktree for repo 'frontend'...",
  "percentage": 50
}
```

## User Experience Flow

### Initial Launch
1. Open app → Empty state with welcoming message
2. Large "Add Project" button prominently displayed

### Adding a Project
1. Click "Add Project" → Native directory picker dialog
2. Select directory containing `.ramp/ramp.yaml`
3. App validates and loads project configuration
4. Project appears in sidebar/list

### Managing Projects
1. Sidebar shows all added projects
2. Click project → Main view shows:
   - Project name and path
   - List of existing features/trees
   - Custom command buttons (from `commands:` in config)
   - "New Feature" button
   - Refresh/Prune buttons

### Creating Features
1. Click "New Feature" → Dialog appears
2. Enter feature name (with auto-suggested prefix from config)
3. Click "Create" → Real-time progress feedback
4. Shows spinner/progress as repos are cloned and setup scripts run
5. Success notification → Feature appears in list

### Managing Features
1. Click on feature → Expanded view shows:
   - List of repos/worktrees
   - Git status for each repo (if enabled)
   - "Open in Terminal" button (nice-to-have)
   - "Delete Feature" button
2. Click "Delete Feature" → Confirmation dialog
3. Warns if uncommitted changes detected
4. Real-time feedback during cleanup

### Running Custom Commands
1. Project view shows buttons for each custom command
2. Click command button → Execute immediately
3. Output streams in real-time (via WebSocket)
4. Success/error notification

## Code Reuse Strategy

The backend HTTP server directly imports and uses existing packages:

```go
// backend/api/features.go
package api

import (
    "encoding/json"
    "net/http"
    "ramp/internal/config"  // ← Direct import!
    "ramp/internal/git"
    "ramp/internal/ports"
    "github.com/gorilla/mux"
)

func (s *Server) handleCreateFeature(w http.ResponseWriter, r *http.Request) {
    var req CreateFeatureRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Load config using existing function
    cfg, err := config.LoadConfig(req.ProjectPath)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Use existing git operations
    for _, repo := range cfg.Repos {
        err := git.CreateWorktree(repo.Path, req.FeatureName, ...)
        // ... handle error, send progress via WebSocket
    }

    json.NewEncoder(w).Encode(FeatureResponse{Success: true})
}
```

**Benefits:**
- Zero code duplication
- Same business logic as CLI
- Bugs fixed once, both interfaces benefit
- Easy to maintain

## Electron Integration

### Backend Process Management

```typescript
// frontend/src/main/index.ts
import { spawn } from 'child_process';
import { app, BrowserWindow } from 'electron';
import path from 'path';

let backendProcess: ChildProcess | null = null;
const BACKEND_PORT = 37429;

app.whenReady().then(async () => {
  // Spawn Go backend as subprocess
  const backendPath = path.join(
    __dirname,
    '../../backend/ramp-server'
  );

  backendProcess = spawn(backendPath, [
    '--port', String(BACKEND_PORT)
  ]);

  backendProcess.stdout?.on('data', (data) => {
    console.log(`[Backend] ${data}`);
  });

  // Wait for backend to be ready
  await waitForBackend(BACKEND_PORT);

  // Create window
  createWindow();
});

app.on('quit', () => {
  backendProcess?.kill();
});
```

### Type Safety Across Stack

Generate TypeScript types from Go structs (using tools like `tygo` or manual maintenance):

```go
// backend/models/project.go
type Project struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Path     string   `json:"path"`
    Features []string `json:"features"`
}
```

```typescript
// shared/types.ts (generated)
export interface Project {
  id: string;
  name: string;
  path: string;
  features: string[];
}
```

## App Configuration Storage

Store user's project list and preferences in platform-specific locations:

- **macOS**: `~/Library/Application Support/ramp-ui/config.json`
- **Linux**: `~/.config/ramp-ui/config.json`
- **Windows**: `%APPDATA%/ramp-ui/config.json`

Example config:
```json
{
  "projects": [
    {
      "id": "abc123",
      "path": "/Users/rob/projects/my-app",
      "addedAt": "2025-01-15T10:30:00Z"
    }
  ],
  "preferences": {
    "theme": "dark",
    "showGitStatus": true
  }
}
```

## Distribution

### Homebrew Cask

```ruby
# homebrew-tap/Casks/ramp-ui.rb
cask "ramp-ui" do
  version "1.0.0"
  sha256 "..."

  url "https://github.com/robrichardson13/ramp/releases/download/v#{version}/ramp-ui-#{version}-darwin.dmg"
  name "Ramp UI"
  desc "Native desktop app for Ramp multi-repo workflow manager"
  homepage "https://github.com/robrichardson13/ramp"

  app "Ramp.app"
end
```

Install command:
```bash
brew install --cask ramp-ui
```

### GitHub Releases

Use `electron-builder` to create installers for all platforms:

```yaml
# frontend/electron-builder.yml
appId: com.ramp.ui
productName: Ramp
directories:
  buildResources: build
files:
  - '!**/.vscode/*'
  - '!src/*'
  - '!**/*.map'
mac:
  target: dmg
  category: public.app-category.developer-tools
win:
  target: nsis
linux:
  target: AppImage
  category: Development
```

Release artifacts:
- `ramp-ui-1.0.0.dmg` (macOS)
- `ramp-ui-1.0.0.exe` (Windows installer)
- `ramp-ui-1.0.0.AppImage` (Linux)

## Development Workflow

### Setup

```bash
# Install backend dependencies
cd ramp-ui/backend
go mod download

# Build backend
go build -o ramp-server .

# Install frontend dependencies
cd ../frontend
npm install

# Start development mode (hot reload)
npm run dev
```

### Development Mode

Vite provides hot module replacement for fast iteration:
- Backend runs on `http://localhost:37429`
- Frontend dev server runs on `http://localhost:5173`
- Electron loads dev server in development
- Changes to React components update instantly

### Building for Production

```bash
# Build backend binary
cd ramp-ui/backend
go build -o ../frontend/resources/ramp-server .

# Build Electron app
cd ../frontend
npm run build
npm run package  # Creates distributable
```

Add to root `Makefile`:
```makefile
build-ui:
	cd ramp-ui/backend && go build -o ../frontend/resources/ramp-server .
	cd ramp-ui/frontend && npm run build && npm run package

install-ui-deps:
	cd ramp-ui/backend && go mod download
	cd ramp-ui/frontend && npm install
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
- [ ] Create `ramp-ui/` directory structure
- [ ] Set up Go HTTP server with basic routing
- [ ] Implement project listing endpoint
- [ ] Scaffold Electron app with React + Vite
- [ ] Implement subprocess management (Electron spawns Go)
- [ ] Create basic API client hook in React
- [ ] Test end-to-end communication

**Deliverable:** App launches, backend starts, can list projects (even if empty)

### Phase 2: Project Management (Week 2)
- [ ] Build project list UI component
- [ ] Implement empty state with "Add Project" CTA
- [ ] Create directory picker dialog
- [ ] Add project validation (check for `.ramp/ramp.yaml`)
- [ ] Implement project storage in app config
- [ ] Build project detail view showing configuration
- [ ] Create feature list component for selected project

**Deliverable:** Can add projects, view project details, see existing features

### Phase 3: Feature Operations (Week 3)
- [ ] Build "New Feature" dialog with form validation
- [ ] Implement WebSocket connection for real-time updates
- [ ] Create progress UI component (spinner, status messages)
- [ ] Wire up "Create Feature" flow (`ramp up`)
- [ ] Implement feature deletion with confirmation dialog
- [ ] Add uncommitted changes warning
- [ ] Comprehensive error handling and user feedback

**Deliverable:** Full feature lifecycle (create, view, delete)

### Phase 4: Custom Commands (Week 4)
- [ ] Parse custom commands from project config
- [ ] Render command buttons dynamically
- [ ] Implement command execution endpoint
- [ ] Stream command output via WebSocket
- [ ] Build command output viewer component
- [ ] Add command history/logs

**Deliverable:** Can run custom commands and see output

### Phase 5: Nice-to-Haves (Week 5+)
- [ ] Integrate xterm.js for embedded terminal
- [ ] "Open in Terminal" button (opens native terminal at path)
- [ ] Git status visualization (uncommitted changes badge, branch info)
- [ ] Implement refresh operation UI
- [ ] Implement prune operation UI
- [ ] Settings panel (theme, update preferences)
- [ ] Dark/light theme support

**Deliverable:** Polished UX with advanced features

### Phase 6: Distribution (Week 6)
- [ ] Configure electron-builder for all platforms
- [ ] Set up code signing certificates (macOS/Windows)
- [ ] Create GitHub Actions workflow for releases
- [ ] Test builds on all platforms
- [ ] Write Homebrew cask formula
- [ ] Create installation documentation
- [ ] Update main README with UI download links

**Deliverable:** Downloadable installers, Homebrew installation

## Alternative Consideration: CLI JSON Mode

A simpler approach would be adding `--json` output flags to existing commands:

```bash
ramp status --json     # Returns structured JSON
ramp up feat-1 --json  # JSON output with progress updates
```

**Pros:**
- Simpler implementation
- No HTTP server needed
- Reuses CLI binary directly

**Cons:**
- No real-time streaming progress
- Harder to implement WebSocket-like updates
- More complex to capture and parse output
- Less clean separation of concerns

**Recommendation:** Stick with HTTP server approach for better UX and maintainability.

## Success Metrics

- **Adoption**: Track downloads and active users
- **Usability**: Users can complete core workflows without CLI knowledge
- **Performance**: Operations complete in comparable time to CLI
- **Reliability**: Same stability as CLI (shared codebase ensures this)

## Future Enhancements

- **Cloud sync**: Sync project list across devices
- **Collaboration**: Share feature environments with team
- **Notifications**: Desktop notifications for long-running operations
- **Plugins**: Extension system for custom integrations
- **Templates**: Built-in project templates for common stacks

## Questions to Resolve

1. Should the app bundle the Go backend binary, or download it separately?
   - **Recommendation**: Bundle for simplicity, ensure version compatibility
2. How to handle multiple Ramp UI instances running?
   - Use different backend ports or single-instance check
3. Auto-update strategy for the app itself?
   - Electron has built-in update mechanisms (electron-updater)

---

**Next Steps**: Create directory structure and implement Phase 1 (Core Infrastructure)
