import { useState, useCallback, useRef, useEffect } from 'react';
import { useRunCommand, useWebSocket } from '../hooks/useRampAPI';
import { WSMessage, Feature } from '../types';
import Convert from 'ansi-to-html';

// Create a singleton converter with options matching dark terminal background
const ansiConverter = new Convert({
  fg: '#d1d5db', // text-gray-300
  bg: '#111827', // bg-gray-900
  newline: false,
  escapeXML: true,
});

interface RunCommandDialogProps {
  projectId: string;
  commandName: string;
  featureName?: string; // Pre-selected feature (from card menu) or undefined
  features: Feature[];  // All features for selector
  runImmediately?: boolean; // Skip selection and run immediately (for source mode)
  onClose: () => void;
}

export default function RunCommandDialog({
  projectId,
  commandName,
  featureName: initialFeatureName,
  features,
  runImmediately = false,
  onClose,
}: RunCommandDialogProps) {
  // If featureName is provided or runImmediately is true, skip selection and go straight to execution
  const shouldAutoRun = !!initialFeatureName || runImmediately;
  const [selectedTarget, setSelectedTarget] = useState<string>(initialFeatureName || '');
  const [isRunning, setIsRunning] = useState(shouldAutoRun);
  const [outputLines, setOutputLines] = useState<{ text: string; isError: boolean }[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const runCommand = useRunCommand(projectId);
  const outputRef = useRef<HTMLDivElement>(null);
  const hasStartedRef = useRef(false);

  // Close on Escape key (only when not running)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isRunning) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose, isRunning]);

  // Auto-scroll output
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [outputLines]);

  // Determine target for message filtering
  const targetForFiltering = selectedTarget || 'source';

  // Handle WebSocket messages for the "run" operation
  const handleWSMessage = useCallback((message: unknown) => {
    const msg = message as WSMessage;
    if (msg.operation !== 'run') return;
    // Filter by command name AND target
    if (msg.command !== commandName) return;
    if (msg.target && msg.target !== targetForFiltering) return;

    if (msg.type === 'output') {
      const isError = msg.message.startsWith('[stderr]');
      const text = isError ? msg.message.replace('[stderr] ', '') : msg.message;
      setOutputLines(prev => [...prev, { text, isError }]);
    } else if (msg.type === 'progress') {
      // Don't add progress messages to outputLines - they're status updates,
      // not actual command output. This ensures auto-close works correctly.
    } else if (msg.type === 'complete') {
      setSuccess(true);
      setIsRunning(false);
    } else if (msg.type === 'error') {
      setError(msg.message);
      setIsRunning(false);
    }
  }, [commandName, targetForFiltering]);

  // Only subscribe to WebSocket while running
  useWebSocket(handleWSMessage, isRunning);

  // Auto-start if featureName was provided or runImmediately is true
  // Use ref to prevent double-execution from React 18 Strict Mode
  useEffect(() => {
    if (shouldAutoRun && isRunning && outputLines.length === 0 && !hasStartedRef.current) {
      hasStartedRef.current = true;
      executeCommand(selectedTarget);
    }
  }, []); // Run once on mount

  // Auto-close on success with no output
  useEffect(() => {
    if (success && outputLines.length === 0) {
      onClose();
    }
  }, [success, outputLines.length, onClose]);

  const executeCommand = async (target: string) => {
    setIsRunning(true);
    setOutputLines([]);
    setError(null);
    setSuccess(false);

    try {
      const featureToRun = target || undefined; // Empty string = source mode
      await runCommand.mutateAsync({ commandName, featureName: featureToRun });
      // Don't close here - wait for WebSocket 'complete' message
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setIsRunning(false);
    }
  };

  const handleRun = () => {
    executeCommand(selectedTarget);
  };

  const handleRetry = () => {
    executeCommand(selectedTarget);
  };

  // Selection view (choose target)
  const renderSelectionView = () => (
    <div className="p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        Run "{commandName}"
      </h2>
      <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
        Select where to run this command:
      </p>

      <div className="mt-4">
        <select
          value={selectedTarget}
          onChange={(e) => setSelectedTarget(e.target.value)}
          className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        >
          <option value="">Source (project root)</option>
          {features.map((feature) => (
            <option key={feature.name} value={feature.name}>
              {feature.name}
            </option>
          ))}
        </select>
      </div>

      <div className="mt-6 flex justify-end gap-3">
        <button
          type="button"
          onClick={onClose}
          className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-md transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={handleRun}
          className="px-4 py-2 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 rounded-md transition-colors"
        >
          Run
        </button>
      </div>
    </div>
  );

  // Output view (shown during/after execution)
  const renderOutputView = () => (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
          {commandName}
        </h2>
        <span className="text-sm text-gray-500 dark:text-gray-400">
          {selectedTarget || 'source'}
        </span>
      </div>

      {/* Output terminal */}
      <div
        ref={outputRef}
        className="mt-4 bg-gray-900 rounded-md p-4 min-h-48 max-h-80 overflow-y-auto font-mono text-sm"
      >
        {outputLines.map((line, i) => (
          <div
            key={i}
            className={line.isError ? 'text-red-400' : 'text-gray-300'}
            dangerouslySetInnerHTML={{ __html: ansiConverter.toHtml(line.text) }}
          />
        ))}

        {/* Show cursor while running */}
        {isRunning && (
          <div className="flex items-center gap-1 text-gray-400">
            <span className="animate-pulse">_</span>
          </div>
        )}
      </div>

      {/* Status bar */}
      <div className="mt-4 flex items-center justify-between">
        {isRunning ? (
          <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            <span>Running...</span>
          </div>
        ) : success ? (
          <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
            </svg>
            <span>Completed successfully</span>
          </div>
        ) : error ? (
          <div className="flex items-center gap-2 text-sm text-red-600 dark:text-red-400">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            <span>Failed</span>
          </div>
        ) : null}

        <div className="flex gap-2">
          {error && !isRunning && (
            <button
              onClick={handleRetry}
              className="px-3 py-1.5 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 rounded-md transition-colors"
            >
              Retry
            </button>
          )}
          <button
            onClick={onClose}
            disabled={isRunning}
            className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
              isRunning
                ? 'text-gray-400 cursor-not-allowed'
                : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600'
            }`}
          >
            {success || error ? 'Close' : 'Cancel'}
          </button>
        </div>
      </div>
    </div>
  );

  // Determine which view to show
  const showOutputView = isRunning || success || error || shouldAutoRun;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={isRunning ? undefined : onClose}
      />

      {/* Dialog */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-lg mx-4">
        {showOutputView ? renderOutputView() : renderSelectionView()}
      </div>
    </div>
  );
}
