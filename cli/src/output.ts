/**
 * Shared output helpers. All commands output JSON by default, or formatted
 * text when --pretty is passed.
 */

export interface OutputOptions {
  pretty: boolean;
}

export function output(data: unknown, opts: OutputOptions): void {
  // JSON.stringify(undefined) returns undefined (not a string), which would
  // print the literal text "undefined" — not valid JSON. Normalise to null.
  const value = data !== undefined ? data : null;
  if (opts.pretty) {
    console.log(JSON.stringify(value, null, 2));
  } else {
    console.log(JSON.stringify(value));
  }
}
