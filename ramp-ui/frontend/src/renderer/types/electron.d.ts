// Type definitions for Electron IPC exposed via contextBridge

export interface UpdateInfo {
  version: string;
  releaseNotes?: string;
}

export interface UpdateProgress {
  percent: number;
  bytesPerSecond: number;
  transferred: number;
  total: number;
}

export interface UpdateCheckResult {
  status: 'dev-mode' | 'checking' | 'error';
  version?: string;
  error?: string;
}

export interface ElectronAPI {
  selectDirectory: () => Promise<string | null>;
  getBackendPort: () => Promise<number>;
  platform: NodeJS.Platform;

  // Auto-updater methods
  checkForUpdates: () => Promise<UpdateCheckResult>;
  quitAndInstall: () => void;

  // Auto-updater event listeners (return cleanup function)
  onUpdateAvailable: (callback: (info: UpdateInfo) => void) => () => void;
  onUpdateDownloadProgress: (callback: (progress: UpdateProgress) => void) => () => void;
  onUpdateDownloaded: (callback: (info: UpdateInfo) => void) => () => void;
}

declare global {
  interface Window {
    electronAPI?: ElectronAPI;
  }
}

export {};
