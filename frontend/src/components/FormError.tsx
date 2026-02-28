interface FormErrorProps {
  /** The error message to display, or null/undefined to render nothing. */
  error: string | null | undefined;
  /**
   * When true, prefixes the message with a bold "Error:" label.
   * Used in auth-flow forms; omitted in dialogs where context is obvious.
   */
  prefix?: boolean;
}

/**
 * Inline form error message with consistent styling and accessibility.
 * Renders nothing when `error` is falsy.
 */
export function FormError({ error, prefix = false }: FormErrorProps) {
  if (!error) return null;

  return (
    <p className="text-sm text-destructive" role="alert">
      {prefix && <strong>Error: </strong>}
      {error}
    </p>
  );
}
