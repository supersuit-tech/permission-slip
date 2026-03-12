/**
 * Generates agent-facing instructions for each step of the registration flow.
 *
 * The @permission-slip/cli package handles key generation, signing, config
 * storage, and API calls — so each step is now a single npx command.
 */

// ---------------------------------------------------------------------------
// Step 1 — Invite instructions (sent with the invite code)
// ---------------------------------------------------------------------------

export function generateInviteInstructions(
  inviteCode: string,
  origin: string,
): string {
  return `npx @permission-slip/cli register --invite-code ${inviteCode} --server ${origin}

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
  return `npx @permission-slip/cli verify --code ${confirmationCode} --server ${origin}

Your agent ID is: ${agentId}`;
}

// ---------------------------------------------------------------------------
// Step 3 — Post-registration instructions
// ---------------------------------------------------------------------------

export function generatePostRegistrationInstructions(
  _agentId: number,
  _origin: string,
): string {
  return `npx @permission-slip/cli capabilities

The CLI handles signing, config storage, and API calls automatically.
Run \`npx @permission-slip/cli --help\` for all available commands.`;
}
