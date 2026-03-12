/**
 * Shared output helpers. All commands output JSON by default, or formatted
 * text when --pretty is passed.
 */

export interface OutputOptions {
  pretty: boolean;
}

export function output(data: unknown, opts: OutputOptions): void {
  if (opts.pretty) {
    console.log(JSON.stringify(data, null, 2));
  } else {
    console.log(JSON.stringify(data));
  }
}
