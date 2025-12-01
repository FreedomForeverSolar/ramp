import { contextBridge, ipcRenderer } from 'electron';

// Expose protected methods that allow the renderer process to use
// ipcRenderer without exposing the entire object
contextBridge.exposeInMainWorld('electronAPI', {
  selectDirectory: () => ipcRenderer.invoke('select-directory'),
  getBackendPort: () => ipcRenderer.invoke('get-backend-port'),

  // Platform info
  platform: process.platform,
});

// Type definitions for the exposed API
declare global {
  interface Window {
    electronAPI: {
      selectDirectory: () => Promise<string | null>;
      getBackendPort: () => Promise<number>;
      platform: NodeJS.Platform;
    };
  }
}
