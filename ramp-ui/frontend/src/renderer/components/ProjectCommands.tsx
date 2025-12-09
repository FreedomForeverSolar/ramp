import { useState } from 'react';
import { Command, Feature } from '../types';
import RunCommandDialog from './RunCommandDialog';

interface ProjectCommandsProps {
  projectId: string;
  commands: Command[];
  features: Feature[];
}

export default function ProjectCommands({
  projectId,
  commands,
  features,
}: ProjectCommandsProps) {
  const [selectedCommand, setSelectedCommand] = useState<string | null>(null);

  if (commands.length === 0) {
    return null;
  }

  return (
    <div className="mt-4">
      <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-2">
        Commands
      </h3>
      <div className="flex flex-wrap gap-2">
        {commands.map((cmd) => (
          <button
            key={cmd.name}
            onClick={() => setSelectedCommand(cmd.name)}
            className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-md transition-colors"
          >
            {cmd.name}
          </button>
        ))}
      </div>

      {selectedCommand && (
        <RunCommandDialog
          projectId={projectId}
          commandName={selectedCommand}
          features={features}
          onClose={() => setSelectedCommand(null)}
        />
      )}
    </div>
  );
}
