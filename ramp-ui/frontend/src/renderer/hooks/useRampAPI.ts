import { useEffect, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Project,
  ProjectsResponse,
  FeaturesResponse,
  Feature,
  AddProjectRequest,
  CreateFeatureRequest,
  RenameFeatureRequest,
  SuccessResponse,
  ConfigStatusResponse,
  ConfigResponse,
  SaveConfigRequest,
  CommandsResponse,
  RunCommandRequest,
  RunCommandResponse,
  CancelCommandRequest,
  ToggleFavoriteResponse,
  SourceReposResponse,
  InstallResponse,
  OpenTerminalRequest,
  AppSettingsResponse,
  SaveAppSettingsRequest,
  PruneResponse,
} from '../types';

// Dynamic port configuration - fetched from Electron IPC
// Defaults to production port (37429), dev uses 37430
let backendPort = 37429;
const portInitPromise = window.electronAPI?.getBackendPort().then(port => {
  backendPort = port;
}).catch(() => {
  // Fallback to default port if IPC fails
});

const getApiBase = () => `http://localhost:${backendPort}/api`;
const getWsUrl = () => `ws://localhost:${backendPort}/ws/logs`;

// Helper function for API calls
async function fetchAPI<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  // Ensure port is initialized before making requests
  await portInitPromise;
  const response = await fetch(`${getApiBase()}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

// Projects
export function useProjects() {
  return useQuery<ProjectsResponse>({
    queryKey: ['projects'],
    queryFn: () => fetchAPI<ProjectsResponse>('/projects'),
  });
}

export function useAddProject() {
  const queryClient = useQueryClient();

  return useMutation<Project, Error, AddProjectRequest>({
    mutationFn: (data) =>
      fetchAPI<Project>('/projects', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
  });
}

export function useRemoveProject() {
  const queryClient = useQueryClient();

  return useMutation<SuccessResponse, Error, string>({
    mutationFn: (id) =>
      fetchAPI<SuccessResponse>(`/projects/${id}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
  });
}

export function useReorderProjects() {
  const queryClient = useQueryClient();

  return useMutation<SuccessResponse, Error, string[]>({
    mutationFn: (projectIds) =>
      fetchAPI<SuccessResponse>('/projects/reorder', {
        method: 'PUT',
        body: JSON.stringify({ projectIds }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
  });
}

export function useToggleFavorite() {
  const queryClient = useQueryClient();

  return useMutation<ToggleFavoriteResponse, Error, string>({
    mutationFn: (id) =>
      fetchAPI<ToggleFavoriteResponse>(`/projects/${id}/favorite`, {
        method: 'PUT',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
  });
}

// Features
export function useFeatures(projectId: string) {
  return useQuery<FeaturesResponse>({
    queryKey: ['projects', projectId, 'features'],
    queryFn: () => fetchAPI<FeaturesResponse>(`/projects/${projectId}/features`),
    enabled: !!projectId,
  });
}

export function useCreateFeature(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation<Feature, Error, CreateFeatureRequest>({
    mutationFn: (data) =>
      fetchAPI<Feature>(`/projects/${projectId}/features`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      // Only invalidate features - the dialog handles immediate cache updates via setQueryData
      // This serves as a fallback to ensure data is fresh after HTTP response
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
    },
    onError: () => {
      // Invalidate on error to ensure fresh state (operation may have partially completed)
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
    },
  });
}

export function useDeleteFeature(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation<SuccessResponse, Error, string>({
    mutationFn: (featureName) =>
      fetchAPI<SuccessResponse>(`/projects/${projectId}/features/${featureName}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      // Only invalidate features - the dialog handles immediate cache updates via setQueryData
      // This serves as a fallback to ensure data is fresh after HTTP response
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
    },
    onError: () => {
      // Invalidate on error to ensure fresh state (operation may have partially completed)
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
    },
  });
}

export function usePruneFeatures(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation<PruneResponse, Error, void>({
    mutationFn: () =>
      fetchAPI<PruneResponse>(`/projects/${projectId}/features/prune`, {
        method: 'POST',
      }),
    onSuccess: () => {
      // Only invalidate features - the component handles immediate cache updates via setQueryData
      // This serves as a fallback to ensure data is fresh after HTTP response
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
    },
    onError: () => {
      // Invalidate on error to ensure fresh state (operation may have partially completed)
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
    },
  });
}

export function useRenameFeature(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation<SuccessResponse, Error, { featureName: string } & RenameFeatureRequest>({
    mutationFn: ({ featureName, displayName }) =>
      fetchAPI<SuccessResponse>(`/projects/${projectId}/features/${featureName}/rename`, {
        method: 'PUT',
        body: JSON.stringify({ displayName }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
    },
  });
}

// Config (local preferences)
export function useConfigStatus(projectId: string) {
  return useQuery<ConfigStatusResponse>({
    queryKey: ['projects', projectId, 'config', 'status'],
    queryFn: () => fetchAPI<ConfigStatusResponse>(`/projects/${projectId}/config/status`),
    enabled: !!projectId,
  });
}

export function useConfig(projectId: string) {
  return useQuery<ConfigResponse>({
    queryKey: ['projects', projectId, 'config'],
    queryFn: () => fetchAPI<ConfigResponse>(`/projects/${projectId}/config`),
    enabled: !!projectId,
  });
}

export function useSaveConfig(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation<SuccessResponse, Error, SaveConfigRequest>({
    mutationFn: (data) =>
      fetchAPI<SuccessResponse>(`/projects/${projectId}/config`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'config'] });
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'config', 'status'] });
    },
  });
}

export function useResetConfig(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation<SuccessResponse, Error, void>({
    mutationFn: () =>
      fetchAPI<SuccessResponse>(`/projects/${projectId}/config`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'config'] });
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'config', 'status'] });
    },
  });
}

// Commands
export function useCommands(projectId: string) {
  return useQuery<CommandsResponse>({
    queryKey: ['projects', projectId, 'commands'],
    queryFn: () => fetchAPI<CommandsResponse>(`/projects/${projectId}/commands`),
    enabled: !!projectId,
  });
}

export function useRunCommand(projectId: string) {
  return useMutation<RunCommandResponse, Error, { commandName: string } & RunCommandRequest>({
    mutationFn: ({ commandName, featureName }) =>
      fetchAPI<RunCommandResponse>(`/projects/${projectId}/commands/${commandName}/run`, {
        method: 'POST',
        body: JSON.stringify({ featureName }),
      }),
  });
}

export function useCancelCommand(projectId: string) {
  return useMutation<SuccessResponse, Error, { commandName: string } & CancelCommandRequest>({
    mutationFn: ({ commandName, target }) =>
      fetchAPI<SuccessResponse>(`/projects/${projectId}/commands/${commandName}/cancel`, {
        method: 'POST',
        body: JSON.stringify({ target }),
      }),
  });
}

// WebSocket hook for real-time updates
export function useWebSocket(
  onMessage: (message: unknown) => void,
  enabled: boolean = true
) {
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;

  useEffect(() => {
    if (!enabled) return;

    let ws: WebSocket | null = null;
    let reconnectTimeout: NodeJS.Timeout | null = null;
    let isMounted = true;
    let wasConnected = false;

    const connect = () => {
      if (!isMounted) return;

      // Wait for port initialization before connecting
      portInitPromise?.then(() => {
        if (!isMounted) return;
        ws = new WebSocket(getWsUrl());

        ws.onopen = () => {
          wasConnected = true;
        };

        ws.onmessage = (event) => {
          // Guard against messages arriving after cleanup (React StrictMode)
          if (!isMounted) return;
          try {
            const message = JSON.parse(event.data);
            onMessageRef.current(message);
          } catch (e) {
            console.error('Failed to parse WebSocket message:', e);
          }
        };

        ws.onclose = () => {
          // Only reconnect if we were actually connected and still mounted
          // (avoids noise from React Strict Mode double-mounting)
          if (wasConnected && isMounted) {
            reconnectTimeout = setTimeout(connect, 2000);
          }
        };

        ws.onerror = () => {
          // Suppress error logging - onclose handles reconnection
        };
      });
    };

    connect();

    return () => {
      isMounted = false;
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
      // Close WebSocket regardless of state - close() is safe on CONNECTING
      // and prevents zombie connections in React StrictMode
      if (ws) {
        ws.close();
      }
    };
  }, [enabled]);
}

// Source Repos
export function useSourceRepos(projectId: string) {
  return useQuery<SourceReposResponse>({
    queryKey: ['projects', projectId, 'source-repos'],
    queryFn: () => fetchAPI<SourceReposResponse>(`/projects/${projectId}/source-repos`),
    enabled: !!projectId,
  });
}

export function useRefreshSourceRepos(projectId: string) {
  return useMutation<SuccessResponse, Error, void>({
    mutationFn: () =>
      fetchAPI<SuccessResponse>(`/projects/${projectId}/source-repos/refresh`, {
        method: 'POST',
      }),
    // No onSuccess invalidation - refresh is async and the WebSocket handler
    // in SourceRepoList refetches when the operation actually completes
  });
}

export function useInstallRepos(projectId: string) {
  return useMutation<InstallResponse, Error, void>({
    mutationFn: () =>
      fetchAPI<InstallResponse>(`/projects/${projectId}/source-repos/install`, {
        method: 'POST',
      }),
    // No onSuccess invalidation - install is async and the WebSocket handler
    // in SourceRepoList refetches when the operation actually completes
  });
}

// Terminal
export function useOpenTerminal() {
  return useMutation<SuccessResponse, Error, OpenTerminalRequest>({
    mutationFn: (data) =>
      fetchAPI<SuccessResponse>('/terminal/open', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  });
}

// App Settings
export function useAppSettings() {
  return useQuery<AppSettingsResponse>({
    queryKey: ['settings'],
    queryFn: () => fetchAPI<AppSettingsResponse>('/settings'),
  });
}

export function useSaveAppSettings() {
  const queryClient = useQueryClient();

  return useMutation<SuccessResponse, Error, SaveAppSettingsRequest>({
    mutationFn: (data) =>
      fetchAPI<SuccessResponse>('/settings', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: (_data, variables) => {
      // Update cache directly instead of refetching - we know what we just saved
      queryClient.setQueryData<AppSettingsResponse>(['settings'], (old) => {
        if (!old) return old;
        return {
          terminalApp: variables.terminalApp ?? old.terminalApp,
          lastSelectedProjectId: variables.lastSelectedProjectId ?? old.lastSelectedProjectId,
          theme: variables.theme ?? old.theme,
        };
      });
    },
  });
}
