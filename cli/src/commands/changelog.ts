/**
 * permission-slip changelog [--mark-read] [--since <version>]
 *
 * Shows CLI changes agents should know about. Run at the start of each session.
 */

import type { Command } from "commander";
import {
  currentCliVersion,
  entriesSinceVersion,
  getLastChangelogVersion,
  loadChangelogEntries,
  markChangelogRead,
} from "../changelog.js";
import { output, type OutputOptions } from "../output.js";

export function changelogCommand(program: Command): void {
  program
    .command("changelog")
    .description(
      "Show CLI changes since your last use — run at the start of each agent session",
    )
    .option("--since <version>", "Show entries newer than this version")
    .option("--mark-read", "Record current CLI version as read after displaying")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action((opts: { since?: string; markRead?: boolean; pretty?: boolean }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      const entries = loadChangelogEntries();
      const since = opts.since ?? getLastChangelogVersion();
      const unread = entriesSinceVersion(entries, since);

      output(
        {
          current_version: currentCliVersion(),
          last_read_version: getLastChangelogVersion() ?? null,
          unread_count: unread.length,
          agent_note:
            unread.length > 0
              ? "Review unread entries and update your workflow notes. Prefer request-bulk for multiple same-type actions."
              : "No unread changelog entries.",
          entries: unread.map((e) => ({
            version: e.version,
            date: e.date ?? null,
            summary: e.body.split("\n").slice(0, 8).join("\n"),
          })),
        },
        outputOpts,
      );

      if (opts.markRead) {
        markChangelogRead();
      }
    });
}
