import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Project,
  ProjectsResponse,
  FeaturesResponse,
  Feature,
  AddProjectRequest,
  CreateFeatureRequest,
  SuccessResponse,
} from '../types';

const API_BASE = 'http://localhost:37429/api';

// Helper function for API calls
async function fetchAPI<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
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

export function useProject(id: string) {
  return useQuery<Project>({
    queryKey: ['projects', id],
    queryFn: () => fetchAPI<Project>(`/projects/${id}`),
    enabled: !!id,
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
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
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
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'features'] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
  });
}

// Commands
export function useRunCommand(projectId: string) {
  return useMutation<SuccessResponse, Error, { name: string; featureName?: string }>({
    mutationFn: ({ name, featureName }) =>
      fetchAPI<SuccessResponse>(`/projects/${projectId}/commands/${name}/run`, {
        method: 'POST',
        body: JSON.stringify({ featureName }),
      }),
  });
}

// WebSocket hook for real-time updates
export function useWebSocket(onMessage: (message: unknown) => void) {
  const wsRef = { current: null as WebSocket | null };

  const connect = () => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    const ws = new WebSocket('ws://localhost:37429/ws/logs');

    ws.onopen = () => {
      console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        onMessage(message);
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      // Reconnect after a delay
      setTimeout(connect, 2000);
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    wsRef.current = ws;
  };

  const disconnect = () => {
    wsRef.current?.close();
    wsRef.current = null;
  };

  return { connect, disconnect };
}
