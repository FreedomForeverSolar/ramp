import { useState, useEffect } from 'react';
import { Project, WSMessage } from '../types';
import { useFeatures, useRemoveProject, useWebSocket } from '../hooks/useRampAPI';
import FeatureList from './FeatureList';
import NewFeatureDialog from './NewFeatureDialog';

interface ProjectViewProps {
  project: Project;
}

export default function ProjectView({ project }: ProjectViewProps) {
  const [showNewFeatureDialog, setShowNewFeatureDialog] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const { data: featuresData, isLoading: featuresLoading } = useFeatures(project.id);
  const removeProject = useRemoveProject();

  // WebSocket for real-time updates
  const { connect, disconnect } = useWebSocket((message) => {
    const msg = message as WSMessage;
    if (msg.type === 'progress' || msg.type === 'error') {
      setStatusMessage(msg.message);
    } else if (msg.type === 'complete') {
      setStatusMessage(msg.message);
      // Clear message after 3 seconds
      setTimeout(() => setStatusMessage(null), 3000);
    }
  });

  useEffect(() => {
    connect();
    return () => disconnect();
  }, []);

  const handleRemoveProject = async () => {
    if (confirm(`Remove "${project.name}" from Ramp UI?\n\nThis will not delete any files.`)) {
      await removeProject.mutateAsync(project.id);
    }
  };

  const features = featuresData?.features ?? [];

  return (
    <div className="h-full flex flex-col">
      {/* Project header */}
      <div className="p-6 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {project.name}
            </h1>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400 font-mono">
              {project.path}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowNewFeatureDialog(true)}
              className="inline-flex items-center px-4 py-2 bg-primary-500 hover:bg-primary-600 text-white text-sm font-medium rounded-md transition-colors"
            >
              <svg
                className="w-4 h-4 mr-2"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 4v16m8-8H4"
                />
              </svg>
              New Feature
            </button>
            <button
              onClick={handleRemoveProject}
              className="p-2 text-gray-500 hover:text-red-500 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-md transition-colors"
              title="Remove project"
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

        {/* Repos summary */}
        <div className="mt-4 flex flex-wrap gap-2">
          {project.repos.map((repo) => (
            <span
              key={repo.name}
              className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200"
            >
              {repo.name}
            </span>
          ))}
        </div>

        {/* Status message */}
        {statusMessage && (
          <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md">
            <p className="text-sm text-blue-700 dark:text-blue-300">
              {statusMessage}
            </p>
          </div>
        )}
      </div>

      {/* Features */}
      <div className="flex-1 overflow-auto p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Features
        </h2>
        <FeatureList
          projectId={project.id}
          features={features}
          isLoading={featuresLoading}
        />
      </div>

      {/* Custom commands */}
      {project.commands.length > 0 && (
        <div className="border-t border-gray-200 dark:border-gray-700 p-4">
          <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
            Quick Actions
          </h3>
          <div className="flex flex-wrap gap-2">
            {project.commands.map((cmd) => (
              <button
                key={cmd.name}
                className="px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-md transition-colors"
              >
                {cmd.name}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* New Feature Dialog */}
      {showNewFeatureDialog && (
        <NewFeatureDialog
          projectId={project.id}
          onClose={() => setShowNewFeatureDialog(false)}
        />
      )}
    </div>
  );
}
