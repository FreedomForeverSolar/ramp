// Type definitions for Electron IPC exposed via contextBridge

export interface ElectronAPI {
  selectDirectory: () => Promise<string | null>;
  getBackendPort: () => Promise<number>;
  platform: NodeJS.Platform;
}

declare global {
  interface Window {
    electronAPI?: ElectronAPI;
  }
}

export {};
