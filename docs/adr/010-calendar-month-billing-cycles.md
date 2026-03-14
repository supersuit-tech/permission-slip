# ADR-010: Calendar-Month Billing Cycles for Free Tier

**Status:** Accepted
**Date:** 2026-03-14

## Context

We needed to decide how billing cycles work for free-tier users. The main options were:

1. **Per-user billing anchor** — cycle starts on the user's signup date (e.g., sign up Mar 15 → cycle is Mar 15 – Apr 15)
2. **Calendar month with proration** — everyone uses calendar months, but the first partial month is prorated (sign up Mar 15 → get ~50% of the limit for March)
3. **Calendar month, full allowance** — everyone uses calendar months, new users get the full limit regardless of signup date

For paid-tier users, Stripe manages billing anchors natively via subscriptions, and we sync period dates from `invoice.paid` webhooks. This decision only affects the **free tier**.

## Decision

We use **calendar-month cycles with the full allowance** for all free-tier users. Billing periods always run from the 1st of the month (00:00 UTC) to the 1st of the next month, calculated on-demand via `BillingPeriodBounds()`. New users receive the full monthly request limit immediately, regardless of when they sign up.

## Rationale

**Per-user anchors create confusing UX on the free tier.** Showing users "your 250 requests reset on the 23rd" feels arbitrary and is harder to reason about. Clean calendar boundaries ("resets on the 1st") are simple to communicate and understand.

**Proration punishes new users.** If someone signs up on March 25 and gets prorated to ~40 requests for the rest of the month, their first impression is hitting a paywall almost immediately. Generous first experiences convert better — the whole point of a free tier is to let people evaluate the product without friction.

**The "exploit" isn't worth optimizing against.** The worst case is someone signing up on March 31, using 250 requests, then getting another 250 on April 1. That's 500 free requests — a negligible cost that doesn't justify the added complexity of proration or per-user anchors.

**Simplicity reduces bugs.** Calendar-month alignment means one function (`BillingPeriodBounds`) handles all period calculations. No per-user state to track, no edge cases around anchor dates falling on the 29th/30th/31st, no timezone confusion.

## Consequences

- Free-tier users who sign up late in the month get a better deal for their first partial month. This is intentional.
- All free-tier usage reporting and quota enforcement can rely on `BillingPeriodBounds(time.Now())` without consulting user-specific dates.
- Paid-tier billing is unaffected — Stripe continues to manage its own billing cycle anchors, and we sync period dates via the `invoice.paid` webhook.
- If we ever need per-user anchors (e.g., for an enterprise tier with custom billing dates), `BillingPeriodBounds` would need to accept a user-specific anchor parameter. But we should not add this complexity until there's a real need.
