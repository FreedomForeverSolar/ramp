import { useState, useEffect } from 'react';
import type { UpdateInfo, UpdateProgress } from '../types/electron';

type UpdateState = 'idle' | 'available' | 'downloading' | 'ready';

export default function UpdateNotification() {
  const [state, setState] = useState<UpdateState>('idle');
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [progress, setProgress] = useState<UpdateProgress | null>(null);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    const api = window.electronAPI;
    if (!api) return;

    const cleanupAvailable = api.onUpdateAvailable((info) => {
      setUpdateInfo(info);
      setState('available');
      setDismissed(false);
    });

    const cleanupProgress = api.onUpdateDownloadProgress((prog) => {
      setProgress(prog);
      setState('downloading');
    });

    const cleanupDownloaded = api.onUpdateDownloaded((info) => {
      setUpdateInfo(info);
      setState('ready');
      setProgress(null);
    });

    return () => {
      cleanupAvailable();
      cleanupProgress();
      cleanupDownloaded();
    };
  }, []);

  const handleRestart = () => {
    window.electronAPI?.quitAndInstall();
  };

  const handleDismiss = () => {
    setDismissed(true);
  };

  if (state === 'idle' || dismissed) {
    return null;
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 max-w-sm">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 p-4">
        <div className="flex items-start gap-3">
          {/* Icon */}
          <div className="flex-shrink-0">
            {state === 'downloading' ? (
              <div className="animate-spin rounded-full h-5 w-5 border-2 border-primary-500 border-t-transparent" />
            ) : (
              <svg
                className="w-5 h-5 text-primary-500"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
            )}
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-gray-900 dark:text-white">
              {state === 'available' && 'Update Available'}
              {state === 'downloading' && 'Downloading Update'}
              {state === 'ready' && 'Update Ready'}
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
              {state === 'available' && `Version ${updateInfo?.version} is available`}
              {state === 'downloading' && progress && `${progress.percent.toFixed(0)}% complete`}
              {state === 'ready' && 'Restart to apply the update'}
            </p>

            {/* Progress bar */}
            {state === 'downloading' && progress && (
              <div className="mt-2 h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary-500 transition-all duration-300"
                  style={{ width: `${progress.percent}%` }}
                />
              </div>
            )}

            {/* Actions */}
            {state === 'ready' && (
              <div className="mt-3 flex gap-2">
                <button
                  onClick={handleRestart}
                  className="px-3 py-1.5 text-sm font-medium bg-primary-500 hover:bg-primary-600 text-white rounded transition-colors"
                >
                  Restart Now
                </button>
                <button
                  onClick={handleDismiss}
                  className="px-3 py-1.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
                >
                  Later
                </button>
              </div>
            )}
          </div>

          {/* Dismiss button (only for available state) */}
          {state === 'available' && (
            <button
              onClick={handleDismiss}
              className="flex-shrink-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
