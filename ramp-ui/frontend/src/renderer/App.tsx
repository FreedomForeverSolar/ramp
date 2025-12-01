import { useState } from 'react';
import { useProjects } from './hooks/useRampAPI';
import ProjectList from './components/ProjectList';
import ProjectView from './components/ProjectView';
import EmptyState from './components/EmptyState';
import { Project } from './types';

function App() {
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);
  const { data: projectsData, isLoading, error } = useProjects();

  const projects = projectsData?.projects ?? [];
  const selectedProject = projects.find((p: Project) => p.id === selectedProjectId);

  // Auto-select first project if none selected
  if (!selectedProjectId && projects.length > 0) {
    setSelectedProjectId(projects[0].id);
  }

  return (
    <div className="flex h-screen bg-white dark:bg-gray-900">
      {/* Sidebar */}
      <div className="w-64 flex-shrink-0 border-r border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
        {/* Title bar drag region */}
        <div className="titlebar-drag-region h-8 border-b border-gray-200 dark:border-gray-700 flex items-center px-20">
          <span className="text-xs font-medium text-gray-500 dark:text-gray-400">Ramp</span>
        </div>

        {/* Project list */}
        <div className="flex flex-col h-[calc(100%-2rem)]">
          <ProjectList
            projects={projects}
            selectedId={selectedProjectId}
            onSelect={setSelectedProjectId}
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
