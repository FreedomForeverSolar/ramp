/**
 * Sanitize feature name by removing invalid characters as user types.
 * Allows: letters, numbers, hyphens, underscores, and dots
 * Disallows: spaces, slashes, and special characters (~^:?*\[@{)
 *
 * @param input - Raw user input
 * @returns Sanitized string with only valid characters
 */
export function sanitizeFeatureName(input: string): string {
  return input.replace(/[^a-zA-Z0-9_.-]/g, '');
}

/**
 * Check if input was modified during sanitization.
 * Useful for showing feedback to users when characters are filtered.
 *
 * @param raw - Original input
 * @param sanitized - Result of sanitizeFeatureName()
 * @returns true if characters were removed
 */
export function wasInputSanitized(raw: string, sanitized: string): boolean {
  return raw !== sanitized;
}

/**
 * Validation hint message shown when characters are filtered.
 */
export const FEATURE_NAME_VALIDATION_HINT =
  'Invalid characters removed (spaces and special characters are not allowed)';
