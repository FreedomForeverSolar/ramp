import { useAddProject } from '../hooks/useRampAPI';
import { Project } from '../types';

interface ProjectListProps {
  projects: Project[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  isLoading: boolean;
}

export default function ProjectList({
  projects,
  selectedId,
  onSelect,
  isLoading,
}: ProjectListProps) {
  const addProject = useAddProject();

  const handleAddProject = async () => {
    // Use Electron's native dialog if available
    const path = window.electronAPI?.selectDirectory
      ? await window.electronAPI.selectDirectory()
      : prompt('Enter project path:');

    if (path) {
      try {
        await addProject.mutateAsync({ path });
      } catch (error) {
        alert(`Failed to add project: ${error instanceof Error ? error.message : 'Unknown error'}`);
      }
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-3 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
            Projects
          </h2>
          <button
            onClick={handleAddProject}
            disabled={addProject.isPending}
            className="titlebar-no-drag p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-600 dark:text-gray-300 disabled:opacity-50"
            title="Add Project"
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
                d="M12 4v16m8-8H4"
              />
            </svg>
          </button>
        </div>
      </div>

      {/* Project list */}
      <div className="flex-1 overflow-auto p-2">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-500"></div>
          </div>
        ) : projects.length === 0 ? (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400 text-sm">
            No projects yet
          </div>
        ) : (
          <ul className="space-y-1">
            {projects.map((project) => (
              <li key={project.id}>
                <button
                  onClick={() => onSelect(project.id)}
                  className={`titlebar-no-drag w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${
                    selectedId === project.id
                      ? 'bg-primary-500 text-white'
                      : 'hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300'
                  }`}
                >
                  <div className="font-medium truncate">{project.name}</div>
                  <div
                    className={`text-xs truncate ${
                      selectedId === project.id
                        ? 'text-primary-100'
                        : 'text-gray-500 dark:text-gray-400'
                    }`}
                  >
                    {(project.features?.length ?? 0)} feature
                    {(project.features?.length ?? 0) !== 1 ? 's' : ''}
                  </div>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
