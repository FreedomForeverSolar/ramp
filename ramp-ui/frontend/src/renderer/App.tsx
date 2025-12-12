import { useState, useEffect, useRef, useMemo } from 'react';
// import { useQueryClient } from '@tanstack/react-query';
import { useProjects, useAppSettings, useSaveAppSettings } from './hooks/useRampAPI';
import ProjectList from './components/ProjectList';
import ProjectView from './components/ProjectView';
import EmptyState from './components/EmptyState';
import UpdateNotification from './components/UpdateNotification';
import { Project } from './types';
import { getThemeById, applyTheme } from './themes';

function App() {
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);
  const hasInitialized = useRef(false);
  // const queryClient = useQueryClient();
  const { data: projectsData, isLoading, error } = useProjects();
  const { data: settingsData } = useAppSettings();
  const saveSettings = useSaveAppSettings();

  const projects = projectsData?.projects ?? [];
  const selectedProject = projects.find((p: Project) => p.id === selectedProjectId);

  // Sort projects same as ProjectList: favorites first, then by order
  const sortedProjects = useMemo(() => {
    return [...projects].sort((a, b) => {
      if (a.isFavorite !== b.isFavorite) {
        return a.isFavorite ? -1 : 1;
      }
      return a.order - b.order;
    });
  }, [projects]);

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

  // Apply theme when settings load or change
  useEffect(() => {
    if (settingsData?.theme) {
      const theme = getThemeById(settingsData.theme);
      applyTheme(theme);
    }
  }, [settingsData?.theme]);

  // Listen for menu keyboard shortcuts
  useEffect(() => {
    const cleanups = [
      window.electronAPI?.onMenuNewFeature?.(() => {
        window.dispatchEvent(new CustomEvent('ramp:new-feature'));
      }),
      window.electronAPI?.onMenuRefresh?.(() => {
        window.dispatchEvent(new CustomEvent('ramp:refresh'));
      }),
      window.electronAPI?.onMenuSettings?.(() => {
        window.dispatchEvent(new CustomEvent('ramp:settings'));
      }),
    ];
    return () => cleanups.forEach(cleanup => cleanup?.());
  }, []);

  // Handler that saves selection to settings
  const handleSelectProject = (projectId: string | null) => {
    setSelectedProjectId(projectId);
    if (projectId) {
      saveSettings.mutate({ lastSelectedProjectId: projectId });
    }
  };

  // Listen for project switch shortcuts (CMD+1-9)
  useEffect(() => {
    const cleanup = window.electronAPI?.onMenuSwitchProject?.((index: number) => {
      if (sortedProjects[index]) {
        handleSelectProject(sortedProjects[index].id);
      }
    });
    return cleanup;
  }, [sortedProjects, handleSelectProject]);

  return (
    <div className="flex h-screen bg-[var(--color-bg)]">
      {/* Auto-update notification */}
      <UpdateNotification />

      {/* Sidebar */}
      <div className="w-64 flex-shrink-0 border-r border-[var(--color-border)] bg-[var(--color-bg-secondary)]">
        {/* Title bar drag region */}
        <div className="titlebar-drag-region h-8 border-b border-[var(--color-border)]" />


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
        <div className="titlebar-drag-region h-8 border-b border-[var(--color-border)] flex items-center justify-center">
          <span className="text-sm font-medium text-[var(--color-text-secondary)]">
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
