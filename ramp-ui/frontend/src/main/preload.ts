import { contextBridge, ipcRenderer, IpcRendererEvent } from 'electron';

// Expose protected methods that allow the renderer process to use
// ipcRenderer without exposing the entire object
contextBridge.exposeInMainWorld('electronAPI', {
  selectDirectory: () => ipcRenderer.invoke('select-directory'),
  getBackendPort: () => ipcRenderer.invoke('get-backend-port'),
  platform: process.platform,

  // Auto-updater methods
  checkForUpdates: () => ipcRenderer.invoke('check-for-updates'),
  quitAndInstall: () => ipcRenderer.invoke('quit-and-install'),

  // Auto-updater event listeners
  onUpdateAvailable: (callback: (info: { version: string; releaseNotes?: string }) => void) => {
    const listener = (_event: IpcRendererEvent, info: { version: string; releaseNotes?: string }) => callback(info);
    ipcRenderer.on('update-available', listener);
    return () => ipcRenderer.removeListener('update-available', listener);
  },
  onUpdateDownloadProgress: (callback: (progress: { percent: number; bytesPerSecond: number; transferred: number; total: number }) => void) => {
    const listener = (_event: IpcRendererEvent, progress: { percent: number; bytesPerSecond: number; transferred: number; total: number }) => callback(progress);
    ipcRenderer.on('update-download-progress', listener);
    return () => ipcRenderer.removeListener('update-download-progress', listener);
  },
  onUpdateDownloaded: (callback: (info: { version: string; releaseNotes?: string }) => void) => {
    const listener = (_event: IpcRendererEvent, info: { version: string; releaseNotes?: string }) => callback(info);
    ipcRenderer.on('update-downloaded', listener);
    return () => ipcRenderer.removeListener('update-downloaded', listener);
  },
});
