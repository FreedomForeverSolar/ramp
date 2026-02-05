import { useState, useCallback, useMemo, useEffect } from 'react';
import { useCreateFeature, useWebSocket } from '../hooks/useRampAPI';
import { WSMessage } from '../types';
import {
  sanitizeFeatureName,
  sanitizeBranchName,
  wasInputSanitized,
  FEATURE_NAME_VALIDATION_HINT,
  BRANCH_NAME_VALIDATION_HINT,
} from '../utils/validation';

interface FromBranchDialogProps {
  projectId: string;
  onClose: () => void;
}

export default function FromBranchDialog({
  projectId,
  onClose,
}: FromBranchDialogProps) {
  const [remoteBranch, setRemoteBranch] = useState('');
  const [featureNameOverride, setFeatureNameOverride] = useState('');
  const [featureNameValidationHint, setFeatureNameValidationHint] = useState<string | null>(null);
  const [remoteBranchValidationHint, setRemoteBranchValidationHint] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [progressMessages, setProgressMessages] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const createFeature = useCreateFeature(projectId);

  // Parse the remote branch to derive prefix and feature name
  const parsed = useMemo(() => {
    const branch = remoteBranch.trim();
    if (!branch) {
      return { prefix: '', derivedName: '', target: '' };
    }

    const lastSlash = branch.lastIndexOf('/');
    if (lastSlash === -1) {
      // No slash - entire string is feature name, no prefix
      return {
        prefix: '',
        derivedName: branch,
        target: `origin/${branch}`,
      };
    }

    // Found slash - split into prefix and feature name
    return {
      prefix: branch.substring(0, lastSlash + 1), // Include trailing slash
      derivedName: branch.substring(lastSlash + 1),
      target: `origin/${branch}`,
    };
  }, [remoteBranch]);

  const effectiveFeatureName = featureNameOverride.trim() || parsed.derivedName;

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

  // Handle WebSocket messages for the "up" operation
  // Filter by both operation AND target (feature name) to prevent cross-contamination
  const handleWSMessage = useCallback((message: unknown) => {
    const msg = message as WSMessage;
    if (msg.operation !== 'up') return;
    // Only process messages for THIS feature to prevent race conditions
    if (msg.target && msg.target !== effectiveFeatureName) return;

    if (msg.type === 'progress') {
      setProgressMessages(prev => [...prev, msg.message]);
    } else if (msg.type === 'complete') {
      onClose();
    } else if (msg.type === 'error') {
      setError(msg.message);
    }
  }, [onClose, effectiveFeatureName]);

  useWebSocket(handleWSMessage, isCreating);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!remoteBranch.trim()) return;

    setIsCreating(true);
    setProgressMessages([]);
    setError(null);

    try {
      await createFeature.mutateAsync({
        name: featureNameOverride.trim() || undefined,
        fromBranch: remoteBranch.trim(),
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    }
  };

  const handleRetry = () => {
    setError(null);
    setProgressMessages([]);
    setIsCreating(true);
    createFeature.mutateAsync({
      name: featureNameOverride.trim() || undefined,
      fromBranch: remoteBranch.trim(),
    }).catch(err => {
      setError(err instanceof Error ? err.message : 'Unknown error');
    });
  };

  const handleCancel = () => {
    onClose();
  };

  // Progress view (shown during creation)
  const renderProgressView = () => (
    <div className="p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        Creating from "{remoteBranch}"
      </h2>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Setting up feature worktrees from remote branch...
      </p>

      <div className="mt-4 space-y-2 min-h-24 max-h-64 overflow-y-auto scrollbar-hide">
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

        {!error && (
          <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            <span>Working...</span>
          </div>
        )}
      </div>

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

  // Form view
  const renderFormView = () => (
    <form onSubmit={handleSubmit}>
      <div className="p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
          Create from Remote Branch
        </h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Create a feature from an existing remote branch.
        </p>

        <div className="mt-4">
          <label
            htmlFor="remote-branch"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            Remote Branch
          </label>
          <input
            type="text"
            id="remote-branch"
            value={remoteBranch}
            onChange={(e) => {
              const raw = e.target.value;
              const sanitized = sanitizeBranchName(raw);
              setRemoteBranch(sanitized);
              setRemoteBranchValidationHint(
                wasInputSanitized(raw, sanitized) ? BRANCH_NAME_VALIDATION_HINT : null
              );
            }}
            placeholder="e.g., claude/feature-123, feature/my-branch"
            className="mt-1 block w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            autoFocus
          />
          {remoteBranchValidationHint ? (
            <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {remoteBranchValidationHint}
            </p>
          ) : (
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Enter the branch name without &quot;origin/&quot; prefix
            </p>
          )}
        </div>

        {/* Feature Name Override */}
        <div className="mt-4">
          <label
            htmlFor="feature-name-override"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            Feature Name (optional)
          </label>
          <input
            type="text"
            id="feature-name-override"
            value={featureNameOverride}
            onChange={(e) => {
              const raw = e.target.value;
              const sanitized = sanitizeFeatureName(raw);
              setFeatureNameOverride(sanitized);
              // Show hint if characters were removed
              setFeatureNameValidationHint(
                wasInputSanitized(raw, sanitized) ? FEATURE_NAME_VALIDATION_HINT : null
              );
            }}
            placeholder={parsed.derivedName || 'Auto-derived from branch'}
            className="mt-1 block w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          />
          {featureNameValidationHint ? (
            <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {featureNameValidationHint}
            </p>
          ) : (
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Override the feature name (defaults to last part of branch). Letters, numbers, hyphens, underscores, and dots only.
            </p>
          )}
        </div>

        {/* Preview */}
        {remoteBranch.trim() && (
          <div className="mt-4 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-2">
            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Preview</h3>
            <div className="text-sm grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 items-baseline">
              <span className="text-gray-500 dark:text-gray-400">Feature name:</span>
              <span className="font-mono text-gray-900 dark:text-white truncate" title={effectiveFeatureName || '(invalid)'}>
                {effectiveFeatureName || '(invalid)'}
              </span>
              <span className="text-gray-500 dark:text-gray-400">Worktree path:</span>
              <span className="font-mono text-gray-900 dark:text-white truncate" title={`trees/${effectiveFeatureName || '...'}/`}>
                trees/{effectiveFeatureName || '...'}/
              </span>
              <span className="text-gray-500 dark:text-gray-400">From remote:</span>
              <span className="font-mono text-gray-900 dark:text-white truncate" title={parsed.target}>
                {parsed.target}
              </span>
              {parsed.prefix && (
                <>
                  <span className="text-gray-500 dark:text-gray-400">Branch prefix:</span>
                  <span className="font-mono text-gray-900 dark:text-white truncate" title={parsed.prefix}>
                    {parsed.prefix}
                  </span>
                </>
              )}
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
          disabled={!remoteBranch.trim() || !effectiveFeatureName || createFeature.isPending}
          className="px-4 py-2 text-sm font-medium text-white bg-primary-500 hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed rounded-md transition-colors"
        >
          Create Feature
        </button>
      </div>
    </form>
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="absolute inset-0 bg-black/50"
        onClick={isCreating ? undefined : onClose}
      />
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-xl mx-4">
        {isCreating ? renderProgressView() : renderFormView()}
      </div>
    </div>
  );
}
