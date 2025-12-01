import { useAddProject } from '../hooks/useRampAPI';

export default function EmptyState() {
  const addProject = useAddProject();

  const handleAddProject = async () => {
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
    <div className="flex flex-col items-center justify-center h-full p-8">
      <div className="text-center max-w-md">
        {/* Icon */}
        <div className="mx-auto w-24 h-24 bg-primary-100 dark:bg-primary-900/30 rounded-full flex items-center justify-center mb-6">
          <svg
            className="w-12 h-12 text-primary-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
            />
          </svg>
        </div>

        {/* Title */}
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
          Welcome to Ramp
        </h2>

        {/* Description */}
        <p className="text-gray-500 dark:text-gray-400 mb-8">
          Ramp helps you manage multi-repository development workflows using git
          worktrees. Add a project to get started.
        </p>

        {/* CTA Button */}
        <button
          onClick={handleAddProject}
          disabled={addProject.isPending}
          className="inline-flex items-center px-6 py-3 bg-primary-500 hover:bg-primary-600 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
        >
          {addProject.isPending ? (
            <>
              <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white mr-2"></div>
              Adding...
            </>
          ) : (
            <>
              <svg
                className="w-5 h-5 mr-2"
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
              Add Project
            </>
          )}
        </button>

        {/* Help text */}
        <p className="mt-4 text-sm text-gray-400 dark:text-gray-500">
          Select a directory containing a{' '}
          <code className="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded">
            .ramp/ramp.yaml
          </code>{' '}
          file
        </p>
      </div>
    </div>
  );
}
