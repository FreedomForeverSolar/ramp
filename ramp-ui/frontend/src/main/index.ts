import { app, BrowserWindow, dialog, ipcMain, Menu } from 'electron';
import { spawn, execSync, ChildProcess } from 'child_process';
import { autoUpdater } from 'electron-updater';
import path from 'path';
import http from 'http';

let mainWindow: BrowserWindow | null = null;
let backendProcess: ChildProcess | null = null;
let lastUpdateCheck = 0;

const isDev = !app.isPackaged;
const BACKEND_PORT = isDev ? 37430 : 37429;

/**
 * Get environment variables from the user's login shell.
 * When Electron is launched from Finder/Dock on macOS, it doesn't inherit
 * the user's shell environment. This function sources the user's shell
 * profile to get the full environment including PATH.
 */
function getShellEnv(): NodeJS.ProcessEnv {
  // In development (launched from terminal), we already have the full environment
  if (isDev) {
    return process.env;
  }

  const env = { ...process.env };
  const shell = process.env.SHELL || '/bin/zsh';

  try {
    // Use the user's actual shell to get their full environment
    // -l = login shell (sources profile files like .zshrc, .bash_profile)
    // -i = interactive shell (may source additional files like .zshrc)
    // -c = execute command
    // We use `env` command to print all environment variables
    const envOutput = execSync(`${shell} -l -i -c env`, {
      encoding: 'utf8',
      timeout: 10000,
      // Suppress any stderr output (shell warnings, etc.)
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    // Parse the env output and merge into our environment
    for (const line of envOutput.split('\n')) {
      const eqIndex = line.indexOf('=');
      if (eqIndex > 0) {
        const key = line.substring(0, eqIndex);
        const value = line.substring(eqIndex + 1);
        // Only override if we got a valid value
        if (key && value !== undefined) {
          env[key] = value;
        }
      }
    }

    console.log('Successfully loaded shell environment from', shell);
  } catch (err) {
    // Log the error but continue - the Go backend has its own shell env loading
    console.warn('Failed to get environment from login shell:', err);
  }

  return env;
}

function getBackendPath(): string {
  if (isDev) {
    // In development, use the backend binary in the resources folder
    // or the one built in the backend directory
    const devPath = path.join(__dirname, '../../resources/ramp-server');
    const backendDir = path.join(__dirname, '../../../backend/ramp-server');

    // Try dev path first, then backend directory
    return require('fs').existsSync(devPath) ? devPath : backendDir;
  }

  // In production, it's in the resources folder
  return path.join(process.resourcesPath, 'resources', 'ramp-server');
}

async function waitForBackend(port: number, maxAttempts = 30): Promise<boolean> {
  for (let i = 0; i < maxAttempts; i++) {
    try {
      await new Promise<void>((resolve, reject) => {
        const req = http.request(
          { host: 'localhost', port, path: '/health', method: 'GET', timeout: 1000 },
          (res) => {
            if (res.statusCode === 200) {
              resolve();
            } else {
              reject(new Error(`Unexpected status: ${res.statusCode}`));
            }
          }
        );
        req.on('error', reject);
        req.on('timeout', () => {
          req.destroy();
          reject(new Error('Timeout'));
        });
        req.end();
      });
      return true;
    } catch {
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
  }
  return false;
}

async function startBackend(): Promise<void> {
  const backendPath = getBackendPath();
  const shellEnv = getShellEnv();

  console.log(`Starting backend from: ${backendPath}`);
  console.log(`PATH: ${shellEnv.PATH}`);

  backendProcess = spawn(backendPath, ['--port', String(BACKEND_PORT)], {
    stdio: ['ignore', 'pipe', 'pipe'],
    env: shellEnv,
  });

  backendProcess.stdout?.on('data', (data: Buffer) => {
    console.log(`[Backend] ${data.toString().trim()}`);
  });

  backendProcess.stderr?.on('data', (data: Buffer) => {
    console.error(`[Backend Error] ${data.toString().trim()}`);
  });

  backendProcess.on('error', (err) => {
    console.error('Failed to start backend:', err);
    dialog.showErrorBox(
      'Backend Error',
      `Failed to start the Ramp backend server.\n\nError: ${err.message}\n\nPath: ${backendPath}`
    );
  });

  backendProcess.on('exit', (code, signal) => {
    console.log(`Backend exited with code ${code}, signal ${signal}`);
    if (code !== 0 && code !== null && mainWindow) {
      dialog.showErrorBox(
        'Backend Crashed',
        `The Ramp backend server crashed unexpectedly.\n\nExit code: ${code}`
      );
    }
  });

  // Wait for backend to be ready
  const ready = await waitForBackend(BACKEND_PORT);
  if (!ready) {
    throw new Error('Backend failed to start within timeout');
  }

  console.log('Backend is ready');
}

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 600,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
    titleBarStyle: 'hiddenInset',
    show: false,
  });

  // Load the app
  if (isDev) {
    mainWindow.loadURL('http://localhost:5173');
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, '../renderer/index.html'));
  }

  mainWindow.once('ready-to-show', () => {
    mainWindow?.show();
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Check for updates when window regains focus (debounced to once per hour)
  mainWindow.on('focus', () => {
    if (!app.isPackaged) return;

    const now = Date.now();
    const oneHour = 60 * 60 * 1000;

    if (now - lastUpdateCheck > oneHour) {
      lastUpdateCheck = now;
      autoUpdater.checkForUpdates().catch((err) => {
        console.error('Focus-triggered update check failed:', err);
      });
    }
  });

  // Application menu with keyboard shortcuts
  const menu = Menu.buildFromTemplate([
    {
      label: 'Ramp',
      submenu: [
        { role: 'about' },
        { type: 'separator' },
        { role: 'hide' },
        { role: 'hideOthers' },
        { role: 'unhide' },
        { type: 'separator' },
        { role: 'quit' }
      ]
    },
    {
      label: 'File',
      submenu: [
        {
          label: 'New Feature',
          accelerator: 'CmdOrCtrl+N',
          click: () => mainWindow?.webContents.send('menu-new-feature')
        },
        { type: 'separator' },
        {
          label: 'Refresh Repos',
          accelerator: 'CmdOrCtrl+R',
          click: () => mainWindow?.webContents.send('menu-refresh')
        },
        { type: 'separator' },
        {
          label: 'Project Settings...',
          accelerator: 'CmdOrCtrl+,',
          click: () => mainWindow?.webContents.send('menu-settings')
        }
      ]
    },
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' },
        { role: 'redo' },
        { type: 'separator' },
        { role: 'cut' },
        { role: 'copy' },
        { role: 'paste' },
        { role: 'selectAll' }
      ]
    },
    {
      label: 'Go',
      submenu: [
        { label: 'Project 1', accelerator: 'CmdOrCtrl+1', click: () => mainWindow?.webContents.send('menu-switch-project', 0) },
        { label: 'Project 2', accelerator: 'CmdOrCtrl+2', click: () => mainWindow?.webContents.send('menu-switch-project', 1) },
        { label: 'Project 3', accelerator: 'CmdOrCtrl+3', click: () => mainWindow?.webContents.send('menu-switch-project', 2) },
        { label: 'Project 4', accelerator: 'CmdOrCtrl+4', click: () => mainWindow?.webContents.send('menu-switch-project', 3) },
        { label: 'Project 5', accelerator: 'CmdOrCtrl+5', click: () => mainWindow?.webContents.send('menu-switch-project', 4) },
        { label: 'Project 6', accelerator: 'CmdOrCtrl+6', click: () => mainWindow?.webContents.send('menu-switch-project', 5) },
        { label: 'Project 7', accelerator: 'CmdOrCtrl+7', click: () => mainWindow?.webContents.send('menu-switch-project', 6) },
        { label: 'Project 8', accelerator: 'CmdOrCtrl+8', click: () => mainWindow?.webContents.send('menu-switch-project', 7) },
        { label: 'Project 9', accelerator: 'CmdOrCtrl+9', click: () => mainWindow?.webContents.send('menu-switch-project', 8) },
      ]
    },
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' },
        { role: 'zoom' },
        { role: 'close' }
      ]
    }
  ]);
  Menu.setApplicationMenu(menu);
}

// IPC handlers
ipcMain.handle('select-directory', async () => {
  if (!mainWindow) return null;

  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openDirectory'],
    title: 'Select Ramp Project Directory',
  });

  if (result.canceled || result.filePaths.length === 0) {
    return null;
  }

  return result.filePaths[0];
});

ipcMain.handle('get-backend-port', () => {
  return BACKEND_PORT;
});

ipcMain.handle('get-version', () => {
  return app.getVersion();
});

// Auto-updater IPC handlers
ipcMain.handle('check-for-updates', async () => {
  if (!app.isPackaged) {
    return { status: 'dev-mode' };
  }
  try {
    const result = await autoUpdater.checkForUpdates();
    return { status: 'checking', version: result?.updateInfo?.version };
  } catch (err) {
    console.error('Update check failed:', err);
    return { status: 'error', error: err instanceof Error ? err.message : String(err) };
  }
});

ipcMain.handle('quit-and-install', () => {
  autoUpdater.quitAndInstall();
});

// Auto-updater setup
function setupAutoUpdater(): void {
  if (!app.isPackaged) {
    console.log('Auto-updater disabled in development mode');
    return;
  }

  autoUpdater.autoDownload = true;
  autoUpdater.autoInstallOnAppQuit = true;

  autoUpdater.on('checking-for-update', () => {
    console.log('Checking for updates...');
  });

  autoUpdater.on('update-available', (info) => {
    console.log('Update available:', info.version);
    mainWindow?.webContents.send('update-available', {
      version: info.version,
      releaseNotes: info.releaseNotes,
    });
  });

  autoUpdater.on('update-not-available', () => {
    console.log('No updates available');
  });

  autoUpdater.on('download-progress', (progress) => {
    console.log(`Download progress: ${progress.percent.toFixed(1)}%`);
    mainWindow?.webContents.send('update-download-progress', {
      percent: progress.percent,
      bytesPerSecond: progress.bytesPerSecond,
      transferred: progress.transferred,
      total: progress.total,
    });
  });

  autoUpdater.on('update-downloaded', (info) => {
    console.log('Update downloaded:', info.version);
    mainWindow?.webContents.send('update-downloaded', {
      version: info.version,
      releaseNotes: info.releaseNotes,
    });
  });

  autoUpdater.on('error', (err) => {
    console.error('Auto-updater error:', err);
  });

  // Check for updates after a short delay (don't block startup)
  setTimeout(() => {
    lastUpdateCheck = Date.now();
    autoUpdater.checkForUpdates().catch((err) => {
      console.error('Initial update check failed:', err);
    });
  }, 3000);

  // Periodic check every 4 hours while running
  setInterval(() => {
    lastUpdateCheck = Date.now();
    autoUpdater.checkForUpdates().catch((err) => {
      console.error('Periodic update check failed:', err);
    });
  }, 4 * 60 * 60 * 1000);
}

// App lifecycle
app.whenReady().then(async () => {
  try {
    await startBackend();
    createWindow();
    setupAutoUpdater();
  } catch (err) {
    console.error('Failed to initialize app:', err);
    dialog.showErrorBox(
      'Initialization Error',
      `Failed to start Ramp UI.\n\nError: ${err instanceof Error ? err.message : String(err)}`
    );
    app.quit();
  }

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('quit', () => {
  if (backendProcess) {
    console.log('Stopping backend...');
    backendProcess.kill();
  }
});
