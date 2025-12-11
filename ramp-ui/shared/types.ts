// Shared TypeScript types for Ramp UI
// These types mirror the Go backend models

export interface Project {
  id: string;
  name: string;
  path: string;
  addedAt: string;
  repos: Repo[];
  features: string[];
  commands: Command[];
  basePort?: number;
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

export interface WSMessage {
  type: 'progress' | 'error' | 'complete' | 'connected';
  operation?: string;
  message: string;
  percentage?: number;
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
