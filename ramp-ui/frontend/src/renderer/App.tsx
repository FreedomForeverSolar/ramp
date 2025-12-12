import { useState, useEffect, useRef } from 'react';
// import { useQueryClient } from '@tanstack/react-query';
import { useProjects, useAppSettings, useSaveAppSettings } from './hooks/useRampAPI';
import ProjectList from './components/ProjectList';
import ProjectView from './components/ProjectView';
import EmptyState from './components/EmptyState';
import UpdateNotification from './components/UpdateNotification';
import { Project } from './types';

function App() {
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);
  const hasInitialized = useRef(false);
  // const queryClient = useQueryClient();
  const { data: projectsData, isLoading, error } = useProjects();
  const { data: settingsData } = useAppSettings();
  const saveSettings = useSaveAppSettings();

  const projects = projectsData?.projects ?? [];
  const selectedProject = projects.find((p: Project) => p.id === selectedProjectId);

  // Refresh features and source repo status when app comes to foreground
  // useEffect(() => {
  //   const isFeaturesQuery = (query: { queryKey: unknown }) => {
  //     const key = query.queryKey;
  //     return Array.isArray(key) && key.length >= 3 && key[0] === 'projects' && key[2] === 'features';
  //   };

  //   const isSourceReposQuery = (query: { queryKey: unknown }) => {
  //     const key = query.queryKey;
  //     return Array.isArray(key) && key.length >= 3 && key[0] === 'projects' && key[2] === 'source-repos';
  //   };

  //   const handleFocus = () => {
  //     // Skip if already fetching features
  //     const alreadyFetching = queryClient.isFetching({ predicate: isFeaturesQuery }) > 0;
  //     if (alreadyFetching) return;

  //     // Refetch all features queries (matches any project's features)
  //     queryClient.invalidateQueries({ predicate: isFeaturesQuery });
  //     // Also refresh source repo status (triggers git fetch to check behind/ahead)
  //     queryClient.invalidateQueries({ predicate: isSourceReposQuery });
  //   };

  //   const handleVisibilityChange = () => {
  //     if (document.visibilityState === 'visible') {
  //       handleFocus();
  //     }
  //   };

  //   window.addEventListener('focus', handleFocus);
  //   document.addEventListener('visibilitychange', handleVisibilityChange);

  //   return () => {
  //     window.removeEventListener('focus', handleFocus);
  //     document.removeEventListener('visibilitychange', handleVisibilityChange);
  //   };
  // }, [queryClient]);

  // Initialize selection from saved settings (runs once when both data sources are ready)
  useEffect(() => {
    if (hasInitialized.current || projects.length === 0) return;

    const lastSelectedId = settingsData?.lastSelectedProjectId;
    const projectExists = lastSelectedId && projects.some(p => p.id === lastSelectedId);

    if (projectExists) {
      setSelectedProjectId(lastSelectedId);
    } else {
      setSelectedProjectId(projects[0].id);
    }

    hasInitialized.current = true;
  }, [projects, settingsData]);

  // Handler that saves selection to settings
  const handleSelectProject = (projectId: string | null) => {
    setSelectedProjectId(projectId);
    if (projectId) {
      saveSettings.mutate({ lastSelectedProjectId: projectId });
    }
  };

  return (
    <div className="flex h-screen bg-white dark:bg-gray-900">
      {/* Auto-update notification */}
      <UpdateNotification />

      {/* Sidebar */}
      <div className="w-64 flex-shrink-0 border-r border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
        {/* Title bar drag region */}
        <div className="titlebar-drag-region h-8 border-b border-gray-200 dark:border-gray-700" />


        {/* Project list */}
        <div className="flex flex-col h-[calc(100%-2rem)]">
          <ProjectList
            projects={projects}
            selectedId={selectedProjectId}
            onSelect={handleSelectProject}
            isLoading={isLoading}
          />
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Title bar drag region */}
        <div className="titlebar-drag-region h-8 border-b border-gray-200 dark:border-gray-700 flex items-center justify-center">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {selectedProject?.name ?? 'Ramp'}
          </span>
        </div>

        {/* Content area */}
        <div className="flex-1 overflow-auto">
          {error ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center p-8">
                <div className="text-red-500 text-lg font-medium mb-2">
                  Failed to connect to backend
                </div>
                <div className="text-gray-500 text-sm">
                  {error instanceof Error ? error.message : 'Unknown error'}
                </div>
              </div>
            </div>
          ) : projects.length === 0 && !isLoading ? (
            <EmptyState />
          ) : selectedProject ? (
            <ProjectView project={selectedProject} />
          ) : (
            <div className="flex items-center justify-center h-full text-gray-500">
              Select a project to get started
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
