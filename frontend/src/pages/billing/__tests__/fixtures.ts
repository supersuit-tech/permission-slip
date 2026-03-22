/**
 * Shared test fixtures for billing page tests.
 *
 * Provides canonical mock data for billing API responses so individual
 * test files don't duplicate boilerplate.
 */

const freePlanLimits = {
  max_requests_per_month: 1000,
  max_agents: 3,
  max_standing_approvals: 5,
  max_credentials: 5,
  audit_retention_days: 7,
};

export const freePlanResponse = {
  plan: {
    id: "free",
    name: "Free",
    ...freePlanLimits,
  },
  effective_limits: freePlanLimits,
  subscription: {
    status: "active" as const,
    current_period_start: "2026-03-01T00:00:00Z",
    current_period_end: "2026-04-01T00:00:00Z",
    has_payment_method: false,
    can_upgrade: true,
    can_downgrade: false,
    can_end_quota_grace_now: false,
    grace_period_ends_at: null,
    quota_entitlements_until: null,
  },
  usage: {
    requests: 150,
    agents: 2,
    standing_approvals: 3,
    credentials: 1,
  },
  pricing: {
    free_request_allowance: 1000,
    price_per_request_display: "$0.005",
  },
  coupon_redemption_enabled: false,
};

const paidPlanLimits = {
  max_requests_per_month: null as number | null,
  max_agents: null as number | null,
  max_standing_approvals: null as number | null,
  max_credentials: null as number | null,
  audit_retention_days: 90,
};

export const paidPlanResponse = {
  plan: {
    id: "pay_as_you_go",
    name: "Pay As You Go",
    ...paidPlanLimits,
  },
  effective_limits: paidPlanLimits,
  subscription: {
    status: "active" as const,
    current_period_start: "2026-03-01T00:00:00Z",
    current_period_end: "2026-04-01T00:00:00Z",
    has_payment_method: true,
    can_upgrade: false,
    can_downgrade: true,
    can_end_quota_grace_now: false,
    grace_period_ends_at: null,
    quota_entitlements_until: null,
  },
  usage: {
    requests: 1542,
    agents: 5,
    standing_approvals: 10,
    credentials: 8,
  },
  pricing: {
    free_request_allowance: 1000,
    price_per_request_display: "$0.005",
  },
  coupon_redemption_enabled: false,
};

/** Free plan with active paid-quota grace (cancelled pay-as-you-go, period not ended). */
export const freeWithQuotaGraceResponse = {
  ...freePlanResponse,
  effective_limits: paidPlanLimits,
  subscription: {
    ...freePlanResponse.subscription,
    status: "cancelled" as const,
    can_upgrade: true,
    can_downgrade: false,
    can_end_quota_grace_now: true,
    quota_entitlements_until: "2026-04-01T00:00:00Z",
  },
};

/** Paid plan where usage exceeds all free plan limits. */
export const overLimitPaidPlanResponse = {
  ...paidPlanResponse,
  usage: {
    requests: 3000,
    agents: 10,
    standing_approvals: 25,
    credentials: 20,
  },
};

/** Paid plan where usage is exactly at free plan limits (no warnings expected). */
export const atLimitPaidPlanResponse = {
  ...paidPlanResponse,
  usage: {
    requests: 1000,
    agents: 3,
    standing_approvals: 5,
    credentials: 5,
  },
};

export const usageDetailResponse = {
  period_start: "2026-03-01T00:00:00Z",
  period_end: "2026-04-01T00:00:00Z",
  requests: { total: 1542, included: 1000, overage: 542, cost_cents: 271 },
  sms: { total: 5, cost_cents: 5 },
  breakdown: {
    by_agent: { "1": 500, "2": 1042 },
    by_connector: { gmail: 300, stripe: 1242 },
    by_action_type: { "email.send": 300, "payment.create": 1242 },
  },
};

/** Paid plan usage under the free allowance (no overage). */
export const paidUnderAllowanceUsageResponse = {
  period_start: "2026-03-01T00:00:00Z",
  period_end: "2026-04-01T00:00:00Z",
  requests: { total: 80, included: 1000, overage: 0, cost_cents: 0 },
  sms: { total: 0, cost_cents: 0 },
  breakdown: {
    by_agent: { "1": 80 },
    by_connector: {},
    by_action_type: {},
  },
};

export const freeUsageDetailResponse = {
  period_start: "2026-03-01T00:00:00Z",
  period_end: "2026-04-01T00:00:00Z",
  requests: { total: 150, included: 1000, overage: 0, cost_cents: 0 },
  sms: { total: 0, cost_cents: 0 },
  breakdown: {
    by_agent: { "1": 80, "2": 70 },
    by_connector: {},
    by_action_type: {},
  },
};

export const agentsResponse = {
  data: [
    {
      agent_id: 1,
      status: "registered",
      metadata: { name: "Gmail Bot" },
      created_at: "2026-02-01T00:00:00Z",
    },
    {
      agent_id: 2,
      status: "registered",
      metadata: { name: "Stripe Bot" },
      created_at: "2026-02-01T00:00:00Z",
    },
  ],
  has_more: false,
};

export const invoicesResponse = {
  invoices: [
    {
      id: "inv_001",
      date: "2026-02-01T00:00:00Z",
      period_start: "2026-02-01T00:00:00Z",
      period_end: "2026-03-01T00:00:00Z",
      amount_cents: 271,
      status: "paid",
      stripe_invoice_url: "https://invoice.stripe.com/i/test",
    },
  ],
  has_more: false,
};
