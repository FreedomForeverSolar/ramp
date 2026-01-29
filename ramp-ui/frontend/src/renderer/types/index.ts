// API Types - matching Go backend models

export interface Project {
  id: string;
  name: string;
  path: string;
  addedAt: string;
  order: number;
  isFavorite: boolean;
  repos: Repo[];
  features: string[];
  basePort?: number;
  defaultBranchPrefix?: string;
}

export interface Repo {
  name: string;
  path: string;
  git: string;
  localName?: string;
  autoRefresh: boolean;
}

// Git diff statistics for uncommitted changes
export interface DiffStats {
  filesChanged: number;
  insertions: number;
  deletions: number;
}

// Git status statistics for working directory
export interface StatusStats {
  untrackedFiles: number;
  stagedFiles: number;
  modifiedFiles: number;
}

// Detailed status for a single repo worktree
export interface FeatureWorktreeStatus {
  repoName: string;
  branchName: string;
  hasUncommitted: boolean;
  diffStats?: DiffStats;
  statusStats?: StatusStats;
  aheadCount: number;
  behindCount: number;
  isMerged: boolean;
  error?: string;
}

// Feature category type
export type FeatureCategory = 'in_flight' | 'merged' | 'clean';

export interface Feature {
  name: string;
  displayName?: string;
  repos: string[];
  created?: string;
  hasUncommittedChanges: boolean;
  category: FeatureCategory;
  worktreeStatuses?: FeatureWorktreeStatus[];
}

// API Responses
export interface ProjectsResponse {
  projects: Project[];
}

export interface FeaturesResponse {
  features: Feature[];
}

export interface SuccessResponse {
  success: boolean;
  message?: string;
}

// WebSocket Messages
export interface WSMessage {
  type: 'progress' | 'error' | 'complete' | 'connected' | 'output' | 'warning' | 'info' | 'cancelled';
  operation?: string;
  message: string;
  percentage?: number;
  target?: string; // Feature name for filtering messages
  command?: string; // Command name for run operations
}

// Command types
export interface Command {
  name: string;
  command: string;
  scope?: 'source' | 'feature'; // Optional - undefined means available everywhere
}

export interface CommandsResponse {
  commands: Command[];
}

export interface RunCommandRequest {
  featureName?: string; // Optional - if empty, runs against source
  args?: string[]; // Optional - arguments to pass to the script
}

export interface CancelCommandRequest {
  target?: string; // "source" or feature name
}

export interface RunCommandResponse {
  success: boolean;
  exitCode: number;
  duration: number; // milliseconds
  error?: string;
}

// Request types
export interface AddProjectRequest {
  path: string;
}

export interface CreateFeatureRequest {
  name?: string; // Required unless fromBranch is set (auto-derived)
  displayName?: string; // Human-readable display name
  // Optional - branch configuration
  prefix?: string;
  noPrefix?: boolean;
  target?: string;
  // Optional - pre-operation behavior
  autoInstall?: boolean;
  forceRefresh?: boolean;
  skipRefresh?: boolean;
  // For "From Branch" flow - when set, name is auto-derived if not provided
  fromBranch?: string;
}

export interface RenameFeatureRequest {
  displayName: string; // New display name (empty string to clear)
}

// Config types for local preferences
export interface PromptOption {
  value: string;
  label: string;
}

export interface Prompt {
  name: string;
  question: string;
  options: PromptOption[];
  default?: string;
}

export interface ConfigStatusResponse {
  needsConfig: boolean;
  prompts?: Prompt[];
}

export interface ConfigResponse {
  preferences: Record<string, string>;
}

export interface SaveConfigRequest {
  preferences: Record<string, string>;
}

// Project ordering and favorites
export interface ReorderProjectsRequest {
  projectIds: string[];
}

export interface ToggleFavoriteResponse {
  isFavorite: boolean;
}

// Source repo types
export interface SourceRepoStatus {
  name: string;
  branch: string;
  aheadCount: number;
  behindCount: number;
  isInstalled: boolean;
  error?: string;
}

export interface SourceReposResponse {
  repos: SourceRepoStatus[];
}

export interface InstallResponse {
  clonedRepos: string[];
  skippedRepos: string[];
  message: string;
}

// Terminal types
export interface OpenTerminalRequest {
  path: string;
}

// App settings types
export interface AppSettingsResponse {
  terminalApp: string;
  lastSelectedProjectId: string;
  theme: string;
}

export interface SaveAppSettingsRequest {
  terminalApp?: string;
  lastSelectedProjectId?: string;
  theme?: string;
}

// Prune types
export interface PruneFailure {
  name: string;
  error: string;
}

export interface PruneResponse {
  pruned: string[];
  failed: PruneFailure[];
  message: string;
}
