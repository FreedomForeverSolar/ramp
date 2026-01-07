import { useState, useEffect } from 'react';
import { useRenameFeature } from '../hooks/useRampAPI';

interface RenameFeatureDialogProps {
  projectId: string;
  featureName: string;
  currentDisplayName: string;
  onClose: () => void;
}

export default function RenameFeatureDialog({
  projectId,
  featureName,
  currentDisplayName,
  onClose,
}: RenameFeatureDialogProps) {
  const [displayName, setDisplayName] = useState(currentDisplayName);
  const [error, setError] = useState<string | null>(null);
  const renameFeature = useRenameFeature(projectId);

  // Close on Escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !renameFeature.isPending) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose, renameFeature.isPending]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    try {
      await renameFeature.mutateAsync({
        featureName,
        displayName: displayName.trim(),
      });
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to rename feature');
    }
  };

  const hasChanged = displayName.trim() !== currentDisplayName;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={renameFeature.isPending ? undefined : onClose}
      />

      {/* Dialog */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
        <form onSubmit={handleSubmit}>
          <div className="p-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              Rename Feature
            </h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Set a human-readable display name for{' '}
              <span className="font-mono text-gray-700 dark:text-gray-300">{featureName}</span>
            </p>

            <div className="mt-4">
              <label
                htmlFor="display-name"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                Display Name
              </label>
              <input
                type="text"
                id="display-name"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="e.g., User Authentication Feature"
                className="mt-1 block w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                autoFocus
              />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Leave empty to use the feature name ({featureName})
              </p>
            </div>

            {/* Error state */}
            {error && (
              <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
              </div>
            )}
          </div>

          <div className="px-6 py-4 bg-gray-50 dark:bg-gray-700/50 rounded-b-lg flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              disabled={renameFeature.isPending}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-md transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!hasChanged || renameFeature.isPending}
              className="px-4 py-2 text-sm font-medium text-white bg-primary-500 hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed rounded-md transition-colors flex items-center gap-2"
            >
              {renameFeature.isPending && (
                <svg
                  className="w-4 h-4 animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
              )}
              {renameFeature.isPending ? 'Saving...' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
