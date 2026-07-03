#!/usr/bin/env node
/**
 * @permission-slip/cli — agent-facing CLI for Permission Slip
 *
 * Usage:
 *   npx @permission-slip/cli <command> [options]
 *
 * Commands:
 *   register      Generate keys and register with a Permission Slip server
 *   verify        Complete registration with the confirmation code
 *   status        Show registration state, or check approval status
 *   capabilities  List available action configurations and standing approvals
 *   connectors    List available connectors
 *   request        Request approval for an action (auto-approves if standing approval matches)
 *   watch          Poll a pending approval in the background and wake the session on resolve
 *   request-bulk   Request bulk approval for N same-type actions (one notification)
 *   request-status Check the status/outcome of an approval request
 *   changelog      Show CLI updates since your last session (read before multi-step work)
 *   config        Show or update saved configuration and registrations
 *   whoami        Show agent identity and registration info
 *
 * All commands output compact JSON by default. Pass --pretty for pretty-printed JSON.
 */

import { Command } from "commander";
import { registerCommand } from "./commands/register.js";
import { verifyCommand } from "./commands/verify.js";
import { statusCommand } from "./commands/status.js";
import { capabilitiesCommand } from "./commands/capabilities.js";
import { connectorsCommand } from "./commands/connectors.js";
import { requestCommand } from "./commands/request.js";
import { requestBulkCommand } from "./commands/request-bulk.js";
import { requestStatusCommand } from "./commands/request-status.js";
import { changelogCommand } from "./commands/changelog.js";
import { configCommand } from "./commands/config.js";
import { whoamiCommand } from "./commands/whoami.js";
import { autoApproveRequestCommand } from "./commands/autoApproveRequest.js";
import { watchCommand } from "./commands/watch.js";
import { printUnreadChangelogNotice, currentCliVersion } from "./changelog.js";

const program = new Command();

program
  .name("permission-slip")
  .description(
    "Agent-facing CLI for Permission Slip — register, verify, and interact with Permission Slip servers.\n\n" +
    "All commands output compact JSON by default. Pass --pretty for pretty-printed JSON.\n\n" +
    "Server URL (required): use --server, set PS_SERVER, or run: permission-slip config set default_server <url> " +
    "(stored in ~/.permission-slip/config.json). There is no default host.\n\n" +
    "Quick start:\n" +
    "  1. Register:  permission-slip register --invite-code <code>\n" +
    "  2. Verify:    permission-slip verify --code <confirmation_code>\n" +
    "  3. Changelog:  permission-slip changelog   (check for new capabilities)\n" +
    "  4. Discover:  permission-slip capabilities\n" +
    "  5. Bulk work: permission-slip request-bulk --action <type> --actions '[...]'",
  )
  .version(currentCliVersion());

registerCommand(program);
verifyCommand(program);
statusCommand(program);
capabilitiesCommand(program);
connectorsCommand(program);
requestCommand(program);
watchCommand(program);
requestBulkCommand(program);
requestStatusCommand(program);
changelogCommand(program);
configCommand(program);
whoamiCommand(program);
autoApproveRequestCommand(program);

// Surface unread changelog entries unless this IS the changelog command.
const invoked = process.argv[2];
if (invoked !== "changelog") {
  printUnreadChangelogNotice();
}

program.parse(process.argv);
