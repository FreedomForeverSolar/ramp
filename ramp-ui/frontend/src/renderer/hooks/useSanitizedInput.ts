import { useState, useCallback } from 'react';
import { type SanitizeFn, SANITIZATION_HINT } from '../utils/validation';

interface SanitizedInput {
  value: string;
  validationHint: string | null;
  onChange: (raw: string) => void;
}

/**
 * Manages an input field that sanitizes user input on change.
 * Returns the sanitized value, a validation hint (shown when characters
 * are filtered), and an onChange handler to wire into input elements.
 *
 * @param sanitize - A sanitization function (e.g., sanitizeFeatureName)
 * @param initialValue - Optional initial value for the input
 */
export function useSanitizedInput(
  sanitize: SanitizeFn,
  initialValue = '',
): SanitizedInput {
  const [value, setValue] = useState(initialValue);
  const [validationHint, setValidationHint] = useState<string | null>(null);

  const onChange = useCallback(
    (raw: string) => {
      const sanitized = sanitize(raw);
      setValue(sanitized);
      setValidationHint(raw !== sanitized ? SANITIZATION_HINT : null);
    },
    [sanitize],
  );

  return { value, validationHint, onChange };
}
