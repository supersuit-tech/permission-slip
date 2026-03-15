# Stripe Setup Guide

This guide covers setting up Stripe for Permission Slip — both for **production** and **local development/testing**.

Billing is gated behind `BILLING_ENABLED` (defaults to `false`). When disabled, all users get an unlimited plan and Stripe is entirely skipped. Enable it only when you're ready to charge users.

## Prerequisites

- A [Stripe account](https://stripe.com) (free to create)
- Permission Slip deployed or running locally

## Production Setup

### 1. Get API keys

1. Go to [Stripe Dashboard → Developers → API keys](https://dashboard.stripe.com/apikeys)
2. Copy the **Secret key** (starts with `sk_live_`) — this is `STRIPE_SECRET_KEY`
3. Copy the **Publishable key** (starts with `pk_live_`) — this is `STRIPE_PUBLISHABLE_KEY`

### 2. Create a metered Price for per-request billing

This creates a usage-based price so users are charged based on API request volume (250 free per month, then $0.005/request). See `config/plans.json` for the current free-tier limit.

1. Go to [Stripe Dashboard → Products](https://dashboard.stripe.com/products)
2. Click **+ Add product**
3. Set the product name (e.g., "API Requests")
4. Under **Price information**:
   - Pricing model: **Usage-based** (metered billing)
   - Price per unit: **$0.005**
   - Billing period: **Monthly**
   - Usage type: **Metered** (reported to Stripe during the period)
5. Click **Save product**
6. On the product page, copy the **Price ID** (starts with `price_`) — this is `STRIPE_PRICE_ID_REQUEST`

### 3. Set up the webhook endpoint

Webhooks let Stripe notify the server when payment events happen (successful checkout, failed payment, subscription changes).

1. Go to [Stripe Dashboard → Developers → Webhooks](https://dashboard.stripe.com/webhooks)
2. Click **+ Add endpoint**
3. Set the endpoint URL to: `https://app.permissionslip.dev/api/webhooks/stripe`
   - For self-hosted deployments, use your `BASE_URL` + `/api/webhooks/stripe`
4. Under **Select events to listen to**, add these 5 events:
   - `checkout.session.completed` — user completed payment
   - `invoice.paid` — invoice payment succeeded
   - `invoice.payment_failed` — payment collection failed
   - `customer.subscription.updated` — subscription status changed
   - `customer.subscription.deleted` — user cancelled subscription
5. Click **Add endpoint**
6. On the endpoint detail page, click **Reveal** next to **Signing secret** and copy it (starts with `whsec_`) — this is `STRIPE_WEBHOOK_SECRET`

### 4. Deploy the secrets

#### Fly.io

```bash
fly secrets set \
  BILLING_ENABLED="true" \
  STRIPE_SECRET_KEY="sk_live_xxxx" \
  STRIPE_PUBLISHABLE_KEY="pk_live_xxxx" \
  STRIPE_WEBHOOK_SECRET="whsec_xxxx" \
  STRIPE_PRICE_ID_REQUEST="price_xxxx"
```

The frontend publishable key is set in `fly.toml` under `[build.args]`:

```toml
[build.args]
  VITE_STRIPE_PUBLISHABLE_KEY = "pk_live_xxxx"
```

#### Self-hosted / Docker

Set these as environment variables:

```bash
BILLING_ENABLED=true
STRIPE_SECRET_KEY=sk_live_xxxx
STRIPE_PUBLISHABLE_KEY=pk_live_xxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxx
STRIPE_PRICE_ID_REQUEST=price_xxxx
```

Pass the frontend key as a Docker build arg:

```bash
docker build --build-arg VITE_STRIPE_PUBLISHABLE_KEY="pk_live_xxxx" ...
```

### Startup validation

When `BILLING_ENABLED=true`:

- The server **refuses to start** if any of `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, or `BASE_URL` are missing
- `STRIPE_PRICE_ID_REQUEST` only logs a warning if missing, but checkout will fail without it

| Secret | Where to find it |
|--------|-----------------|
| `STRIPE_SECRET_KEY` | Stripe Dashboard → Developers → API keys → Secret key |
| `STRIPE_PUBLISHABLE_KEY` | Stripe Dashboard → Developers → API keys → Publishable key |
| `STRIPE_WEBHOOK_SECRET` | Stripe Dashboard → Developers → Webhooks → your endpoint → Signing secret |
| `STRIPE_PRICE_ID_REQUEST` | Stripe Dashboard → Products → your metered product → Price ID |

---

## Local Development Setup

Before testing the checkout flow locally, configure a development environment that talks to Stripe's **test mode** instead of live. This lets you use test credit cards without real charges.

### 1. Get test-mode API keys

1. In the [Stripe Dashboard](https://dashboard.stripe.com), toggle **"Test mode"** in the top-right corner
2. Go to [Developers → API keys](https://dashboard.stripe.com/test/apikeys)
3. Copy the **test Secret key** (starts with `sk_test_`) and **test Publishable key** (starts with `pk_test_`)

### 2. Create a test-mode product and price

Stripe test mode has its own separate products. You need to create the same metered price in test mode.

1. While still in test mode, go to [Products](https://dashboard.stripe.com/test/products)
2. Create the same "API Requests" product with usage-based pricing ($0.005/request, monthly)
3. Copy the test **Price ID** (starts with `price_`) — this will be different from the live one

### 3. Set up webhook forwarding

For local development, Stripe needs a way to reach your local server. Two options:

**Option A: Stripe CLI (recommended)**

1. Install the [Stripe CLI](https://docs.stripe.com/stripe-cli)
2. Log in: `stripe login`
3. Forward events to your local server:
   ```bash
   stripe listen --forward-to localhost:8080/api/webhooks/stripe
   ```
4. The CLI prints a webhook signing secret (`whsec_...`) — use this as your `STRIPE_WEBHOOK_SECRET`

**Option B: ngrok**

1. Run `ngrok http 8080`
2. In the Stripe test-mode dashboard, go to [Developers → Webhooks](https://dashboard.stripe.com/test/webhooks)
3. Add a new endpoint: `https://your-subdomain.ngrok-free.app/api/webhooks/stripe`
4. Add the same 5 events listed in [Production Setup → Step 3](#3-set-up-the-webhook-endpoint)
5. Copy the signing secret from the endpoint detail page

### 4. Configure your local `.env`

Copy from `.env.example` if you haven't already, then set:

```bash
MODE=development
BILLING_ENABLED=true
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...          # from Stripe CLI or test webhook endpoint
STRIPE_PRICE_ID_REQUEST=price_...        # test-mode price ID from step 2
BASE_URL=http://localhost:8080           # or your ngrok URL if using ngrok
```

For the frontend, set in your shell or `.env`:

```bash
VITE_STRIPE_PUBLISHABLE_KEY_TEST=pk_test_...
```

The frontend automatically uses `VITE_STRIPE_PUBLISHABLE_KEY_TEST` in development mode (`MODE=development`) and `VITE_STRIPE_PUBLISHABLE_KEY` in production builds.

> **Note:** In development mode (`MODE=development`), the server relaxes Stripe config validation — it warns instead of refusing to start if keys are missing.

### 5. Start the dev servers

```bash
# Terminal 1: backend
make run

# Terminal 2: frontend (Vite dev server)
cd frontend && npm run dev

# Terminal 3: webhook forwarding (if using Stripe CLI)
stripe listen --forward-to localhost:8080/api/webhooks/stripe
```

---

## Verifying the Checkout Flow

After setup (local or production), verify the end-to-end flow:

1. Open the app and click the upgrade/subscribe button
2. Confirm you're redirected to Stripe Checkout
3. Complete a test payment using one of [Stripe's test cards](https://docs.stripe.com/testing#cards):

   | Card number | Behavior |
   |---|---|
   | `4242 4242 4242 4242` | Success — any future expiry, any CVC, any ZIP |
   | `4000 0025 0000 3155` | Requires 3D Secure authentication |
   | `4000 0000 0000 0002` | Card is declined |
   | `4000 0000 0000 0341` | Succeeds initially, fails on the *next* invoice |

4. Confirm you're redirected back to the app after payment
5. Check the database — the user's subscription should be upgraded from `free` to `pay_as_you_go`:

   ```sql
   SELECT plan_id, status, stripe_customer_id, stripe_subscription_id
   FROM subscriptions WHERE user_id = '<your-user-id>';
   ```

## Verifying Webhook Events

1. In the [Stripe Dashboard → Developers → Webhooks → your endpoint](https://dashboard.stripe.com/test/webhooks), check that events show a **200** response status
2. Verify `checkout.session.completed` was received after your test checkout
3. If using Stripe CLI, check the terminal output — it shows each event forwarded and the response code
4. Test failure scenarios: use card `4000 0000 0000 0341` to trigger a payment failure on the *next* invoice, then confirm `invoice.payment_failed` is received and the subscription status updates to `past_due`

## Environment Variable Reference

| Variable | Type | Required when billing enabled | Description |
|---|---|---|---|
| `BILLING_ENABLED` | Runtime | — | `true` to enable billing (default: `false`) |
| `STRIPE_SECRET_KEY` | Runtime | Yes (server won't start) | Stripe API secret key |
| `STRIPE_PUBLISHABLE_KEY` | Runtime | No (used for API responses) | Stripe publishable key |
| `STRIPE_WEBHOOK_SECRET` | Runtime | Yes (server won't start) | Webhook signature verification |
| `STRIPE_PRICE_ID_REQUEST` | Runtime | Warns if missing | Metered Price ID for per-request billing |
| `VITE_STRIPE_PUBLISHABLE_KEY` | Build-time | Yes (checkout won't work) | Stripe publishable key for frontend |
| `BASE_URL` | Runtime | Yes (server won't start) | Public URL (used for Stripe redirect URLs) |

## Pricing Model

| Plan | Included requests | Overage cost | Audit retention |
|---|---|---|---|
| `free` | 250/month | N/A (blocked) | 7 days |
| `pay_as_you_go` | Unlimited | $0.005/request | 90 days |

> **Source of truth:** Plan limits are defined in `config/plans.json`. The values above reflect that config at time of writing.

SMS charges are separate and region-based: $0.01/message (US/CA), $0.04 (UK/EU), $0.05 (international).

## Key Rotation

Rotate Stripe secrets every 90 days:

- **`STRIPE_SECRET_KEY`** — roll the key in the [Stripe dashboard](https://dashboard.stripe.com/apikeys) (supports an overlap period), deploy the new key, then revoke the old one
- **`STRIPE_WEBHOOK_SECRET`** — create a new webhook endpoint in Stripe, update the secret in your deployment, then delete the old endpoint

See the [production deployment guide](deployment-production.md#rotating-secrets) for the full rotation schedule.
