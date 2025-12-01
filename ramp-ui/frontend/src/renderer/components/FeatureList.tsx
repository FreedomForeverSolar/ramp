import { Feature } from '../types';
import { useDeleteFeature } from '../hooks/useRampAPI';

interface FeatureListProps {
  projectId: string;
  features: Feature[];
  isLoading: boolean;
}

export default function FeatureList({
  projectId,
  features,
  isLoading,
}: FeatureListProps) {
  const deleteFeature = useDeleteFeature(projectId);

  const handleDelete = async (featureName: string) => {
    if (confirm(`Delete feature "${featureName}"?\n\nThis will remove all worktrees for this feature.`)) {
      await deleteFeature.mutateAsync(featureName);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  if (features.length === 0) {
    return (
      <div className="text-center py-12">
        <svg
          className="mx-auto h-12 w-12 text-gray-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
          />
        </svg>
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
          No features yet
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Create a new feature to get started.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {features.map((feature) => (
        <div
          key={feature.name}
          className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
        >
          <div>
            <div className="flex items-center gap-2">
              <h3 className="font-medium text-gray-900 dark:text-white">
                {feature.name}
              </h3>
              {feature.hasUncommittedChanges && (
                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
                  Uncommitted changes
                </span>
              )}
            </div>
            <div className="mt-1 flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <span>{feature.repos.length} repos</span>
              {feature.created && (
                <>
                  <span>•</span>
                  <span>
                    Created {new Date(feature.created).toLocaleDateString()}
                  </span>
                </>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => handleDelete(feature.name)}
              disabled={deleteFeature.isPending}
              className="p-2 text-gray-500 hover:text-red-500 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-md transition-colors disabled:opacity-50"
              title="Delete feature"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
