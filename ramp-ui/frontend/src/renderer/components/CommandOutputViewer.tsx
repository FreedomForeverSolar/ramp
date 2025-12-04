import { useEffect, useRef } from 'react';

interface CommandOutputViewerProps {
  output: string[];
  isRunning: boolean;
  onClose: () => void;
  commandName: string;
}

export default function CommandOutputViewer({
  output,
  isRunning,
  onClose,
  commandName,
}: CommandOutputViewerProps) {
  const outputRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new output arrives
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [output]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={isRunning ? undefined : onClose}
      />

      {/* Dialog */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-2xl mx-4 max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              {commandName}
            </h2>
            {isRunning && (
              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                <svg className="w-3 h-3 mr-1 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Running
              </span>
            )}
          </div>
          <button
            onClick={onClose}
            disabled={isRunning}
            className="p-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 disabled:opacity-50"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Output */}
        <div
          ref={outputRef}
          className="flex-1 overflow-auto p-4 bg-gray-900 font-mono text-sm"
        >
          {output.length === 0 ? (
            <div className="text-gray-500">Waiting for output...</div>
          ) : (
            output.map((line, index) => (
              <div
                key={index}
                className={`whitespace-pre-wrap ${
                  line.startsWith('[stderr]')
                    ? 'text-red-400'
                    : 'text-green-400'
                }`}
              >
                {line}
              </div>
            ))
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end p-4 border-t border-gray-200 dark:border-gray-700">
          <button
            onClick={onClose}
            disabled={isRunning}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-md transition-colors disabled:opacity-50"
          >
            {isRunning ? 'Running...' : 'Close'}
          </button>
        </div>
      </div>
    </div>
  );
}
