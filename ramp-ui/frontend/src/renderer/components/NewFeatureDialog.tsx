import { useState } from 'react';
import { useCreateFeature } from '../hooks/useRampAPI';

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
  const createFeature = useCreateFeature(projectId);

  // Build the full branch name preview
  const branchPreview = defaultBranchPrefix
    ? `${defaultBranchPrefix}${name || '<feature-name>'}`
    : name || '<feature-name>';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    try {
      await createFeature.mutateAsync({ name: name.trim() });
      onClose();
    } catch (error) {
      alert(`Failed to create feature: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
      />

      {/* Dialog */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
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
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., user-authentication"
                className="mt-1 block w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                autoFocus
              />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Use lowercase with hyphens (e.g., my-feature)
              </p>
              {defaultBranchPrefix && (
                <div className="mt-2 p-2 bg-gray-100 dark:bg-gray-700 rounded text-xs">
                  <span className="text-gray-500 dark:text-gray-400">Branch: </span>
                  <span className="font-mono text-gray-700 dark:text-gray-300">
                    {branchPreview}
                  </span>
                </div>
              )}
            </div>
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
              {createFeature.isPending ? 'Creating...' : 'Create Feature'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
