import { useMemo, useState, useEffect } from 'react';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { useAddProject, useReorderProjects, useToggleFavorite, useRemoveProject } from '../hooks/useRampAPI';
import { Project } from '../types';
import GlobalSettingsDialog from './GlobalSettingsDialog';

interface ProjectListProps {
  projects: Project[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  isLoading: boolean;
}

// Star icon component (filled and outline variants)
function StarIcon({ filled, className }: { filled: boolean; className?: string }) {
  return filled ? (
    <svg className={className} fill="currentColor" viewBox="0 0 24 24">
      <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
    </svg>
  ) : (
    <svg className={className} fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"
      />
    </svg>
  );
}

// Sortable project item component
interface SortableProjectItemProps {
  project: Project;
  isSelected: boolean;
  onSelect: () => void;
  onToggleFavorite: () => void;
  onContextMenu: (e: React.MouseEvent) => void;
}

function SortableProjectItem({
  project,
  isSelected,
  onSelect,
  onToggleFavorite,
  onContextMenu,
}: SortableProjectItemProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: project.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const handleStarClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onToggleFavorite();
  };

  return (
    <li ref={setNodeRef} style={style} {...attributes}>
      <div
        className={`titlebar-no-drag relative flex items-center w-full text-left px-3 py-2 rounded-md text-sm transition-colors cursor-grab active:cursor-grabbing ${
          isSelected
            ? 'bg-primary-500 text-white'
            : 'hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300'
        }`}
        {...listeners}
        onClick={onSelect}
        onContextMenu={onContextMenu}
      >
        {/* Star button */}
        <button
          onClick={handleStarClick}
          className={`flex-shrink-0 mr-2 p-0.5 rounded hover:bg-black/10 dark:hover:bg-white/10 ${
            project.isFavorite
              ? isSelected
                ? 'text-yellow-300'
                : 'text-yellow-500'
              : isSelected
              ? 'text-white/50 hover:text-white'
              : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'
          }`}
          title={project.isFavorite ? 'Remove from favorites' : 'Add to favorites'}
        >
          <StarIcon filled={project.isFavorite} className="w-4 h-4" />
        </button>

        {/* Project info */}
        <div className="flex-1 min-w-0">
          <div className="font-medium truncate">{project.name}</div>
          <div
            className={`text-xs truncate ${
              isSelected
                ? 'text-primary-100'
                : 'text-gray-500 dark:text-gray-400'
            }`}
          >
            {(project.features?.length ?? 0)} feature
            {(project.features?.length ?? 0) !== 1 ? 's' : ''}
          </div>
        </div>
      </div>
    </li>
  );
}

export default function ProjectList({
  projects,
  selectedId,
  onSelect,
  isLoading,
}: ProjectListProps) {
  const addProject = useAddProject();
  const reorderProjects = useReorderProjects();
  const toggleFavorite = useToggleFavorite();
  const removeProject = useRemoveProject();
  const [showSettings, setShowSettings] = useState(false);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; project: Project } | null>(null);

  // Sort projects: favorites first, then by order
  const sortedProjects = useMemo(() => {
    return [...projects].sort((a, b) => {
      // Favorites first
      if (a.isFavorite !== b.isFavorite) {
        return a.isFavorite ? -1 : 1;
      }
      // Then by order
      return a.order - b.order;
    });
  }, [projects]);

  // Local state for optimistic reordering
  const [localProjects, setLocalProjects] = useState<Project[] | null>(null);
  const displayProjects = localProjects ?? sortedProjects;

  // Reset local state when server data changes
  useEffect(() => {
    setLocalProjects(null);
  }, [sortedProjects]);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8, // Require 8px drag before starting
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = displayProjects.findIndex((p) => p.id === active.id);
      const newIndex = displayProjects.findIndex((p) => p.id === over.id);

      const newOrder = arrayMove(displayProjects, oldIndex, newIndex);

      // Optimistic update
      setLocalProjects(newOrder);

      // Send to server - extract IDs in new order
      const projectIds = newOrder.map((p) => p.id);
      reorderProjects.mutate(projectIds, {
        onError: () => {
          // Revert on error
          setLocalProjects(null);
        },
      });
    }
  };

  const handleToggleFavorite = (projectId: string) => {
    toggleFavorite.mutate(projectId);
  };

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

  const handleContextMenu = (e: React.MouseEvent, project: Project) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, project });
  };

  const handleRemoveProject = async () => {
    if (!contextMenu) return;
    const { project } = contextMenu;
    setContextMenu(null);
    if (confirm(`Remove "${project.name}" from Ramp UI?\n\nThis will not delete any files.`)) {
      await removeProject.mutateAsync(project.id);
    }
  };

  // Close context menu on click outside or Escape
  useEffect(() => {
    if (!contextMenu) return;

    const handleClickOutside = () => setContextMenu(null);
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setContextMenu(null);
    };

    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [contextMenu]);

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
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={displayProjects.map((p) => p.id)}
              strategy={verticalListSortingStrategy}
            >
              <ul className="space-y-1">
                {displayProjects.map((project) => (
                  <SortableProjectItem
                    key={project.id}
                    project={project}
                    isSelected={selectedId === project.id}
                    onSelect={() => onSelect(project.id)}
                    onToggleFavorite={() => handleToggleFavorite(project.id)}
                    onContextMenu={(e) => handleContextMenu(e, project)}
                  />
                ))}
              </ul>
            </SortableContext>
          </DndContext>
        )}
      </div>

      {/* Settings button - sticky at bottom */}
      <div className="p-2 border-t border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setShowSettings(true)}
          className="titlebar-no-drag w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors"
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
              d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
            />
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
            />
          </svg>
          Settings
        </button>
      </div>

      {/* Global Settings Dialog */}
      {showSettings && (
        <GlobalSettingsDialog onClose={() => setShowSettings(false)} />
      )}

      {/* Context Menu */}
      {contextMenu && (
        <div
          className="fixed z-50 bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[140px]"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <button
            onClick={handleRemoveProject}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700"
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
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
            Remove
          </button>
        </div>
      )}
    </div>
  );
}
