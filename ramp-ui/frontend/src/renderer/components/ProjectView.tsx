import { useState, useRef, useEffect } from 'react';
import { Project } from '../types';
import { useFeatures, useRemoveProject, useConfigStatus, useCommands } from '../hooks/useRampAPI';
import FeatureList from './FeatureList';
import NewFeatureDialog from './NewFeatureDialog';
import FromBranchDialog from './FromBranchDialog';
import ProjectSettings from './ProjectSettings';
import ConfigPromptsDialog from './ConfigPromptsDialog';
import SourceRepoList from './SourceRepoList';

interface ProjectViewProps {
  project: Project;
}

export default function ProjectView({ project }: ProjectViewProps) {
  const [showNewFeatureDialog, setShowNewFeatureDialog] = useState(false);
  const [showFromBranchDialog, setShowFromBranchDialog] = useState(false);
  const [showConfigDialog, setShowConfigDialog] = useState(false);
  const [pendingNewFeature, setPendingNewFeature] = useState(false);
  const [pendingFromBranch, setPendingFromBranch] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const { data: featuresData, isLoading: featuresLoading } = useFeatures(project.id);
  const { data: commandsData } = useCommands(project.id);
  const { data: configStatus } = useConfigStatus(project.id);
  const removeProject = useRemoveProject();

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowDropdown(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleRemoveProject = async () => {
    if (confirm(`Remove "${project.name}" from Ramp UI?\n\nThis will not delete any files.`)) {
      await removeProject.mutateAsync(project.id);
    }
  };

  const handleNewFeature = () => {
    // Check if config is needed before allowing feature creation
    if (configStatus?.needsConfig) {
      setPendingNewFeature(true);
      setShowConfigDialog(true);
    } else {
      setShowNewFeatureDialog(true);
    }
  };

  const handleFromBranch = () => {
    setShowDropdown(false);
    // Check if config is needed before allowing feature creation
    if (configStatus?.needsConfig) {
      setPendingFromBranch(true);
      setShowConfigDialog(true);
    } else {
      setShowFromBranchDialog(true);
    }
  };

  const handleConfigSaved = () => {
    setShowConfigDialog(false);
    // If user was trying to create a feature, show the appropriate dialog
    if (pendingNewFeature) {
      setPendingNewFeature(false);
      setShowNewFeatureDialog(true);
    } else if (pendingFromBranch) {
      setPendingFromBranch(false);
      setShowFromBranchDialog(true);
    }
  };

  const handleConfigClosed = () => {
    setShowConfigDialog(false);
    setPendingNewFeature(false);
    setPendingFromBranch(false);
  };

  const features = featuresData?.features ?? [];
  const commands = commandsData?.commands ?? [];

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
            {/* Local Preferences cog button */}
            <ProjectSettings projectId={project.id} />

            {/* Split button for New Feature */}
            <div className="relative" ref={dropdownRef}>
              <div className="inline-flex rounded-md shadow-sm">
                <button
                  onClick={handleNewFeature}
                  className="inline-flex items-center px-4 py-2 bg-primary-500 hover:bg-primary-600 text-white text-sm font-medium rounded-l-md transition-colors"
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
                  onClick={() => setShowDropdown(!showDropdown)}
                  className="inline-flex items-center px-2 py-2 bg-primary-500 hover:bg-primary-600 text-white text-sm font-medium rounded-r-md border-l border-primary-400 transition-colors"
                >
                  <svg
                    className={`w-4 h-4 transition-transform ${showDropdown ? 'rotate-180' : ''}`}
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
              </div>

              {/* Dropdown menu */}
              {showDropdown && (
                <div className="absolute right-0 mt-1 w-48 bg-white dark:bg-gray-800 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-10">
                  <div className="py-1">
                    <button
                      onClick={handleFromBranch}
                      className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                    >
                      <svg
                        className="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
                        />
                      </svg>
                      From Branch...
                    </button>
                  </div>
                </div>
              )}
            </div>
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

      </div>

      {/* Main content area */}
      <div className="flex-1 overflow-auto p-6">
        {/* Source Repositories */}
        <SourceRepoList
          projectId={project.id}
          projectPath={project.path}
          repoConfigs={project.repos}
          commands={commands}
          features={features}
        />

        {/* Features */}
        <div>
          <h3 className="text-sm font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-3">
            Features
          </h3>
          <FeatureList
            projectId={project.id}
            projectPath={project.path}
            features={features}
            commands={commands}
            isLoading={featuresLoading}
          />
        </div>
      </div>

      {/* New Feature Dialog */}
      {showNewFeatureDialog && (
        <NewFeatureDialog
          projectId={project.id}
          defaultBranchPrefix={project.defaultBranchPrefix}
          onClose={() => setShowNewFeatureDialog(false)}
        />
      )}

      {/* From Branch Dialog */}
      {showFromBranchDialog && (
        <FromBranchDialog
          projectId={project.id}
          onClose={() => setShowFromBranchDialog(false)}
        />
      )}

      {/* Config Prompts Dialog (shown when config is needed) */}
      {showConfigDialog && configStatus?.prompts && (
        <ConfigPromptsDialog
          projectId={project.id}
          prompts={configStatus.prompts}
          onClose={handleConfigClosed}
          onSaved={handleConfigSaved}
        />
      )}
    </div>
  );
}
