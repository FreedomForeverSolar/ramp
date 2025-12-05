import { useState } from 'react';
import { Project } from '../types';
import { useFeatures, useRemoveProject } from '../hooks/useRampAPI';
import FeatureList from './FeatureList';
import NewFeatureDialog from './NewFeatureDialog';

interface ProjectViewProps {
  project: Project;
}

export default function ProjectView({ project }: ProjectViewProps) {
  const [showNewFeatureDialog, setShowNewFeatureDialog] = useState(false);
  const { data: featuresData, isLoading: featuresLoading } = useFeatures(project.id);
  const removeProject = useRemoveProject();

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

        {/* Project config info */}
        <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-sm">
            <span className="text-gray-500 dark:text-gray-400">Repos:</span>
            <span className="ml-2 font-medium text-gray-900 dark:text-white">
              {project.repos.length}
            </span>
          </div>
          {project.defaultBranchPrefix && (
            <div className="text-sm">
              <span className="text-gray-500 dark:text-gray-400">Branch prefix:</span>
              <span className="ml-2 font-mono text-gray-900 dark:text-white">
                {project.defaultBranchPrefix}
              </span>
            </div>
          )}
          {project.basePort && project.basePort > 0 && (
            <div className="text-sm">
              <span className="text-gray-500 dark:text-gray-400">Base port:</span>
              <span className="ml-2 font-mono text-gray-900 dark:text-white">
                {project.basePort}
              </span>
            </div>
          )}
        </div>

        {/* Repos summary */}
        <div className="mt-4 flex flex-wrap gap-2">
          {project.repos.map((repo) => (
            <span
              key={repo.name}
              className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200"
              title={`${repo.git}${repo.autoRefresh ? '' : ' (auto-refresh disabled)'}`}
            >
              {repo.name}
              {!repo.autoRefresh && (
                <svg className="w-3 h-3 ml-1 text-yellow-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              )}
            </span>
          ))}
        </div>
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

      {/* New Feature Dialog */}
      {showNewFeatureDialog && (
        <NewFeatureDialog
          projectId={project.id}
          defaultBranchPrefix={project.defaultBranchPrefix}
          onClose={() => setShowNewFeatureDialog(false)}
        />
      )}
    </div>
  );
}
