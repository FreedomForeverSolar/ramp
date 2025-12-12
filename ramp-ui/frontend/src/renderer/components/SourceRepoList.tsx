import { useState, useRef, useEffect, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useSourceRepos, useRefreshSourceRepos, useOpenTerminal, useWebSocket } from '../hooks/useRampAPI';
import { Command, Feature, Repo, WSMessage } from '../types';
import RunCommandDialog from './RunCommandDialog';

interface SourceRepoListProps {
  projectId: string;
  projectPath: string;
  repoConfigs: Repo[];
  commands: Command[];
  features: Feature[];
}

export default function SourceRepoList({
  projectId,
  projectPath,
  repoConfigs,
  commands,
  features,
}: SourceRepoListProps) {
  const queryClient = useQueryClient();
  const { data: sourceReposData, isLoading, refetch } = useSourceRepos(projectId);
  const refreshSourceRepos = useRefreshSourceRepos(projectId);
  const openTerminal = useOpenTerminal();
  const [showCommandDropdown, setShowCommandDropdown] = useState(false);
  const [selectedCommand, setSelectedCommand] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isCheckingStatus, setIsCheckingStatus] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowCommandDropdown(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Handle WebSocket messages for refresh operation
  const handleWSMessage = useCallback((message: unknown) => {
    const msg = message as WSMessage;
    if (msg.operation !== 'refresh' || msg.target !== 'source') return;

    if (msg.type === 'complete' || msg.type === 'error') {
      setIsRefreshing(false);
      refetch();
      // Also refresh features since branch status may have changed
      queryClient.refetchQueries({ queryKey: ['projects', projectId, 'features'] });
    }
  }, [refetch, queryClient, projectId]);

  useWebSocket(handleWSMessage, isRefreshing);

  const handleCheckStatus = async () => {
    setIsCheckingStatus(true);
    try {
      await refetch();
    } finally {
      setIsCheckingStatus(false);
    }
  };

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await refreshSourceRepos.mutateAsync();
    } catch {
      setIsRefreshing(false);
    }
  };

  const handleOpenTerminal = async () => {
    // Open to the repos directory (use first repo's path from config)
    const reposPath = repoConfigs.length > 0 && repoConfigs[0].path
      ? `${projectPath}/${repoConfigs[0].path}`
      : projectPath;
    try {
      await openTerminal.mutateAsync({ path: reposPath });
    } catch (error) {
      console.error('Failed to open terminal:', error);
    }
  };

  const handleRunCommand = (commandName: string) => {
    setShowCommandDropdown(false);
    setSelectedCommand(commandName);
  };

  const repos = sourceReposData?.repos ?? [];

  // Helper to render status indicator
  const renderStatus = (repo: typeof repos[0]) => {
    if (!repo.isInstalled) {
      return (
        <span className="text-xs text-gray-400 dark:text-gray-500">
          Not installed
        </span>
      );
    }

    if (repo.error) {
      return (
        <span className="text-xs text-red-500" title={repo.error}>
          Error
        </span>
      );
    }

    if (repo.aheadCount === 0 && repo.behindCount === 0) {
      return (
        <span className="text-xs text-green-600 dark:text-green-400">
          up to date
        </span>
      );
    }

    const parts = [];
    if (repo.aheadCount > 0) {
      parts.push(
        <span key="ahead" className="text-blue-600 dark:text-blue-400">
          {repo.aheadCount}
        </span>
      );
    }
    if (repo.behindCount > 0) {
      parts.push(
        <span key="behind" className="text-orange-600 dark:text-orange-400">
          {repo.behindCount}
        </span>
      );
    }

    return (
      <span className="text-xs flex items-center gap-1">
        {repo.aheadCount > 0 && (
          <span className="flex items-center text-blue-600 dark:text-blue-400" title={`${repo.aheadCount} commits ahead`}>
            <svg className="w-3 h-3 mr-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
            </svg>
            {repo.aheadCount}
          </span>
        )}
        {repo.behindCount > 0 && (
          <span className="flex items-center text-orange-600 dark:text-orange-400" title={`${repo.behindCount} commits behind`}>
            <svg className="w-3 h-3 mr-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
            </svg>
            {repo.behindCount}
          </span>
        )}
      </span>
    );
  };

  return (
    <div className="mb-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
          Source Repositories
        </h3>
        <div className="flex items-center gap-2">
          {/* Check status button */}
          <button
            onClick={handleCheckStatus}
            disabled={isCheckingStatus || isRefreshing}
            className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors disabled:opacity-50"
            title="Check status (fetch without pull)"
          >
            <svg
              className={`w-4 h-4 ${isCheckingStatus ? 'animate-pulse' : ''}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </button>

          {/* Refresh button */}
          <button
            onClick={handleRefresh}
            disabled={isRefreshing || isCheckingStatus}
            className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors disabled:opacity-50"
            title="Refresh repositories (pull changes)"
          >
            <svg
              className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
              />
            </svg>
          </button>

          {/* Terminal button */}
          <button
            onClick={handleOpenTerminal}
            className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors"
            title="Open in terminal"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
          </button>

          {/* Run command dropdown */}
          {commands.length > 0 && (
            <div className="relative" ref={dropdownRef}>
              <button
                onClick={() => setShowCommandDropdown(!showCommandDropdown)}
                className="flex items-center gap-1 px-2 py-1 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors"
              >
                Run
                <svg
                  className={`w-3 h-3 transition-transform ${showCommandDropdown ? 'rotate-180' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {showCommandDropdown && (
                <div className="absolute right-0 mt-1 w-40 bg-white dark:bg-gray-800 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-10">
                  <div className="py-1">
                    {commands.map((cmd) => (
                      <button
                        key={cmd.name}
                        onClick={() => handleRunCommand(cmd.name)}
                        className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                      >
                        {cmd.name}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Repo list */}
      <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg overflow-hidden">
        {isLoading ? (
          <div className="p-4">
            <div className="animate-pulse space-y-2">
              <div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-1/2"></div>
              <div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-1/3"></div>
            </div>
          </div>
        ) : repos.length === 0 ? (
          <div className="p-4 text-sm text-gray-500 dark:text-gray-400">
            No repositories configured
          </div>
        ) : (
          <div className="divide-y divide-gray-200 dark:divide-gray-600">
            {repos.map((repo) => (
              <div
                key={repo.name}
                className="flex items-center justify-between px-4 py-2"
              >
                <div className="flex items-center gap-3">
                  <span className="font-medium text-sm text-gray-900 dark:text-white">
                    {repo.name}
                  </span>
                  {repo.isInstalled && repo.branch && (
                    <span className="text-xs font-mono text-gray-500 dark:text-gray-400 bg-gray-200 dark:bg-gray-600 px-1.5 py-0.5 rounded">
                      {repo.branch}
                    </span>
                  )}
                </div>
                {renderStatus(repo)}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Run Command Dialog */}
      {selectedCommand && (
        <RunCommandDialog
          projectId={projectId}
          commandName={selectedCommand}
          features={features}
          runImmediately // Skip selection, run against source immediately
          onClose={() => setSelectedCommand(null)}
        />
      )}
    </div>
  );
}
