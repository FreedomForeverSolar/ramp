// API Types - matching Go backend models

export interface Project {
  id: string;
  name: string;
  path: string;
  addedAt: string;
  repos: Repo[];
  features: string[];
  commands: Command[];
  basePort?: number;
  defaultBranchPrefix?: string;
  hasSetupScript: boolean;
  hasCleanupScript: boolean;
}

export interface Repo {
  name: string;
  path: string;
  git: string;
  autoRefresh: boolean;
}

export interface Command {
  name: string;
  command: string;
}

export interface Feature {
  name: string;
  repos: string[];
  created?: string;
  hasUncommittedChanges: boolean;
}

// API Responses
export interface ProjectsResponse {
  projects: Project[];
}

export interface FeaturesResponse {
  features: Feature[];
}

export interface ErrorResponse {
  error: string;
  details?: string;
}

export interface SuccessResponse {
  success: boolean;
  message?: string;
}

// WebSocket Messages
export interface WSMessage {
  type: 'progress' | 'error' | 'complete' | 'connected' | 'output';
  operation?: string;
  message: string;
  percentage?: number;
}

// Request types
export interface AddProjectRequest {
  path: string;
}

export interface CreateFeatureRequest {
  name: string;
}
