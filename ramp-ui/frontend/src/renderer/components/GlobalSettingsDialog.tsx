import { useState, useEffect } from 'react';
import { useAppSettings, useSaveAppSettings } from '../hooks/useRampAPI';
import { themes, getThemeById, applyTheme } from '../themes';
import type { UpdateInfo, UpdateProgress } from '../types/electron';

interface GlobalSettingsDialogProps {
  onClose: () => void;
}

const TERMINAL_OPTIONS = [
  { value: 'terminal', label: 'Terminal.app', description: 'macOS default terminal' },
  { value: 'iterm', label: 'iTerm2', description: 'Popular macOS terminal' },
  { value: 'warp', label: 'Warp', description: 'Modern terminal with AI' },
];

export default function GlobalSettingsDialog({ onClose }: GlobalSettingsDialogProps) {
  const { data: settings, isLoading } = useAppSettings();
  const saveSettings = useSaveAppSettings();
  const [terminalApp, setTerminalApp] = useState('terminal');
  const [customCommand, setCustomCommand] = useState('');
  const [isCustom, setIsCustom] = useState(false);
  const [selectedTheme, setSelectedTheme] = useState('github-dark');
  const [isSaving, setIsSaving] = useState(false);

  // Version and update state
  const [version, setVersion] = useState<string>('');
  const [updateStatus, setUpdateStatus] = useState<'idle' | 'checking' | 'available' | 'downloading' | 'ready'>('idle');
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [downloadProgress, setDownloadProgress] = useState<UpdateProgress | null>(null);

  // Revert theme and close
  const handleClose = () => {
    if (settings?.theme && settings.theme !== selectedTheme) {
      const originalTheme = getThemeById(settings.theme);
      applyTheme(originalTheme);
    }
    onClose();
  };

  // Close on Escape key (only when not saving)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isSaving) {
        handleClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose, isSaving, settings?.theme, selectedTheme]);

  // Fetch version and set up update listeners
  useEffect(() => {
    const api = window.electronAPI;
    if (!api) return;

    // Get current version
    api.getVersion().then(setVersion);

    // Set up update event listeners
    const cleanupAvailable = api.onUpdateAvailable((info) => {
      setUpdateStatus('available');
      setUpdateInfo(info);
    });

    const cleanupProgress = api.onUpdateDownloadProgress((progress) => {
      setUpdateStatus('downloading');
      setDownloadProgress(progress);
    });

    const cleanupDownloaded = api.onUpdateDownloaded((info) => {
      setUpdateStatus('ready');
      setUpdateInfo(info);
    });

    return () => {
      cleanupAvailable();
      cleanupProgress();
      cleanupDownloaded();
    };
  }, []);

  // Initialize form with current settings
  useEffect(() => {
    if (settings?.terminalApp) {
      const isBuiltIn = TERMINAL_OPTIONS.some(opt => opt.value === settings.terminalApp);
      if (isBuiltIn) {
        setTerminalApp(settings.terminalApp);
        setIsCustom(false);
      } else {
        setIsCustom(true);
        setCustomCommand(settings.terminalApp);
      }
    }
    if (settings?.theme) {
      setSelectedTheme(settings.theme);
    }
  }, [settings]);

  const handleSave = async () => {
    setIsSaving(true);
    try {
      const terminalValue = isCustom ? customCommand : terminalApp;
      await saveSettings.mutateAsync({
        terminalApp: terminalValue,
        theme: selectedTheme,
      });
      onClose();
    } catch (error) {
      console.error('Failed to save settings:', error);
      // Revert theme on error
      if (settings?.theme) {
        const originalTheme = getThemeById(settings.theme);
        applyTheme(originalTheme);
      }
    } finally {
      setIsSaving(false);
    }
  };

  // Handle theme change - apply immediately for preview
  const handleThemeChange = (themeId: string) => {
    setSelectedTheme(themeId);
    const theme = getThemeById(themeId);
    applyTheme(theme);
  };

  const handleTerminalChange = (value: string) => {
    if (value === 'custom') {
      setIsCustom(true);
    } else {
      setIsCustom(false);
      setTerminalApp(value);
    }
  };

  const handleCheckForUpdates = async () => {
    const api = window.electronAPI;
    if (!api) return;

    setUpdateStatus('checking');
    try {
      const result = await api.checkForUpdates();
      if (result.status === 'dev-mode') {
        setUpdateStatus('idle');
      } else if (result.status === 'error') {
        setUpdateStatus('idle');
      }
      // If an update is available, the event listener will handle it
      // If no update, status will remain 'checking' briefly then we reset
      setTimeout(() => {
        setUpdateStatus((current) => current === 'checking' ? 'idle' : current);
      }, 3000);
    } catch {
      setUpdateStatus('idle');
    }
  };

  const handleInstallUpdate = () => {
    window.electronAPI?.quitAndInstall();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={handleClose} />

      {/* Dialog */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
        <div className="p-6">
          {/* Header */}
          <div className="flex items-center gap-3 mb-6">
            <div className="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg">
              <svg
                className="w-5 h-5 text-gray-600 dark:text-gray-300"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                />
              </svg>
            </div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              Settings
            </h2>
          </div>

          {isLoading ? (
            <div className="animate-pulse space-y-4">
              <div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-1/3"></div>
              <div className="h-10 bg-gray-200 dark:bg-gray-600 rounded"></div>
            </div>
          ) : (
            <div className="space-y-6">
              {/* Theme Selection */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Theme
                </label>
                <select
                  value={selectedTheme}
                  onChange={(e) => handleThemeChange(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                >
                  {themes.map((theme) => (
                    <option key={theme.id} value={theme.id}>
                      {theme.name}
                    </option>
                  ))}
                </select>
              </div>

              {/* Terminal Application */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Terminal Application
                </label>
                <div className="space-y-2">
                  {TERMINAL_OPTIONS.map((option) => (
                    <label
                      key={option.value}
                      className={`flex items-center p-3 border rounded-lg cursor-pointer transition-colors ${
                        !isCustom && terminalApp === option.value
                          ? 'border-[var(--color-accent)] bg-[var(--color-accent)]/10'
                          : 'border-[var(--color-border)] hover:border-[var(--color-text-secondary)]'
                      }`}
                    >
                      <input
                        type="radio"
                        name="terminal"
                        value={option.value}
                        checked={!isCustom && terminalApp === option.value}
                        onChange={() => handleTerminalChange(option.value)}
                        className="sr-only"
                      />
                      <div className="flex-1">
                        <div className="font-medium text-gray-900 dark:text-white">
                          {option.label}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {option.description}
                        </div>
                      </div>
                      {!isCustom && terminalApp === option.value && (
                        <svg
                          className="w-5 h-5 text-primary-500"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                        >
                          <path
                            fillRule="evenodd"
                            d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                            clipRule="evenodd"
                          />
                        </svg>
                      )}
                    </label>
                  ))}

                  {/* Custom option */}
                  <label
                    className={`flex items-start p-3 border rounded-lg cursor-pointer transition-colors ${
                      isCustom
                        ? 'border-[var(--color-accent)] bg-[var(--color-accent)]/10'
                        : 'border-[var(--color-border)] hover:border-[var(--color-text-secondary)]'
                    }`}
                  >
                    <input
                      type="radio"
                      name="terminal"
                      value="custom"
                      checked={isCustom}
                      onChange={() => handleTerminalChange('custom')}
                      className="sr-only"
                    />
                    <div className="flex-1">
                      <div className="font-medium text-gray-900 dark:text-white">
                        Custom
                      </div>
                      <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                        App name or command (use $PATH for directory)
                      </div>
                      {isCustom && (
                        <input
                          type="text"
                          value={customCommand}
                          onChange={(e) => setCustomCommand(e.target.value)}
                          placeholder='e.g., "Alacritty" or "open -a MyTerm $PATH"'
                          className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                          onClick={(e) => e.stopPropagation()}
                        />
                      )}
                    </div>
                    {isCustom && (
                      <svg
                        className="w-5 h-5 text-primary-500 flex-shrink-0"
                        fill="currentColor"
                        viewBox="0 0 20 20"
                      >
                        <path
                          fillRule="evenodd"
                          d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                          clipRule="evenodd"
                        />
                      </svg>
                    )}
                  </label>
                </div>
              </div>

              {/* Version and Updates */}
              <div className="pt-4 border-t border-[var(--color-border)]">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      Version
                    </div>
                    <div className="text-sm text-gray-500 dark:text-gray-400">
                      {version || 'Loading...'}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {updateStatus === 'idle' && (
                      <button
                        type="button"
                        onClick={handleCheckForUpdates}
                        className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors"
                      >
                        Check for Updates
                      </button>
                    )}
                    {updateStatus === 'checking' && (
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        Checking...
                      </span>
                    )}
                    {updateStatus === 'available' && (
                      <span className="text-sm text-green-600 dark:text-green-400">
                        v{updateInfo?.version} downloading...
                      </span>
                    )}
                    {updateStatus === 'downloading' && downloadProgress && (
                      <span className="text-sm text-blue-600 dark:text-blue-400">
                        Downloading... {downloadProgress.percent.toFixed(0)}%
                      </span>
                    )}
                    {updateStatus === 'ready' && (
                      <button
                        type="button"
                        onClick={handleInstallUpdate}
                        className="px-3 py-1.5 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md transition-colors"
                      >
                        Install v{updateInfo?.version}
                      </button>
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="mt-6 flex justify-end gap-3">
            <button
              type="button"
              onClick={handleClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={isSaving || (isCustom && !customCommand)}
              className="px-4 py-2 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSaving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
