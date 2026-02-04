import { useState, useCallback, useRef, useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useCreateFeature, useWebSocket } from '../hooks/useRampAPI';
import { WSMessage } from '../types';
import {
  sanitizeFeatureName,
  wasInputSanitized,
  FEATURE_NAME_VALIDATION_HINT,
} from '../utils/validation';
import Convert from 'ansi-to-html';

// Create a singleton converter with options matching dark terminal background
const ansiConverter = new Convert({
  fg: '#d1d5db', // text-gray-300
  bg: '#111827', // bg-gray-900
  newline: false,
  escapeXML: true,
});

interface NewFeatureDialogProps {
  projectId: string;
  defaultBranchPrefix?: string;
  onClose: () => void;
}

export default function NewFeatureDialog({
  projectId,
  defaultBranchPrefix,
  onClose,
}: NewFeatureDialogProps) {
  const [name, setName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [prefix, setPrefix] = useState(defaultBranchPrefix || '');
  const [noPrefix, setNoPrefix] = useState(false);
  const [target, setTarget] = useState('');
  const [isCreating, setIsCreating] = useState(false);
  const [progressMessages, setProgressMessages] = useState<string[]>([]);
  const [outputLines, setOutputLines] = useState<{ text: string; isError: boolean }[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [nameValidationHint, setNameValidationHint] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const createFeature = useCreateFeature(projectId);
  const scrollRef = useRef<HTMLDivElement>(null);
  const outputRef = useRef<HTMLDivElement>(null);

  // Close on Escape key (only when not creating)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isCreating) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose, isCreating]);

  // Auto-scroll progress messages to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [progressMessages]);

  // Auto-scroll output terminal to bottom
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [outputLines]);

  // Build the full branch name preview
  const effectivePrefix = noPrefix ? '' : prefix;
  const branchPreview = effectivePrefix
    ? `${effectivePrefix}${name || '<feature-name>'}`
    : name || '<feature-name>';

  // Handle WebSocket messages for the "up" operation
  // Filter by both operation AND target (feature name) to prevent cross-contamination
  const handleWSMessage = useCallback((message: unknown) => {
    const msg = message as WSMessage;
    if (msg.operation !== 'up') return;
    // Only process messages for THIS feature to prevent race conditions
    // when multiple create operations happen in quick succession
    if (msg.target && msg.target !== name.trim()) return;

    if (msg.type === 'progress') {
      setProgressMessages(prev => [...prev, msg.message]);
    } else if (msg.type === 'output') {
      // Capture setup script output
      const isError = msg.message.startsWith('[stderr]');
      const text = isError ? msg.message.replace('[stderr] ', '') : msg.message;
      setOutputLines(prev => [...prev, { text, isError }]);
    } else if (msg.type === 'complete') {
      const featureName = name.trim();

      // Immediately add the new feature to cache (instant UI update)
      // We add minimal data - background refetch will fill in details
      queryClient.setQueryData(
        ['projects', projectId, 'features'],
        (old: { features: Array<{ name: string }> } | undefined) => {
          if (!old) return old;
          // Check if feature already exists (avoid duplicates)
          if (old.features.some(f => f.name === featureName)) return old;
          return {
            ...old,
            features: [
              ...old.features,
              {
                name: featureName,
                repos: [],
                hasUncommittedChanges: false,
                category: 'clean' as const,
              },
            ],
          };
        }
      );

      // Update the project's feature list in the sidebar (for feature count)
      queryClient.setQueryData(
        ['projects'],
        (old: { projects: Array<{ id: string; features: string[] }> } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            projects: old.projects.map(p =>
              p.id === projectId && !p.features.includes(featureName)
                ? { ...p, features: [...p.features, featureName] }
                : p
            ),
          };
        }
      );

      // Don't invalidate here - onSuccess in the mutation hook will do it
      // setQueryData above provides the immediate UI update
      onClose();
    } else if (msg.type === 'error') {
      setError(msg.message);
    }
  }, [onClose, name, queryClient, projectId]);

  // Only subscribe to WebSocket while creating
  useWebSocket(handleWSMessage, isCreating);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setIsCreating(true);
    setProgressMessages([]);
    setOutputLines([]);
    setError(null);

    try {
      await createFeature.mutateAsync({
        name: name.trim(),
        displayName: displayName.trim() || undefined,
        prefix: prefix !== defaultBranchPrefix ? prefix : undefined,
        noPrefix: noPrefix || undefined,
        target: target.trim() || undefined,
      });
      // Don't close here - wait for WebSocket 'complete' message
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      // Keep isCreating true so user stays on progress view and can see the error
      // They can retry or cancel from there
    }
  };

  const handleRetry = () => {
    setError(null);
    setProgressMessages([]);
    setOutputLines([]);
    // isCreating is already true, no need to set it again
    createFeature.mutateAsync({
      name: name.trim(),
      displayName: displayName.trim() || undefined,
      prefix: prefix !== defaultBranchPrefix ? prefix : undefined,
      noPrefix: noPrefix || undefined,
      target: target.trim() || undefined,
    }).catch(err => {
      setError(err instanceof Error ? err.message : 'Unknown error');
      // Keep isCreating true so user stays on progress view and can see the error
    });
  };

  const handleCancel = () => {
    // Allow closing even during creation (operation will continue in background)
    onClose();
  };

  // Progress view (shown during creation)
  const renderProgressView = () => (
    <div className="p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        Creating "{name}"
      </h2>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Setting up feature worktrees...
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

      {/* Setup script output terminal (shown when there's output) */}
      {outputLines.length > 0 && (
        <div className="mt-4">
          <p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Setup script output:
          </p>
          <div
            ref={outputRef}
            className="bg-gray-900 rounded-md p-3 max-h-48 overflow-y-auto font-mono text-xs"
          >
            {outputLines.map((line, i) => (
              <div
                key={i}
                className={line.isError ? 'text-red-400' : 'text-gray-300'}
                dangerouslySetInnerHTML={{ __html: ansiConverter.toHtml(line.text) }}
              />
            ))}
          </div>
        </div>
      )}

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
              onClick={handleCancel}
              className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-md transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );

  // Form view (initial state)
  const renderFormView = () => (
    <form onSubmit={handleSubmit}>
      <div className="p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
          New Feature
        </h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Create a new feature branch and worktrees across all repos.
        </p>

        <div className="mt-4">
          <label
            htmlFor="feature-name"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            Feature name
          </label>
          <input
            type="text"
            id="feature-name"
            value={name}
            onChange={(e) => {
              const raw = e.target.value;
              const sanitized = sanitizeFeatureName(raw);
              setName(sanitized);
              // Show hint if characters were removed
              setNameValidationHint(
                wasInputSanitized(raw, sanitized) ? FEATURE_NAME_VALIDATION_HINT : null
              );
            }}
            placeholder="e.g., user-authentication"
            className="mt-1 block w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            autoFocus
          />
          {nameValidationHint ? (
            <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {nameValidationHint}
            </p>
          ) : (
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Letters, numbers, hyphens, underscores, and dots only
            </p>
          )}
          <div className="mt-2 p-2 bg-gray-100 dark:bg-gray-700 rounded text-xs">
            <span className="text-gray-500 dark:text-gray-400">Branch: </span>
            <span className="font-mono text-gray-700 dark:text-gray-300">
              {branchPreview}
            </span>
          </div>
        </div>

        {/* Display Name (optional) */}
        <div className="mt-4">
          <label
            htmlFor="display-name"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            Display Name <span className="text-gray-400 font-normal">(optional)</span>
          </label>
          <input
            type="text"
            id="display-name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="e.g., User Authentication Feature"
            className="mt-1 block w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          />
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Human-readable name shown in the UI. Leave empty to use feature name.
          </p>
        </div>

        {/* Advanced Options Toggle */}
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="mt-4 flex items-center gap-1 text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300"
        >
          <svg
            className={`w-4 h-4 transition-transform ${showAdvanced ? 'rotate-90' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
          Advanced Options
        </button>

        {/* Advanced Options */}
        {showAdvanced && (
          <div className="mt-3 space-y-4 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            {/* Target Branch */}
            <div>
              <label
                htmlFor="target-branch"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                Source Branch (optional)
              </label>
              <input
                type="text"
                id="target-branch"
                value={target}
                onChange={(e) => setTarget(e.target.value)}
                placeholder="e.g., main, develop, feature/other"
                className="mt-1 block w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 text-sm"
              />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Branch to create feature from (defaults to main/master)
              </p>
            </div>

            {/* Branch Prefix */}
            <div>
              <label
                htmlFor="branch-prefix"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                Branch Prefix
              </label>
              <div className="mt-1 flex items-center gap-2">
                <input
                  type="text"
                  id="branch-prefix"
                  value={prefix}
                  onChange={(e) => setPrefix(e.target.value)}
                  disabled={noPrefix}
                  placeholder="e.g., feature/"
                  className="block flex-1 px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 text-sm disabled:opacity-50"
                />
              </div>
              <div className="mt-2 flex items-center gap-2">
                <input
                  type="checkbox"
                  id="no-prefix"
                  checked={noPrefix}
                  onChange={(e) => setNoPrefix(e.target.checked)}
                  className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
                />
                <label
                  htmlFor="no-prefix"
                  className="text-sm text-gray-600 dark:text-gray-400"
                >
                  No prefix (use feature name as branch name)
                </label>
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="px-6 py-4 bg-gray-50 dark:bg-gray-700/50 rounded-b-lg flex justify-end gap-3">
        <button
          type="button"
          onClick={onClose}
          className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-md transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={!name.trim() || createFeature.isPending}
          className="px-4 py-2 text-sm font-medium text-white bg-primary-500 hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed rounded-md transition-colors"
        >
          Create Feature
        </button>
      </div>
    </form>
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={isCreating ? undefined : onClose}
      />

      {/* Dialog */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
        {isCreating ? renderProgressView() : renderFormView()}
      </div>
    </div>
  );
}
