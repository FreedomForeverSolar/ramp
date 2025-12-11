import { app, BrowserWindow, dialog, ipcMain } from 'electron';
import { spawn, ChildProcess } from 'child_process';
import { autoUpdater } from 'electron-updater';
import path from 'path';
import http from 'http';

let mainWindow: BrowserWindow | null = null;
let backendProcess: ChildProcess | null = null;

const BACKEND_PORT = 37429;
const isDev = !app.isPackaged;

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

  console.log(`Starting backend from: ${backendPath}`);

  backendProcess = spawn(backendPath, ['--port', String(BACKEND_PORT)], {
    stdio: ['ignore', 'pipe', 'pipe'],
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
    autoUpdater.checkForUpdates().catch((err) => {
      console.error('Initial update check failed:', err);
    });
  }, 3000);
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
