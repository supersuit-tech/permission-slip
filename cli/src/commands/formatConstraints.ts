/**
 * permission-slip auto-approve format — human-readable standing approval constraints.
 */

import type { Command } from "commander";
import {
  formatStandingApprovalConstraints,
  formatStandingApprovalConstraintsText,
  parseConstraintsJson,
} from "../formatStandingApprovalConstraints.js";
import { output, type OutputOptions } from "../output.js";

export function formatConstraintsCommand(program: Command): void {
  const autoApprove = program.commands.find((cmd) => cmd.name() === "auto-approve");
  if (!autoApprove) {
    throw new Error("auto-approve command must be registered before format");
  }

  autoApprove
    .command("format")
    .description("Format standing approval constraints as human-readable text")
    .requiredOption("--constraints <json>", "Constraints JSON object")
    .option("--pretty", "Pretty-printed JSON output")
    .option("--text", "Print plain text instead of JSON")
    .action((opts: { constraints: string; pretty?: boolean; text?: boolean }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const constraints = parseConstraintsJson(opts.constraints);
        const lines = formatStandingApprovalConstraints(constraints);
        const text = formatStandingApprovalConstraintsText(constraints);

        if (opts.text) {
          console.log(text);
          return;
        }

        output({ lines, text }, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
