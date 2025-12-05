import { useState } from 'react';
import { Feature } from '../types';
import DeleteFeatureDialog from './DeleteFeatureDialog';

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
  const [expandedFeature, setExpandedFeature] = useState<string | null>(null);
  const [deletingFeature, setDeletingFeature] = useState<Feature | null>(null);

  const handleDelete = (feature: Feature) => {
    setDeletingFeature(feature);
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

  const toggleExpanded = (featureName: string) => {
    setExpandedFeature(expandedFeature === featureName ? null : featureName);
  };

  return (
    <div className="space-y-3">
      {features.map((feature) => {
        const isExpanded = expandedFeature === feature.name;
        return (
          <div
            key={feature.name}
            className="bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden"
          >
            {/* Feature header */}
            <div className="flex items-center justify-between p-4">
              <button
                onClick={() => toggleExpanded(feature.name)}
                className="flex-1 flex items-start gap-3 text-left"
              >
                <svg
                  className={`w-5 h-5 mt-0.5 text-gray-400 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
                <div className="flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <h3 className="font-medium text-gray-900 dark:text-white">
                      {feature.name}
                    </h3>
                    {feature.hasUncommittedChanges && (
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
                        <svg className="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        </svg>
                        Uncommitted changes
                      </span>
                    )}
                  </div>
                  <div className="mt-1 flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                    <span>{feature.repos.length} repo{feature.repos.length !== 1 ? 's' : ''}</span>
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
              </button>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => handleDelete(feature)}
                  className="p-2 text-gray-500 hover:text-red-500 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-md transition-colors"
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

            {/* Expanded details */}
            {isExpanded && (
              <div className="px-4 pb-4 pt-0 border-t border-gray-200 dark:border-gray-700">
                <div className="pt-3">
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-2">
                    Worktrees
                  </h4>
                  <div className="space-y-2">
                    {feature.repos.map((repoName) => (
                      <div
                        key={repoName}
                        className="flex items-center justify-between py-2 px-3 bg-white dark:bg-gray-700/50 rounded-md"
                      >
                        <span className="text-sm font-mono text-gray-700 dark:text-gray-300">
                          {repoName}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        );
      })}

      {/* Delete Feature Dialog */}
      {deletingFeature && (
        <DeleteFeatureDialog
          projectId={projectId}
          featureName={deletingFeature.name}
          hasUncommittedChanges={deletingFeature.hasUncommittedChanges}
          onClose={() => setDeletingFeature(null)}
        />
      )}
    </div>
  );
}
