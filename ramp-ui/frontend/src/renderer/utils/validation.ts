/**
 * Sanitize feature name by removing invalid characters as user types.
 * Allows: letters, numbers, hyphens, underscores, and dots.
 */
export function sanitizeFeatureName(input: string): string {
  return input.replace(/[^a-zA-Z0-9_.-]/g, '');
}

/**
 * Sanitize branch name by removing invalid characters as user types.
 * Like sanitizeFeatureName but also allows forward slashes for
 * branch prefixes (e.g., "feature/") and branch paths.
 */
export function sanitizeBranchName(input: string): string {
  return input.replace(/[^a-zA-Z0-9_.\-/]/g, '');
}

/** Type for any sanitization function accepted by useSanitizedInput. */
export type SanitizeFn = (input: string) => string;

/** Validation hint shown when invalid characters are filtered from input. */
export const SANITIZATION_HINT =
  'Invalid characters removed (spaces and special characters are not allowed)';
