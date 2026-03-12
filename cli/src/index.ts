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
 *   status        Show current registration state
 *   capabilities  List available action configurations and standing approvals
 *   connectors    List available connectors
 *   request       Request approval for an action
 *   execute       Execute an approved action
 *   config        Show saved configuration and registrations
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
import { executeCommand } from "./commands/execute.js";
import { configCommand } from "./commands/config.js";
import { whoamiCommand } from "./commands/whoami.js";

const program = new Command();

program
  .name("permission-slip")
  .description(
    "Agent-facing CLI for Permission Slip — register, verify, and interact with Permission Slip servers.\n\n" +
    "All commands output compact JSON by default. Pass --pretty for pretty-printed JSON.\n\n" +
    "Quick start:\n" +
    "  1. Register:  permission-slip register --invite-code <code>\n" +
    "  2. Verify:    permission-slip verify --code <confirmation_code>\n" +
    "  3. Discover:  permission-slip capabilities",
  )
  .version("0.1.0");

registerCommand(program);
verifyCommand(program);
statusCommand(program);
capabilitiesCommand(program);
connectorsCommand(program);
requestCommand(program);
executeCommand(program);
configCommand(program);
whoamiCommand(program);

program.parse(process.argv);
