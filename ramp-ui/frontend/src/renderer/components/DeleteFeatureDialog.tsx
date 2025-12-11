import { useState, useCallback, useRef, useEffect } from 'react';
import { useDeleteFeature, useWebSocket } from '../hooks/useRampAPI';
import { WSMessage } from '../types';

interface DeleteFeatureDialogProps {
  projectId: string;
  featureName: string;
  hasUncommittedChanges: boolean;
  onClose: () => void;
}

export default function DeleteFeatureDialog({
  projectId,
  featureName,
  hasUncommittedChanges,
  onClose,
}: DeleteFeatureDialogProps) {
  const [isDeleting, setIsDeleting] = useState(false);
  const [progressMessages, setProgressMessages] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [acknowledged, setAcknowledged] = useState(false);
  const deleteFeature = useDeleteFeature(projectId);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll progress messages to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [progressMessages]);

  // Delete button is disabled if there are uncommitted changes and user hasn't acknowledged
  const canDelete = !hasUncommittedChanges || acknowledged;

  // Handle WebSocket messages for the "down" operation
  // Filter by both operation AND target (feature name) to prevent cross-contamination
  const handleWSMessage = useCallback((message: unknown) => {
    const msg = message as WSMessage;
    if (msg.operation !== 'down') return;
    // Only process messages for THIS feature to prevent race conditions
    // when multiple delete operations happen in quick succession
    if (msg.target && msg.target !== featureName) return;

    if (msg.type === 'progress') {
      setProgressMessages(prev => [...prev, msg.message]);
    } else if (msg.type === 'complete') {
      // Success - close the modal
      onClose();
    } else if (msg.type === 'error') {
      setError(msg.message);
    }
  }, [onClose, featureName]);

  // Only subscribe to WebSocket while deleting
  useWebSocket(handleWSMessage, isDeleting);

  const handleDelete = async () => {
    setIsDeleting(true);
    setProgressMessages([]);
    setError(null);

    try {
      await deleteFeature.mutateAsync(featureName);
      // Don't close here - wait for WebSocket 'complete' message
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setIsDeleting(false); // Reset to allow retry
    }
  };

  const handleRetry = () => {
    setError(null);
    setProgressMessages([]);
    setIsDeleting(true);
    deleteFeature.mutateAsync(featureName).catch(err => {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setIsDeleting(false); // Reset to allow retry
    });
  };

  // Progress view (shown during deletion)
  const renderProgressView = () => (
    <div className="p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        Deleting "{featureName}"
      </h2>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Removing worktrees and cleaning up...
      </p>

      <div ref={scrollRef} className="mt-4 space-y-2 min-h-24 max-h-64 overflow-y-auto scrollbar-hide">
        {progressMessages.map((msg, i) => (
          <div
            key={i}
            className="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-400"
          >
            <span className="text-green-500 mt-0.5">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
              </svg>
            </span>
            <span>{msg}</span>
          </div>
        ))}

        {/* Show spinner while still working (no error) */}
        {!error && (
          <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
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
            <span>Working...</span>
          </div>
        )}
      </div>

      {/* Error state */}
      {error && (
        <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
          <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
          <div className="mt-3 flex gap-2">
            <button
              onClick={handleRetry}
              className="px-3 py-1.5 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors"
            >
              Try Again
            </button>
            <button
              onClick={onClose}
              className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-md transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );

  // Confirm view (initial state)
  const renderConfirmView = () => (
    <div className="p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        Delete Feature
      </h2>
      <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
        Delete "<span className="font-medium text-gray-900 dark:text-white">{featureName}</span>"?
      </p>
      <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
        This will remove all worktrees for this feature.
      </p>

      {hasUncommittedChanges && (
        <div className="mt-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-md">
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5 text-yellow-600 dark:text-yellow-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
              This feature has uncommitted changes
            </p>
          </div>
          <label className="mt-3 flex items-start gap-2 cursor-pointer ml-7">
            <input
              type="checkbox"
              checked={acknowledged}
              onChange={(e) => setAcknowledged(e.target.checked)}
              className="mt-0.5 h-4 w-4 rounded border-yellow-400 text-yellow-600 focus:ring-yellow-500"
            />
            <span className="text-sm text-yellow-700 dark:text-yellow-300">
              I understand that uncommitted changes will be lost
            </span>
          </label>
        </div>
      )}

      <div className="mt-6 flex justify-end gap-3">
        <button
          type="button"
          onClick={onClose}
          className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-md transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={handleDelete}
          disabled={!canDelete}
          className={`px-4 py-2 text-sm font-medium text-white rounded-md transition-colors ${
            canDelete
              ? 'bg-red-600 hover:bg-red-700'
              : 'bg-red-300 dark:bg-red-800 cursor-not-allowed'
          }`}
        >
          Delete
        </button>
      </div>
    </div>
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={isDeleting ? undefined : onClose}
      />

      {/* Dialog */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
        {isDeleting ? renderProgressView() : renderConfirmView()}
      </div>
    </div>
  );
}
