/**
 * Generates agent-facing instructions for each step of the registration flow.
 *
 * The @permission-slip/cli package handles key generation, signing, config
 * storage, and API calls — so each step is now a single npx command.
 */

// ---------------------------------------------------------------------------
// Step 1 — Invite instructions (sent with the invite code)
// ---------------------------------------------------------------------------

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, "'\\''")}'`;
}

export function generateInviteInstructions(
  inviteCode: string,
  origin: string,
): string {
  return `npx @permission-slip/cli register --invite-code ${shellQuote(inviteCode)} --server ${shellQuote(origin)}

The CLI will generate your signing key (if you don't have one) and register automatically.
Save the agent_id from the response — you'll need to share it with the user so they can send you the confirmation code.`;
}

// ---------------------------------------------------------------------------
// Step 2 — Verification instructions (sent with the confirmation code)
// ---------------------------------------------------------------------------

export function generateVerificationInstructions(
  agentId: number,
  confirmationCode: string,
  origin: string,
): string {
  // Pass --agent-id explicitly so the command works without relying on a
  // saved registration in ~/.permission-slip/registrations.json. Without
  // this, verify silently fails locally (never reaches the server) on any
  // machine that didn't run `register` against this exact --server URL.
  return `npx @permission-slip/cli verify --agent-id ${agentId} --code ${shellQuote(confirmationCode)} --server ${shellQuote(origin)}`;
}

// ---------------------------------------------------------------------------
// Step 3 — Post-registration instructions
// ---------------------------------------------------------------------------

export function generatePostRegistrationInstructions(
  _agentId: number,
  origin: string,
): string {
  return `npx @permission-slip/cli capabilities --server ${shellQuote(origin)}

The CLI handles signing, config storage, and API calls automatically.
Run \`npx @permission-slip/cli --help\` for all available commands.`;
}
