# OAuth Setup Guide

Permission Slip uses OAuth 2.0 to connect with GitHub, Google, HubSpot, Linear, Meta (Facebook/Instagram), Microsoft, Notion, Square, Stripe, and X (Twitter) services. This guide covers how to configure OAuth for both hosted and self-hosted deployments.

## Overview

Permission Slip supports two modes for OAuth provider credentials:

1. **Platform-level (pre-configured)**: The server has Google/Microsoft client credentials set via environment variables. Users can connect their accounts immediately.
2. **BYOA (Bring Your Own App)**: Users provide their own OAuth client credentials through the Settings UI. Required for self-hosted deployments or custom providers.

## Environment Variables

### GitHub OAuth

| Variable | Description |
|---|---|
| `GITHUB_CLIENT_ID` | OAuth App Client ID from GitHub Developer Settings |
| `GITHUB_CLIENT_SECRET` | OAuth App Client Secret from GitHub Developer Settings |

### Google OAuth

| Variable | Description |
|---|---|
| `GOOGLE_CLIENT_ID` | OAuth 2.0 Client ID from Google Cloud Console |
| `GOOGLE_CLIENT_SECRET` | OAuth 2.0 Client Secret from Google Cloud Console |

### Microsoft OAuth

| Variable | Description |
|---|---|
| `MICROSOFT_CLIENT_ID` | Application (client) ID from Azure Portal |
| `MICROSOFT_CLIENT_SECRET` | Client secret value from Azure Portal |

### Meta OAuth

| Variable | Description |
|---|---|
| `META_CLIENT_ID` | App ID from Meta Developer Dashboard |
| `META_CLIENT_SECRET` | App Secret from Meta Developer Dashboard |

### Linear OAuth

| Variable | Description |
|---|---|
| `LINEAR_CLIENT_ID` | OAuth Application ID from Linear Settings |
| `LINEAR_CLIENT_SECRET` | OAuth Client Secret from Linear Settings |

### Notion OAuth

| Variable | Description |
|---|---|
| `NOTION_CLIENT_ID` | OAuth Client ID from [Notion Integrations](https://www.notion.so/my-integrations) |
| `NOTION_CLIENT_SECRET` | OAuth Client Secret from Notion Integrations |

### Square OAuth

| Variable | Description |
|---|---|
| `SQUARE_CLIENT_ID` | Production Application ID from Square Developer Dashboard |
| `SQUARE_CLIENT_SECRET` | Production Application Secret from Square Developer Dashboard |

### Stripe OAuth

| Variable | Description |
|---|---|
| `STRIPE_CLIENT_ID` | OAuth client ID from Stripe Connect settings (starts with `ca_`) |
| `STRIPE_CLIENT_SECRET` | Stripe secret key used as the client secret for the OAuth token exchange |

### OAuth Infrastructure

| Variable | Description | Default |
|---|---|---|
| `OAUTH_REDIRECT_BASE_URL` | Base URL for OAuth callbacks (e.g., `https://app.example.com/api`) | Falls back to `BASE_URL` |
| `OAUTH_STATE_SECRET` | HMAC secret for signing OAuth CSRF state tokens | Falls back to `SUPABASE_JWT_SECRET` |
| `OAUTH_REFRESH_INTERVAL` | Interval for background token refresh job | `10m` |

## GitHub OAuth Setup

### 1. Create a GitHub OAuth App

1. Go to [GitHub Developer Settings > OAuth Apps](https://github.com/settings/developers)
2. Click **New OAuth App**
3. Fill in the required fields:
   - Application name: Your deployment name (e.g., "Permission Slip")
   - Homepage URL: Your deployment URL
   - Authorization callback URL:
     ```
     https://your-domain.com/api/v1/oauth/github/callback
     ```

### 2. Configure Environment

```bash
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
```

### Scopes

The GitHub connector requests the `repo` scope, which provides full access to private and public repositories. This enables all GitHub connector actions (create issues, merge PRs, create releases, manage branches, etc.).

> **Note:** The GitHub connector supports both OAuth and Personal Access Tokens (PATs). OAuth is recommended for end users; PATs can be used as an alternative by configuring a `github_pat` credential with an `api_key` auth type.

## Google OAuth Setup

### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the APIs your connectors need:
   - **Gmail API** (for email actions)
   - **Google Calendar API** (for calendar actions)

### 2. Configure OAuth Consent Screen

1. Navigate to **APIs & Services > OAuth consent screen**
2. Select **External** user type (or **Internal** for Google Workspace)
3. Fill in the required fields:
   - App name: Your deployment name (e.g., "Permission Slip")
   - User support email: Your support email
   - Authorized domains: Your deployment domain
4. Add the following scopes:
   - `openid`
   - `https://www.googleapis.com/auth/userinfo.email`
   - `https://www.googleapis.com/auth/gmail.send`
   - `https://www.googleapis.com/auth/gmail.readonly`
   - `https://www.googleapis.com/auth/calendar.events`

### 3. Create OAuth Credentials

1. Navigate to **APIs & Services > Credentials**
2. Click **Create Credentials > OAuth 2.0 Client ID**
3. Application type: **Web application**
4. Add authorized redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/google/callback
   ```
5. Copy the **Client ID** and **Client Secret**

### 4. Configure Environment

```bash
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
```

## Microsoft OAuth Setup

### 1. Register an Application in Azure

1. Go to [Azure Portal > App Registrations](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade)
2. Click **New registration**
3. Fill in the required fields:
   - Name: Your deployment name (e.g., "Permission Slip")
   - Supported account types: **Accounts in any organizational directory and personal Microsoft accounts**
4. Add a redirect URI:
   - Platform: **Web**
   - URI: `https://your-domain.com/api/v1/oauth/microsoft/callback`

### 2. Configure API Permissions

1. Navigate to **API permissions**
2. Add the following Microsoft Graph **delegated** permissions:
   - `openid`
   - `offline_access`
   - `User.Read`
   - `Mail.Send`
   - `Mail.Read`
   - `Calendars.ReadWrite`

### 3. Create Client Secret

1. Navigate to **Certificates & secrets**
2. Click **New client secret**
3. Set an expiration (recommended: 24 months)
4. Copy the secret **value** (not the ID)

### 4. Configure Environment

```bash
MICROSOFT_CLIENT_ID=your-application-client-id
MICROSOFT_CLIENT_SECRET=your-client-secret-value
```

## Meta (Facebook/Instagram) OAuth Setup

### 1. Create a Meta App

1. Go to [Meta for Developers](https://developers.facebook.com/apps/)
2. Click **Create App** and choose **Business** type
3. Fill in the app name and contact email
4. In the app dashboard, add the **Facebook Login for Business** product

### 2. Configure Facebook Login Settings

1. Navigate to **Facebook Login > Settings**
2. Add the redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/meta/callback
   ```
3. Enable **Client OAuth Login** and **Web OAuth Login**

### 3. Request Permissions

The Meta connector requires these permissions (scopes):
- `pages_manage_posts` — create, edit, and delete Page posts
- `pages_read_engagement` — read Page post engagement (likes, comments, shares)
- `pages_read_user_content` — read user-generated content on Pages
- `instagram_basic` — read Instagram account info
- `instagram_content_publish` — publish photos to Instagram
- `instagram_manage_insights` — read Instagram account insights

For production use, submit your app for [App Review](https://developers.facebook.com/docs/app-review) to request these permissions. During development, permissions work for users with roles on the app (admin, developer, tester).

### 4. Configure Environment

```bash
META_CLIENT_ID=your-meta-app-id
META_CLIENT_SECRET=your-meta-app-secret
```

Find these under **App Settings > Basic** in the Meta Developer Dashboard.

## Linear OAuth Setup

### 1. Create a Linear OAuth Application

1. Go to [Linear Settings > API > OAuth Applications](https://linear.app/settings/api/applications)
2. Click **New OAuth Application**
3. Fill in the required fields:
   - Application name: Your deployment name (e.g., "Permission Slip")
   - Developer URL: Your website URL
   - Redirect callback URLs: `https://your-domain.com/api/v1/oauth/linear/callback`

### 2. Configure Scopes

The Linear connector requires these scopes:
- `read` — read issues, projects, teams, and other workspace data
- `write` — create and update issues, comments, and projects

### 3. Copy Credentials

From the OAuth application settings page, copy:
- **Client ID** (Application ID)
- **Client Secret**

### 4. Configure Environment

```bash
LINEAR_CLIENT_ID=your-linear-client-id
LINEAR_CLIENT_SECRET=your-linear-client-secret
```

> **Note:** Linear also supports API key authentication as an alternative. Users who prefer not to use OAuth can generate a personal API key at [Linear Settings > API > Personal API Keys](https://linear.app/docs/graphql/working-with-the-graphql-api#personal-api-keys) and configure it in the connector's credentials section.

## Notion OAuth Setup

Notion supports both OAuth and internal integration tokens (API keys). OAuth is recommended for end users; API keys are useful for server-to-server integrations or when OAuth is not available.

### 1. Create a Notion Integration

1. Go to [My Integrations](https://www.notion.so/my-integrations)
2. Click **New integration**
3. Fill in the required fields:
   - Name: Your deployment name (e.g., "Permission Slip")
   - Associated workspace: Select the workspace to connect
4. Under **Capabilities**, select the permissions your connectors need (read content, update content, insert content)
5. Under **Distribution**, enable **Public integration** to use OAuth

### 2. Configure OAuth Settings

1. In the integration settings, go to the **OAuth Domain & URIs** section
2. Add the redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/notion/callback
   ```
3. Copy the **OAuth client ID** and **OAuth client secret**

### 3. Configure Environment

```bash
NOTION_CLIENT_ID=your-notion-client-id
NOTION_CLIENT_SECRET=your-notion-client-secret
```

### Alternative: API Key (Internal Integration Token)

If you don't need OAuth, you can use an internal integration token instead:

1. Create a **private** integration (not public) at [My Integrations](https://www.notion.so/my-integrations)
2. Copy the **Internal Integration Secret** (starts with `ntn_`)
3. In Permission Slip, add it as an API key credential in the connector settings
4. Share your Notion pages/databases with the integration

The Notion connector accepts either auth method. When both are configured, OAuth is preferred.

## Square OAuth Setup

### 1. Create a Square Developer Application

1. Go to [Square Developer Dashboard](https://developer.squareup.com/apps)
2. Click **+** to create a new application
3. Fill in the application name (e.g., "Permission Slip")

### 2. Configure OAuth Settings

1. Navigate to **OAuth** in the left sidebar
2. Add the redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/square/callback
   ```
3. Select the required OAuth permissions (scopes):
   - `ORDERS_READ`, `ORDERS_WRITE`
   - `PAYMENTS_READ`, `PAYMENTS_WRITE`
   - `ITEMS_READ`, `ITEMS_WRITE`
   - `CUSTOMERS_READ`, `CUSTOMERS_WRITE`
   - `APPOINTMENTS_READ`, `APPOINTMENTS_WRITE`
   - `INVOICES_READ`, `INVOICES_WRITE`
   - `INVENTORY_READ`, `INVENTORY_WRITE`

### 3. Get Credentials

1. Navigate to **Credentials** in the left sidebar
2. Copy the **Production Application ID** and **Production Application Secret**

### 4. Configure Environment

```bash
SQUARE_CLIENT_ID=your-production-application-id
SQUARE_CLIENT_SECRET=your-production-application-secret
```

> **Note:** The Square connector supports both OAuth and API key authentication. OAuth is recommended for production use. API keys can be generated from the Square Developer Dashboard under **Credentials > Production Access Token**.

## Stripe OAuth Setup

Stripe uses [Stripe Connect](https://docs.stripe.com/connect) OAuth to authorize access to Stripe accounts. This lets users connect their Stripe account without manually creating and pasting API keys.

### 1. Enable Stripe Connect

1. Go to [Stripe Dashboard > Settings > Connect](https://dashboard.stripe.com/settings/connect)
2. Enable Stripe Connect for your platform
3. Note your **Client ID** (starts with `ca_`) from the Connect settings page

### 2. Configure Redirect URI

1. In the Connect settings, add the redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/stripe/callback
   ```
2. For development, add `http://localhost:PORT/api/v1/oauth/stripe/callback`

### 3. Configure Environment

```bash
STRIPE_CLIENT_ID=ca_your-connect-client-id
STRIPE_CLIENT_SECRET=sk_live_your-secret-key
```

The `STRIPE_CLIENT_SECRET` is your platform's Stripe secret key — Stripe uses it as the client secret during the OAuth token exchange.

### 4. Scopes

The Stripe connector requests the `read_write` scope, which provides full API access to the connected account. Stripe Connect does not support more granular OAuth scopes.

### 5. How It Works

When a user connects their Stripe account:
1. They are redirected to `connect.stripe.com/oauth/authorize`
2. After authorizing, Stripe redirects back with an authorization code
3. The platform exchanges the code for the connected account's secret key
4. This key (stored as `access_token`) works like a regular Stripe API key but is scoped to the connected account

The Stripe connector also supports manual API key entry as a fallback for users who prefer not to use OAuth.

## X (Twitter) OAuth Setup

The X connector declares its own OAuth provider in its manifest, so no platform-level environment variables are needed. Users configure X OAuth through the BYOA flow.

### 1. Create an X Developer App

1. Go to the [X Developer Portal](https://developer.x.com/en/portal/dashboard)
2. Create a new project and app (or use an existing one)
3. In the app settings, enable **OAuth 2.0**
4. Set the **Type of App** to "Web App, Automated App or Bot"
5. Add the redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/x/callback
   ```

### 2. Configure OAuth Scopes

The X connector requires these scopes:
- `tweet.read` — read tweets and timelines
- `tweet.write` — post and delete tweets
- `users.read` — read user profiles
- `dm.read` — read direct messages
- `dm.write` — send direct messages
- `offline.access` — refresh tokens (required for long-lived access)

### 3. Configure in Permission Slip

1. In Permission Slip, go to **Settings > OAuth App Credentials**
2. Click **Configure** next to the X provider
3. Enter your **Client ID** and **Client Secret** from the X Developer Portal
4. Connect your X account via **Settings > Connected Accounts**

## Self-Hosted BYOA Setup

For self-hosted deployments without platform-level OAuth credentials, users configure their own OAuth apps through the Settings UI:

1. Follow the Google or Microsoft setup steps above to create an OAuth app
2. In Permission Slip, go to **Settings > OAuth App Credentials**
3. Click **Configure** next to the provider
4. Enter your Client ID and Client Secret
5. The credentials are encrypted and stored in the vault
6. You can now connect your account via **Settings > Connected Accounts**

## External Connector OAuth

External connectors can declare their own OAuth providers in `connector.json`:

```json
{
  "id": "salesforce",
  "name": "Salesforce",
  "oauth_providers": [
    {
      "id": "salesforce",
      "authorize_url": "https://login.salesforce.com/services/oauth2/authorize",
      "token_url": "https://login.salesforce.com/services/oauth2/token",
      "scopes": ["api", "refresh_token"]
    }
  ],
  "required_credentials": [
    {
      "service": "salesforce",
      "auth_type": "oauth2",
      "oauth_provider": "salesforce"
    }
  ]
}
```

The platform reads `oauth_providers` from the manifest and registers them in the provider registry. Users then supply their own OAuth app credentials via the BYOA configuration in Settings.

For more details on creating external connectors, see [Creating Connectors](creating-connectors.md).

## Troubleshooting

### "Provider not configured" error

The OAuth provider doesn't have client credentials. Either:
- Set the platform-level environment variables (`GOOGLE_CLIENT_ID`, etc.)
- Configure BYOA credentials in Settings > OAuth App Credentials

### "Needs Re-auth" status

The refresh token has expired or been revoked. Click **Re-authorize** in Settings > Connected Accounts to re-establish the connection.

### Redirect URI mismatch

Ensure the redirect URI in your OAuth app matches exactly:
- GitHub: `https://your-domain.com/api/v1/oauth/github/callback`
- Google: `https://your-domain.com/api/v1/oauth/google/callback`
- Microsoft: `https://your-domain.com/api/v1/oauth/microsoft/callback`
- Meta: `https://your-domain.com/api/v1/oauth/meta/callback`
- Linear: `https://your-domain.com/api/v1/oauth/linear/callback`
- Notion: `https://your-domain.com/api/v1/oauth/notion/callback`
- Square: `https://your-domain.com/api/v1/oauth/square/callback`
- Stripe: `https://your-domain.com/api/v1/oauth/stripe/callback`
- X: `https://your-domain.com/api/v1/oauth/x/callback`

If using `OAUTH_REDIRECT_BASE_URL`, the callback URL is `{OAUTH_REDIRECT_BASE_URL}/v1/oauth/{provider}/callback`.

### Token refresh failures

Check the server logs for refresh errors. Common causes:
- Refresh token expired (Google refresh tokens expire after 6 months of inactivity)
- OAuth app credentials rotated without updating the server
- Provider API downtime
