import { useState, useMemo, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Feature, FeatureWorktreeStatus, Command, WSMessage } from '../types';
import { useOpenTerminal, usePruneFeatures, useWebSocket } from '../hooks/useRampAPI';
import DeleteFeatureDialog from './DeleteFeatureDialog';
import RunCommandDialog from './RunCommandDialog';
import RenameFeatureDialog from './RenameFeatureDialog';
import DropdownMenu, { DropdownMenuItem, MenuIcons } from './DropdownMenu';

interface FeatureListProps {
  projectId: string;
  projectPath: string;
  features: Feature[];
  commands: Command[];
  isLoading: boolean;
}

// Format status for a single worktree (mirrors CLI formatCompactStatus)
function formatWorktreeStatus(status: FeatureWorktreeStatus): string {
  if (status.error) {
    return `error: ${status.error}`;
  }

  const parts: string[] = [];

  // Show uncommitted changes
  if (status.hasUncommitted) {
    // First try to show diff stats (changes to tracked files)
    if (status.diffStats && (status.diffStats.filesChanged > 0 || status.diffStats.insertions > 0 || status.diffStats.deletions > 0)) {
      const diffParts: string[] = [];
      if (status.diffStats.filesChanged > 0) {
        diffParts.push(`+${status.diffStats.filesChanged}`);
      }
      if (status.diffStats.insertions > 0) {
        diffParts.push(`+${status.diffStats.insertions}`);
      }
      if (status.diffStats.deletions > 0) {
        diffParts.push(`-${status.diffStats.deletions}`);
      }
      parts.push(diffParts.join(' '));
    } else if (status.statusStats) {
      // Show status stats (untracked, staged, modified files)
      const statusParts: string[] = [];
      if (status.statusStats.untrackedFiles > 0) {
        statusParts.push(`${status.statusStats.untrackedFiles} untracked`);
      }
      if (status.statusStats.stagedFiles > 0) {
        statusParts.push(`${status.statusStats.stagedFiles} staged`);
      }
      if (status.statusStats.modifiedFiles > 0) {
        statusParts.push(`${status.statusStats.modifiedFiles} modified`);
      }
      if (statusParts.length > 0) {
        parts.push(statusParts.join(', '));
      } else {
        parts.push('uncommitted');
      }
    } else {
      parts.push('uncommitted');
    }
  }

  // Show ahead status
  if (status.aheadCount > 0) {
    parts.push(`${status.aheadCount} ahead`);
  }

  return parts.join(', ');
}

// Check if a worktree has local work
function hasLocalWork(status: FeatureWorktreeStatus): boolean {
  return status.hasUncommitted || status.aheadCount > 0;
}

// Format feature name with display name if set
function formatFeatureName(feature: Feature): string {
  if (feature.displayName && feature.displayName !== feature.name) {
    return `${feature.displayName} (${feature.name})`;
  }
  return feature.name;
}

export default function FeatureList({
  projectId,
  projectPath,
  features,
  commands,
  isLoading,
}: FeatureListProps) {
  const [deletingFeature, setDeletingFeature] = useState<Feature | null>(null);
  const [renamingFeature, setRenamingFeature] = useState<Feature | null>(null);
  const [openMenu, setOpenMenu] = useState<string | null>(null);
  const [runningCommand, setRunningCommand] = useState<{ commandName: string; featureName: string } | null>(null);
  const [showPruneDialog, setShowPruneDialog] = useState(false);
  const [pruneProgress, setPruneProgress] = useState<string | null>(null);
  const [pruneError, setPruneError] = useState<string | null>(null);
  const menuTriggerRefs = useRef<Map<string, HTMLButtonElement>>(new Map());
  const queryClient = useQueryClient();
  const openTerminal = useOpenTerminal();
  const pruneFeatures = usePruneFeatures(projectId);

  // Listen for WebSocket messages during prune operation
  const handlePruneWSMessage = useCallback((message: unknown) => {
    const msg = message as WSMessage;
    if (msg.operation === 'prune') {
      if (msg.type === 'progress' || msg.type === 'info') {
        setPruneProgress(msg.message);
      } else if (msg.type === 'error') {
        setPruneError(msg.message);
      } else if (msg.type === 'complete') {
        // Get merged feature names to remove (from current features prop)
        const mergedNames = features
          .filter(f => f.category === 'merged')
          .map(f => f.name);

        // Immediately remove pruned features from cache (instant UI update)
        queryClient.setQueryData(
          ['projects', projectId, 'features'],
          (old: { features: Array<{ name: string; category: string }> } | undefined) => {
            if (!old) return old;
            return {
              ...old,
              features: old.features.filter(f => f.category !== 'merged'),
            };
          }
        );

        // Update the project's feature list in the sidebar (for feature count)
        queryClient.setQueryData(
          ['projects'],
          (old: { projects: Array<{ id: string; features: string[] }> } | undefined) => {
            if (!old) return old;
            return {
              ...old,
              projects: old.projects.map(p =>
                p.id === projectId
                  ? { ...p, features: p.features.filter(f => !mergedNames.includes(f)) }
                  : p
              ),
            };
          }
        );

        setPruneProgress(null);
        setShowPruneDialog(false);
      }
    }
  }, [queryClient, projectId, features]);

  useWebSocket(handlePruneWSMessage, showPruneDialog);

  const handleOpenTerminal = async (featureName: string) => {
    const featurePath = `${projectPath}/trees/${featureName}`;
    try {
      await openTerminal.mutateAsync({ path: featurePath });
    } catch (error) {
      console.error('Failed to open terminal:', error);
    }
  };

  // Helper to get menu items for a feature
  const getMenuItems = (feature: Feature): DropdownMenuItem[] => {
    // Filter commands to only show those with scope 'feature' or no scope (available everywhere)
    const featureCommands = commands.filter(cmd => !cmd.scope || cmd.scope === 'feature');
    const items: DropdownMenuItem[] = featureCommands.map((cmd) => ({
      label: cmd.name,
      icon: MenuIcons.play,
      onClick: () => setRunningCommand({ commandName: cmd.name, featureName: feature.name }),
    }));
    items.push({
      label: 'Rename',
      icon: MenuIcons.edit,
      onClick: () => setRenamingFeature(feature),
    });
    items.push({
      label: 'Delete',
      icon: MenuIcons.trash,
      variant: 'danger',
      onClick: () => handleDelete(feature),
    });
    return items;
  };

  // Group features by category
  const groupedFeatures = useMemo(() => {
    const inFlight = features.filter(f => f.category === 'in_flight');
    const merged = features.filter(f => f.category === 'merged');
    const clean = features.filter(f => f.category === 'clean');
    return { inFlight, merged, clean };
  }, [features]);

  const handleDelete = (feature: Feature) => {
    setDeletingFeature(feature);
  };

  const handlePrune = async () => {
    setPruneError(null);
    try {
      await pruneFeatures.mutateAsync();
      // Dialog close is handled by WebSocket 'complete' message
    } catch (error) {
      setPruneError(error instanceof Error ? error.message : 'Prune failed');
    }
  };

  const handleClosePruneDialog = () => {
    setShowPruneDialog(false);
    setPruneProgress(null);
    setPruneError(null);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  if (features.length === 0) {
    return (
      <div className="text-center py-12">
        <svg
          className="mx-auto h-12 w-12 text-gray-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
          />
        </svg>
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
          No features yet
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Create a new feature to get started.
        </p>
      </div>
    );
  }

  // Render a single in-flight feature card
  const renderInFlightFeature = (feature: Feature) => {
    const workingStatuses = feature.worktreeStatuses?.filter(hasLocalWork) || [];
    const isMenuOpen = openMenu === feature.name;

    return (
      <div
        key={feature.name}
        className="relative bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
      >
        <div className="flex items-center justify-between p-4">
          <div className="flex-1">
            <h3 className="font-medium text-gray-900 dark:text-white">
              {formatFeatureName(feature)}
            </h3>
            {/* Show status summary for repos with local work */}
            {workingStatuses.length > 0 && (
              <div className="mt-2 space-y-1">
                {workingStatuses.map((status) => (
                  <div key={status.repoName} className="flex items-center gap-2 text-sm">
                    <span className="text-primary-500 font-medium">&#x25C9;</span>
                    <span className="font-mono text-gray-600 dark:text-gray-400">{status.repoName}:</span>
                    <span className="text-gray-500 dark:text-gray-400">{formatWorktreeStatus(status)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
          <div className="flex items-center gap-1">
            {/* Terminal button */}
            <button
              onClick={() => handleOpenTerminal(feature.name)}
              className="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-md transition-colors"
              title="Open in terminal"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
                />
              </svg>
            </button>

            {/* Actions menu */}
            <button
              ref={(el) => {
                if (el) menuTriggerRefs.current.set(feature.name, el);
              }}
              onClick={() => setOpenMenu(isMenuOpen ? null : feature.name)}
              className="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-md transition-colors"
              title="Actions"
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
              </svg>
            </button>
            <DropdownMenu
              items={getMenuItems(feature)}
              isOpen={isMenuOpen}
              onClose={() => setOpenMenu(null)}
              triggerRef={{ current: menuTriggerRefs.current.get(feature.name) || null }}
            />
          </div>
        </div>
      </div>
    );
  };

  // Render compact feature list (for merged/clean)
  const renderCompactSection = (title: string, featureList: Feature[], count: number, showPruneButton?: boolean) => {
    if (featureList.length === 0) return null;

    return (
      <div className="mt-6">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
            {title} ({count})
          </h3>
          {showPruneButton && featureList.length > 0 && (
            <button
              onClick={() => setShowPruneDialog(true)}
              className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-gray-600 dark:text-gray-400 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors"
              title="Prune all merged features"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Prune All
            </button>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          {featureList.map((feature) => {
            const isMenuOpen = openMenu === `compact-${feature.name}`;
            return (
              <span
                key={feature.name}
                className="group relative inline-flex items-center gap-1 px-2 py-1 bg-gray-100 dark:bg-gray-800 rounded text-sm text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700"
              >
                {formatFeatureName(feature)}
                {/* Terminal button - visible on hover */}
                <button
                  onClick={() => handleOpenTerminal(feature.name)}
                  className="ml-1 opacity-0 group-hover:opacity-100 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-all"
                  title="Open in terminal"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                </button>
                {/* Menu button - visible on hover */}
                <button
                  ref={(el) => {
                    if (el) menuTriggerRefs.current.set(`compact-${feature.name}`, el);
                  }}
                  onClick={() => setOpenMenu(isMenuOpen ? null : `compact-${feature.name}`)}
                  className="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-all"
                  title="Actions"
                >
                  <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
                  </svg>
                </button>
                <DropdownMenu
                  items={getMenuItems(feature)}
                  isOpen={isMenuOpen}
                  onClose={() => setOpenMenu(null)}
                  triggerRef={{ current: menuTriggerRefs.current.get(`compact-${feature.name}`) || null }}
                  align="left"
                />
              </span>
            );
          })}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-4">
      {/* In Flight Section */}
      {groupedFeatures.inFlight.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-3">
            In Flight ({groupedFeatures.inFlight.length})
          </h3>
          <div className="space-y-3">
            {groupedFeatures.inFlight.map(renderInFlightFeature)}
          </div>
        </div>
      )}

      {/* Merged Section - compact list with Prune button */}
      {renderCompactSection('Merged', groupedFeatures.merged, groupedFeatures.merged.length, true)}

      {/* Clean Section - compact list */}
      {renderCompactSection('Clean', groupedFeatures.clean, groupedFeatures.clean.length)}

      {/* Delete Feature Dialog */}
      {deletingFeature && (
        <DeleteFeatureDialog
          projectId={projectId}
          featureName={deletingFeature.name}
          hasUncommittedChanges={deletingFeature.hasUncommittedChanges}
          onClose={() => setDeletingFeature(null)}
        />
      )}

      {/* Run Command Dialog */}
      {runningCommand && (
        <RunCommandDialog
          projectId={projectId}
          commandName={runningCommand.commandName}
          featureName={runningCommand.featureName}
          features={features}
          onClose={() => setRunningCommand(null)}
        />
      )}

      {/* Rename Feature Dialog */}
      {renamingFeature && (
        <RenameFeatureDialog
          projectId={projectId}
          featureName={renamingFeature.name}
          currentDisplayName={renamingFeature.displayName || ''}
          onClose={() => setRenamingFeature(null)}
        />
      )}

      {/* Prune Confirmation Dialog */}
      {showPruneDialog && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                Prune Merged Features
              </h2>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                This will remove {groupedFeatures.merged.length} merged feature{groupedFeatures.merged.length !== 1 ? 's' : ''}:
              </p>
              <ul className="text-sm text-gray-700 dark:text-gray-300 mb-4 max-h-40 overflow-y-auto">
                {groupedFeatures.merged.map((f) => (
                  <li key={f.name} className="py-1 font-mono">{f.name}</li>
                ))}
              </ul>
              <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
                This will delete worktrees, branches, and release allocated ports.
              </p>

              {/* Progress indicator */}
              {pruneProgress && (
                <div className="mb-4 p-3 bg-gray-100 dark:bg-gray-700 rounded-md">
                  <div className="flex items-center gap-2">
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary-500"></div>
                    <span className="text-sm text-gray-700 dark:text-gray-300">{pruneProgress}</span>
                  </div>
                </div>
              )}

              {/* Error display */}
              {pruneError && (
                <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-700 dark:text-red-400">{pruneError}</p>
                </div>
              )}
            </div>

            <div className="flex justify-end gap-3 px-6 py-4 bg-gray-50 dark:bg-gray-900 rounded-b-lg">
              <button
                onClick={handleClosePruneDialog}
                disabled={pruneFeatures.isPending}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handlePrune}
                disabled={pruneFeatures.isPending}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors disabled:opacity-50 flex items-center gap-2"
              >
                {pruneFeatures.isPending && (
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                )}
                {pruneFeatures.isPending ? 'Pruning...' : 'Prune All'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
