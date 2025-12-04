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

**Note:** The actual implementation differs slightly from the original plan. The Go backend is integrated into the main module to reuse `internal/` packages directly.

```
# Go Backend (integrated into main module)
cmd/ramp-ui/
└── main.go                 # HTTP server entry point

internal/uiapi/             # API handlers and models
├── server.go               # Server setup and routing
├── projects.go             # List/add/remove projects
├── features.go             # Up/down/list features
├── websocket.go            # Real-time updates
├── appconfig.go            # App configuration storage
├── models.go               # API request/response types
├── utils.go                # Helper functions
├── appconfig_test.go       # Tests
└── projects_test.go        # Tests

# Electron Frontend
ramp-ui/frontend/
├── src/
│   ├── main/               # Electron main process
│   │   ├── index.ts        # Main entry, spawns Go backend
│   │   └── preload.ts      # Preload script for IPC
│   └── renderer/           # React app
│       ├── App.tsx
│       ├── components/
│       │   ├── ProjectList.tsx
│       │   ├── ProjectView.tsx
│       │   ├── FeatureList.tsx
│       │   ├── NewFeatureDialog.tsx
│       │   └── EmptyState.tsx
│       ├── hooks/
│       │   └── useRampAPI.ts
│       ├── types/
│       │   ├── index.ts
│       │   └── electron.d.ts
│       └── styles/
│           └── index.css   # Tailwind CSS
├── resources/              # Backend binary location
├── package.json
├── tsconfig.json
├── tsconfig.main.json
├── vite.config.ts
├── tailwind.config.js
├── postcss.config.js
└── electron-builder.yml
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

The UI app is distributed **separately from the CLI tool**. Users can have:
- Just the CLI (current users, power users)
- Just the UI (GUI-focused users)
- **Both** (recommended - they complement each other)

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

Users can update via Homebrew:
```bash
brew upgrade ramp-ui
```

### GitHub Releases (Direct Download)

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
publish:
  provider: github
  owner: robrichardson13
  repo: ramp
```

Release artifacts:
- `ramp-ui-1.0.0.dmg` (macOS)
- `ramp-ui-1.0.0.exe` (Windows installer)
- `ramp-ui-1.0.0.AppImage` (Linux)

### Installation Methods Comparison

| Method | Update Process |
|--------|----------------|
| CLI via Homebrew | Existing `internal/autoupdate` package |
| CLI manual install | Manual (no auto-update) |
| **UI via Homebrew Cask** | Homebrew OR Electron auto-updater |
| **UI via DMG/EXE** | Electron auto-updater |

## Auto-Update Strategy

The UI uses **Electron's built-in auto-updater** (`electron-updater` package), which provides seamless updates for all installation methods.

### Why Electron Auto-Updater?

- ✅ Works for **all** install methods (DMG, Homebrew Cask, Windows installer, AppImage)
- ✅ In-app notifications ("Update available - Restart to update")
- ✅ Auto-downloads and installs updates in background
- ✅ Industry standard (VS Code, Slack, Discord use this)
- ✅ Independent versioning from CLI tool
- ✅ No need to duplicate Homebrew-specific update logic

### Implementation

```typescript
// frontend/src/main/index.ts
import { autoUpdater } from 'electron-updater';

app.whenReady().then(() => {
  // Check for updates on startup
  autoUpdater.checkForUpdatesAndNotify();

  // Check for updates every 4 hours
  setInterval(() => {
    autoUpdater.checkForUpdatesAndNotify();
  }, 4 * 60 * 60 * 1000);
});

autoUpdater.on('update-available', (info) => {
  // Show notification to user
  mainWindow.webContents.send('update-available', info.version);
});

autoUpdater.on('update-downloaded', () => {
  // Prompt user to restart
  dialog.showMessageBox({
    type: 'info',
    title: 'Update Ready',
    message: 'A new version has been downloaded. Restart to apply updates.',
    buttons: ['Restart', 'Later']
  }).then((result) => {
    if (result.response === 0) {
      autoUpdater.quitAndInstall();
    }
  });
});
```

### User Settings

Users can control update behavior in the app settings:

```typescript
// Settings panel options
{
  autoUpdate: {
    enabled: true,              // Check for updates automatically
    downloadAutomatically: true, // Download in background
    channel: 'stable'           // stable | beta
  }
}
```

### Update Configuration

```yaml
# frontend/electron-builder.yml
publish:
  provider: github
  owner: robrichardson13
  repo: ramp
  releaseType: release  # or 'draft', 'prerelease'
```

This configuration allows `electron-updater` to:
1. Check GitHub Releases for new versions
2. Download the appropriate installer for user's platform
3. Verify signatures and checksums
4. Apply updates seamlessly

### Benefits

- **Separate release cycles**: UI and CLI can version independently
- **Automatic updates**: Users always have the latest features
- **Cross-platform**: Same update mechanism for macOS, Windows, Linux
- **Safe rollbacks**: Users can download previous versions from GitHub if needed

## Development Workflow

### Setup

```bash
# Build backend binary (from project root)
mkdir -p ramp-ui/frontend/resources
go build -o ramp-ui/frontend/resources/ramp-server ./cmd/ramp-ui

# Install frontend dependencies
cd ramp-ui/frontend
npm install

# Build Electron main process (including preload)
npm run build:electron

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
# From project root:

# Build backend binary
go build -o ramp-ui/frontend/resources/ramp-server ./cmd/ramp-ui

# Build Electron app
cd ramp-ui/frontend
npm run build          # Build renderer (Vite)
npm run build:electron # Build main process (TypeScript)
npm run package        # Creates distributable
```

Potential `Makefile` targets:
```makefile
build-ui:
	go build -o ramp-ui/frontend/resources/ramp-server ./cmd/ramp-ui
	cd ramp-ui/frontend && npm run build && npm run build:electron && npm run package

install-ui-deps:
	cd ramp-ui/frontend && npm install
```

## Implementation Phases

### Phase 1: Core Infrastructure ✅ COMPLETE
- [x] Create `ramp-ui/` directory structure
- [x] Set up Go HTTP server with basic routing (gorilla/mux)
- [x] Implement project listing endpoint
- [x] Scaffold Electron app with React + Vite + Tailwind CSS
- [x] Implement subprocess management (Electron spawns Go)
- [x] Create basic API client hook in React (TanStack Query)
- [x] Test end-to-end communication
- [x] Add unit tests for uiapi package (22 tests)

**Deliverable:** App launches, backend starts, can list projects (even if empty)

### Phase 2: Project Management ✅ COMPLETE
- [x] Build project list UI component
- [x] Implement empty state with "Add Project" CTA
- [x] Create directory picker dialog (native IPC)
- [x] Add project validation (check for `.ramp/ramp.yaml`)
- [x] Implement project storage in app config (platform-specific paths)
- [x] Build project detail view showing configuration (repos, branch prefix, ports, scripts)
- [x] Create feature list component for selected project (expandable with worktree details)
- [x] Add preload script for secure IPC communication

**Deliverable:** Can add projects, view project details, see existing features

### Phase 3: Feature Operations 🔄 IN PROGRESS
- [x] Build "New Feature" dialog with form validation
- [x] Implement WebSocket connection for real-time updates
- [x] Create progress UI component (spinner, status messages)
- [ ] Wire up "Create Feature" flow (`ramp up`) - backend endpoint exists, needs testing
- [x] Implement feature deletion with confirmation dialog
- [x] Add uncommitted changes warning (badge in feature list)
- [ ] Comprehensive error handling and user feedback

**Deliverable:** Full feature lifecycle (create, view, delete)

### Phase 4: Custom Commands
- [x] Parse custom commands from project config
- [x] Render command buttons dynamically
- [ ] Implement command execution endpoint
- [ ] Stream command output via WebSocket
- [ ] Build command output viewer component
- [ ] Add command history/logs

**Deliverable:** Can run custom commands and see output

### Phase 5: Nice-to-Haves
- [ ] Integrate xterm.js for embedded terminal
- [ ] "Open in Terminal" button (opens native terminal at path)
- [x] Git status visualization (uncommitted changes badge)
- [ ] Implement refresh operation UI
- [ ] Implement prune operation UI
- [ ] Settings panel (theme, update preferences)
- [x] Dark/light theme support (Tailwind dark mode classes in place)

**Deliverable:** Polished UX with advanced features

### Phase 6: Distribution
- [x] Configure electron-builder for all platforms (electron-builder.yml)
- [ ] Set up code signing certificates (macOS/Windows)
- [ ] Implement Electron auto-updater integration
- [ ] Configure update channels (stable/beta)
- [ ] Add update settings to preferences panel
- [ ] Create GitHub Actions workflow for releases
- [ ] Test builds on all platforms
- [ ] Test auto-update flow (download, install, rollback)
- [ ] Write Homebrew cask formula
- [ ] Create installation documentation
- [ ] Update main README with UI download links

**Deliverable:** Downloadable installers with auto-update support, Homebrew installation

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
3. ~~Auto-update strategy for the app itself?~~
   - ✅ **Resolved**: Use Electron's auto-updater (see Auto-Update Strategy section above)

---

## Current Status

**Phases 1 & 2 Complete.** The app can:
- Launch and start the Go backend automatically
- Add projects via native directory picker
- Validate `.ramp/ramp.yaml` exists
- Display project configuration (repos, branch prefix, base port, setup/cleanup scripts)
- Show existing features with expandable worktree details
- Remove projects from the app

**Next Steps**:
1. Test the current implementation locally (`npm run dev` in `ramp-ui/frontend`)
2. Complete Phase 3 - verify create/delete feature flows work end-to-end
3. Implement custom command execution (Phase 4)
