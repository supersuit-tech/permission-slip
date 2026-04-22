package api

// Max merged approval context JSON size (bytes) after enriching from resource_details
// (e.g. slack_context, email_thread). Must stay in sync with approvals.context DB check
// and AgentRequestApproval request validation (issue #1004).
const maxApprovalContextJSONBytes = 262144
