/**
 * Centralized PostHog event names.
 *
 * All product analytics events must be defined here. This ensures:
 *  - Autocomplete and type safety when calling trackEvent()
 *  - A single place to audit every event the app tracks
 *  - Typos are caught at compile time, not silently in PostHog
 *
 * Naming convention: <entity>_<past_tense_action>
 *   e.g., "approval_approved", "standing_approval_created"
 */

export const PostHogEvents = {
  // Approval workflow
  APPROVAL_APPROVED: "approval_approved",
  APPROVAL_DENIED: "approval_denied",

  // Standing approvals
  STANDING_APPROVAL_CREATED: "standing_approval_created",
  STANDING_APPROVAL_REVOKED: "standing_approval_revoked",

  // Agent management
  AGENT_UPDATED: "agent_updated",
  AGENT_DEACTIVATED: "agent_deactivated",

  // Agent registration
  INVITE_CREATED: "invite_created",

  // Notification settings
  NOTIFICATION_PREFERENCES_UPDATED: "notification_preferences_updated",

  // Marketing preferences
  MARKETING_OPT_IN_UPDATED: "marketing_opt_in_updated",
} as const;

export type PostHogEventName =
  (typeof PostHogEvents)[keyof typeof PostHogEvents];
