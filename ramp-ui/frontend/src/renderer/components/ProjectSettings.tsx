import { useState } from 'react';
import { useConfig, useConfigStatus } from '../hooks/useRampAPI';
import ConfigPromptsDialog from './ConfigPromptsDialog';

interface ProjectSettingsProps {
  projectId: string;
}

export default function ProjectSettings({ projectId }: ProjectSettingsProps) {
  const { data: configStatus, refetch: refetchStatus } = useConfigStatus(projectId);
  const { data: configData } = useConfig(projectId);
  const [showConfigDialog, setShowConfigDialog] = useState(false);

  const handleOpenDialog = async () => {
    // Refetch to ensure we have latest prompts
    await refetchStatus();
    setShowConfigDialog(true);
  };

  // Check if user has configured preferences
  const hasPreferences = configData?.preferences && Object.keys(configData.preferences).length > 0;

  // Only show if prompts are defined in ramp.yaml
  if (!configStatus?.prompts || configStatus.prompts.length === 0) {
    return null;
  }

  return (
    <>
      <button
        onClick={handleOpenDialog}
        className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-md transition-colors"
        title={hasPreferences ? 'Edit local preferences' : 'Configure local preferences'}
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
            d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
          />
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
          />
        </svg>
      </button>

      {showConfigDialog && configStatus?.prompts && (
        <ConfigPromptsDialog
          projectId={projectId}
          prompts={configStatus.prompts}
          existingPreferences={configData?.preferences}
          onClose={() => setShowConfigDialog(false)}
          onSaved={() => setShowConfigDialog(false)}
        />
      )}
    </>
  );
}
