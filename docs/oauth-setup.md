# OAuth Setup Guide

Permission Slip uses OAuth 2.0 to connect with Atlassian (Jira), Datadog, Dropbox, Figma, GitHub, Google, HubSpot, Linear, Meta (Facebook/Instagram), Microsoft, Notion, PagerDuty, Slack, Square, Stripe, and X (Twitter) services. This guide covers how to configure OAuth for both hosted and self-hosted deployments.

## Overview

OAuth provider credentials are configured via environment variables on the server. Set the appropriate client ID and secret for each provider you want to enable. Users can then connect their accounts through the Settings UI.

## Environment Variables

### Atlassian OAuth (Jira)

| Variable | Description |
|---|---|
| `ATLASSIAN_CLIENT_ID` | OAuth 2.0 Client ID from the Atlassian Developer Console |
| `ATLASSIAN_CLIENT_SECRET` | OAuth 2.0 Client Secret from the Atlassian Developer Console |

### Datadog OAuth

| Variable | Description |
|---|---|
| `DATADOG_CLIENT_ID` | Client ID from the Datadog OAuth application |
| `DATADOG_CLIENT_SECRET` | Client Secret from the Datadog OAuth application |

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

### Figma OAuth

| Variable | Description |
|---|---|
| `FIGMA_CLIENT_ID` | OAuth 2.0 Client ID from the Figma Developer settings |
| `FIGMA_CLIENT_SECRET` | OAuth 2.0 Client Secret from the Figma Developer settings |

### PagerDuty OAuth

| Variable | Description |
|---|---|
| `PAGERDUTY_CLIENT_ID` | Client ID from PagerDuty Developer Dashboard |
| `PAGERDUTY_CLIENT_SECRET` | Client Secret from PagerDuty Developer Dashboard |

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

### Slack OAuth

| Variable | Description |
|---|---|
| `SLACK_CLIENT_ID` | Client ID from [Your Apps](https://api.slack.com/apps) (App Credentials) |
| `SLACK_CLIENT_SECRET` | Client Secret from the same Slack app |

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

### DocuSign OAuth

| Variable | Description |
|---|---|
| `DOCUSIGN_CLIENT_ID` | Integration key from [DocuSign Apps & Keys](https://admindemo.docusign.com/apps-and-keys) |
| `DOCUSIGN_CLIENT_SECRET` | Secret key from DocuSign Apps & Keys |

### Dropbox OAuth

| Variable | Description |
|---|---|
| `DROPBOX_CLIENT_ID` | App key from the [Dropbox App Console](https://www.dropbox.com/developers/apps) |
| `DROPBOX_CLIENT_SECRET` | App secret from the Dropbox App Console |

The platform uses **PKCE** for Dropbox automatically: the server sends an S256 code challenge to Dropbox and keeps the code verifier only in the **signed OAuth state JWT** (the verifier is **sealed** in the JWT payload so it is not plaintext in browser history or logs). `GET /v1/oauth/providers` returns `"pkce": true` for providers that use PKCE.

### OAuth Infrastructure

| Variable | Description | Default / Notes |
|---|---|---|
| `OAUTH_REDIRECT_BASE_URL` | Base URL for OAuth callbacks (e.g., `https://app.example.com/api`) | Falls back to `BASE_URL` |
| `OAUTH_STATE_SECRET` | HMAC secret for signing OAuth CSRF state tokens (generate with `openssl rand -hex 32`). **Required** when using JWKS auth (no `SUPABASE_JWT_SECRET`). | Falls back to `SUPABASE_JWT_SECRET` if unset; no default for JWKS deployments. |
| `OAUTH_REFRESH_INTERVAL` | Interval for background token refresh job | `10m` |

## Atlassian (Jira) OAuth Setup

Atlassian uses OAuth 2.0 with the 3-legged OAuth (3LO) flow for Jira Cloud. This is the recommended authentication method for the Jira connector.

### 1. Create an OAuth 2.0 App

1. Go to the [Atlassian Developer Console](https://developer.atlassian.com/console/myapps/)
2. Click **Create** and select **OAuth 2.0 integration**
3. Fill in the app name (e.g., "Permission Slip")
4. Under **Authorization**, add the callback URL:
   ```
   https://your-domain.com/api/v1/oauth/atlassian/callback
   ```

### 2. Configure Scopes

Navigate to **Permissions** and add the following scopes under **Jira API**:

- `read:me` — user profile, required for cloud ID discovery
- `read:jira-work` — read issues, projects, and search results
- `write:jira-work` — create/update issues, add comments, transitions
- `offline_access` — obtain a refresh token for long-lived connections

### 3. Configure Environment

```bash
ATLASSIAN_CLIENT_ID=your-atlassian-client-id
ATLASSIAN_CLIENT_SECRET=your-atlassian-client-secret
```

Find these under **Settings** in your app's page in the Atlassian Developer Console.

### How It Works

Atlassian OAuth uses a cloud ID to route API requests. When a user connects via OAuth:
1. They are redirected to `auth.atlassian.com/authorize` with `audience=api.atlassian.com` and `prompt=consent`
2. After authorizing, the platform exchanges the code for access and refresh tokens
3. On first API call, the connector calls the [accessible-resources endpoint](https://developer.atlassian.com/cloud/jira/platform/oauth-2-3lo-apps/#2--get-the-cloud-id-for-your-site) to discover the user's Jira Cloud ID
4. The cloud ID is cached (1 hour TTL) and used to construct API URLs: `https://api.atlassian.com/ex/jira/{cloudId}/rest/api/3`

### Alternative: Basic Auth (API Token)

The Jira connector also supports basic authentication with an email and API token. Users can generate an API token from [Atlassian Account Settings](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/). OAuth is recommended for end users; API tokens are useful for service accounts or when OAuth is not available.

## Datadog OAuth Setup

Datadog OAuth is designed for marketplace-style integrations. The connector supports both OAuth (recommended) and API key + application key authentication.

> **Note:** Datadog's OAuth implementation uses split hostnames by design: authorization redirects go through `app.datadoghq.com` and token exchange happens on `api.datadoghq.com`. This is intentional per Datadog's OAuth2 API reference.

### 1. Create a Datadog OAuth Application

1. Log in to your Datadog account and go to [OAuth Apps](https://app.datadoghq.com/organization-settings/oauth-apps)
2. Click **+ New OAuth App**
3. Fill in the required fields:
   - Application name: Your deployment name (e.g., "Permission Slip")
   - Description: Brief description of the integration
4. Under **Redirect URIs**, add:
   ```
   https://your-domain.com/api/v1/oauth/datadog/callback
   ```
5. Copy the **Client ID** and **Client Secret**

### 2. Configure OAuth Scopes

The Datadog connector requests these scopes:
- `metrics_read` — query time series metrics data
- `incidents_read` — read incident details
- `incidents_write` — create and update incidents
- `monitors_read` — read monitor configuration and status
- `monitors_write` — mute/unmute monitors (required for snooze_alert)
- `workflows_run` — trigger Workflow automations (required for trigger_runbook)

### 3. Configure Environment

```bash
DATADOG_CLIENT_ID=your-datadog-client-id
DATADOG_CLIENT_SECRET=your-datadog-client-secret
```

### 4. Multi-Region Support

The Datadog connector supports all Datadog sites. If your organization uses a non-US1 site, users should set the optional `site` credential when configuring the connector:

| Site | Identifier |
|---|---|
| US1 (default) | `datadoghq.com` |
| US3 | `us3.datadoghq.com` |
| US5 | `us5.datadoghq.com` |
| EU | `datadoghq.eu` |
| AP1 | `ap1.datadoghq.com` |
| US1-Fed | `ddog-gov.com` |

The `site` credential is optional for OAuth users — if omitted, requests are routed to the US1 API.

### Alternative: API Key + Application Key Auth

Users who prefer not to use OAuth can connect Datadog with an API key and application key:
1. Go to [Datadog API Keys](https://app.datadoghq.com/organization-settings/api-keys) and create an API key
2. Go to [Datadog Application Keys](https://app.datadoghq.com/organization-settings/application-keys) and create an application key
3. In Permission Slip, add both as a `custom` credential in the connector settings

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

## Figma OAuth Setup

### 1. Create a Figma App

1. Go to the [Figma Developer Settings](https://www.figma.com/developers/apps)
2. Click **Create a new app**
3. Fill in:
   - App name: Your deployment name (e.g., "Permission Slip")
   - Website URL: Your deployment URL
4. Add the redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/figma/callback
   ```

### 2. Required Scopes

The Figma connector requests these scopes:
- `files:read` — read design files, components, and version history
- `file_comments:write` — post comments on design files

### 3. Configure Environment

```bash
FIGMA_CLIENT_ID=your-figma-client-id
FIGMA_CLIENT_SECRET=your-figma-client-secret
```

### Authentication Fallback

The Figma connector supports both OAuth (recommended) and personal access tokens. If a user has not connected via OAuth, the connector will fall back to any stored personal access token. Users can generate a PAT from [Figma's token management page](https://help.figma.com/hc/en-us/articles/8085703771159-Manage-personal-access-tokens).

## PagerDuty OAuth Setup

PagerDuty is the first built-in connector to support **both** OAuth 2.0 and API key authentication. Users can choose to connect via OAuth (recommended) or provide an API key manually.

### 1. Create a PagerDuty App

1. Go to the [PagerDuty Developer Dashboard](https://developer.pagerduty.com/)
2. Navigate to **My Apps** and click **Create New App**
3. Fill in the app details:
   - App Name: Your deployment name (e.g., "Permission Slip")
   - Description: Brief description of the integration
4. Under **OAuth 2.0**, enable it and add the redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/pagerduty/callback
   ```

### 2. Configure OAuth Scopes

The PagerDuty connector requires these scopes:
- `read` — read incidents, services, schedules, and on-call data
- `write` — create/update incidents, add notes, and manage escalations

### 3. Configure Environment

```bash
PAGERDUTY_CLIENT_ID=your-client-id
PAGERDUTY_CLIENT_SECRET=your-client-secret
```

Find these in your app's settings under the **OAuth 2.0** section.

### Alternative: API Key Auth

Users who prefer not to use OAuth can still connect PagerDuty with an API key. See [PagerDuty API Access Keys](https://support.pagerduty.com/main/docs/api-access-keys) for instructions on generating a key.

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

## Slack OAuth Setup

Slack uses [OAuth 2.0 with the V2 flow](https://api.slack.com/authentication/oauth-v2): bot scopes produce a bot user token (`xoxb-`), and user scopes produce a user token (`xoxp-`) used for APIs that only accept user tokens (for example `search.messages`).

### 1. Create a Slack app

1. Go to [Your Apps](https://api.slack.com/apps)
2. Click **Create New App** and choose **From scratch**
3. Name the app (e.g., "Permission Slip") and pick a development workspace

### 2. Configure redirect URL

1. Open **OAuth & Permissions**
2. Under **Redirect URLs**, add:
   ```
   https://your-domain.com/api/v1/oauth/slack/callback
   ```

### 3. Configure scopes

Under **Scopes**, add **Bot Token Scopes** and **User Token Scopes** to match what the connector requests:

**Bot Token Scopes**

- `channels:history`, `channels:join`, `channels:manage`, `channels:read`
- `chat:write`
- `files:write`
- `groups:history`, `groups:read`
- `im:history`, `im:read`, `im:write`
- `mpim:history`, `mpim:read`, `mpim:write`
- `reactions:write`
- `users:read`, `users:read.email`

**User Token Scopes** (required for search actions that only work with user tokens)

- `search:read.public`, `search:read.private`, `search:read.im`, `search:read.mpim`, `search:read.files`

### 4. Configure environment

```bash
SLACK_CLIENT_ID=your-slack-client-id
SLACK_CLIENT_SECRET=your-slack-client-secret
```

Find **Client ID** and **Client Secret** under **Basic Information > App Credentials**.

### 5. Install to workspace

When a user connects Slack from Permission Slip, they complete Slack’s OAuth consent and install the app to their workspace. No manual “reinstall” is required beyond that flow unless you change scopes—in that case, users may need to **Re-authorize** from the connector settings.

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

## DocuSign OAuth Setup

DocuSign OAuth uses the authorization code grant flow, which is the standard approach for user-facing integrations. After connecting, the platform automatically fetches the user's default account ID and API base URL from DocuSign's userinfo endpoint so the connector can make API calls without manual configuration.

> **Note:** DocuSign also supports RSA key / JWT grant (service-to-service) auth. Users who prefer that approach can configure it manually in the connector's credentials section instead.

### 1. Create a DocuSign Integration

1. Log in to [DocuSign Apps & Keys](https://admindemo.docusign.com/apps-and-keys) (use the [production apps page](https://app.docusign.com/integrations/apps-and-keys) for production deployments)
2. Click **Add App and Integration Key**
3. Give it a name (e.g., "Permission Slip")
4. Under **Authentication**, select **Authorization Code Grant**
5. Under **Redirect URIs**, add:
   ```
   https://your-domain.com/api/v1/oauth/docusign/callback
   ```
6. Copy the **Integration Key** (this is your Client ID) and generate a **Secret Key** (Client Secret)

### 2. Required Scopes

The DocuSign connector requests the `signature` scope, which covers all eSignature API operations (creating envelopes, sending for signature, checking status, downloading signed documents).

### 3. Configure Environment

```bash
DOCUSIGN_CLIENT_ID=your-integration-key
DOCUSIGN_CLIENT_SECRET=your-secret-key
```

### 4. Account Info Enrichment

Unlike most OAuth providers, DocuSign does not include the user's account ID or API base URL in the access token. After the token exchange, the platform automatically calls DocuSign's userinfo endpoint (`account.docusign.com/oauth/userinfo`) to fetch the user's default account and stores:

- `account_id` — required for all API calls
- `base_url` — the regional API endpoint (e.g., `https://na2.docusign.net/restapi/v2.1`)

This happens transparently during the OAuth callback. If the userinfo fetch fails, the connection attempt fails cleanly rather than storing a broken connection.

### 5. Production vs. Sandbox

DocuSign's demo environment (used by default) and production environment use the same OAuth endpoints (`account.docusign.com`). When users connect in production, their `base_url` is automatically set to the correct production API URL based on their account region. No additional configuration is needed.

## Dropbox OAuth Setup

Dropbox uses OAuth 2.0 with offline access (refresh tokens) so credentials stay valid long-term without re-authentication.

### 1. Create a Dropbox App

1. Go to the [Dropbox App Console](https://www.dropbox.com/developers/apps)
2. Click **Create app**
3. Choose **Scoped access** (not "App folder")
4. Choose **Full Dropbox** access type
5. Give it a name (e.g., "Permission Slip")
6. Under **Settings**, add the redirect URI:
   ```
   https://your-domain.com/api/v1/oauth/dropbox/callback
   ```
7. Copy the **App key** (Client ID) and **App secret** (Client Secret)

### 2. Configure Permissions

In the **Permissions** tab of your Dropbox app, enable these scopes:

- `files.content.write` — upload files, create folders, move/rename
- `files.content.read` — download files, search
- `sharing.write` — create shared links
- `file_requests.read` — read file request metadata

Click **Submit** after enabling scopes. Scope changes only take effect for new OAuth connections.

### 3. Configure Environment

```bash
DROPBOX_CLIENT_ID=your-app-key
DROPBOX_CLIENT_SECRET=your-app-secret
```

### 4. API Architecture

Dropbox uses two API hosts. The connector handles this automatically:

- **api.dropboxapi.com** — RPC/metadata endpoints (create folder, search, share, move)
- **content.dropboxapi.com** — content endpoints (upload, download) where file bytes are sent/received as raw `application/octet-stream` with metadata in the `Dropbox-API-Arg` header

## X (Twitter) OAuth Setup

The X connector declares its own OAuth provider in its manifest. Set the `X_CLIENT_ID` and `X_CLIENT_SECRET` environment variables to enable X OAuth.

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

1. In Permission Slip, go to an agent's configuration page and click the X connector card
2. On the connector's configuration page, enter your **Client ID** and **Client Secret** from the X Developer Portal
3. Connect your X account via the credential configuration section

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

The platform reads `oauth_providers` from the manifest and registers them in the provider registry. Operators must set the corresponding environment variables (e.g., `SALESFORCE_CLIENT_ID` and `SALESFORCE_CLIENT_SECRET`) to supply client credentials.

For more details on creating external connectors, see [Creating Connectors](creating-connectors.md).

## Troubleshooting

### "Provider not configured" error

The OAuth provider doesn't have client credentials. Set the appropriate environment variables (e.g., `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET`) and restart the server.

### "Needs Re-auth" status

The refresh token has expired or been revoked. Click **Re-authorize** on the connector's credential configuration page to re-establish the connection.

### Redirect URI mismatch

Ensure the redirect URI in your OAuth app matches exactly:
- Atlassian: `https://your-domain.com/api/v1/oauth/atlassian/callback`
- Datadog: `https://your-domain.com/api/v1/oauth/datadog/callback`
- DocuSign: `https://your-domain.com/api/v1/oauth/docusign/callback`
- Figma: `https://your-domain.com/api/v1/oauth/figma/callback`
- GitHub: `https://your-domain.com/api/v1/oauth/github/callback`
- Google: `https://your-domain.com/api/v1/oauth/google/callback`
- Linear: `https://your-domain.com/api/v1/oauth/linear/callback`
- Meta: `https://your-domain.com/api/v1/oauth/meta/callback`
- Microsoft: `https://your-domain.com/api/v1/oauth/microsoft/callback`
- Notion: `https://your-domain.com/api/v1/oauth/notion/callback`
- PagerDuty: `https://your-domain.com/api/v1/oauth/pagerduty/callback`
- Slack: `https://your-domain.com/api/v1/oauth/slack/callback`
- Square: `https://your-domain.com/api/v1/oauth/square/callback`
- Stripe: `https://your-domain.com/api/v1/oauth/stripe/callback`
- X: `https://your-domain.com/api/v1/oauth/x/callback`

If using `OAUTH_REDIRECT_BASE_URL`, the callback URL is `{OAUTH_REDIRECT_BASE_URL}/v1/oauth/{provider}/callback`.

### Slack API errors (`missing_scope`, `invalid_auth`, search without user token)

- **`missing_scope`** — The Slack app is missing bot or user scopes listed in [Slack OAuth Setup](#slack-oauth-setup). Add them under **OAuth & Permissions**, then have the user **Re-authorize** so Slack issues new tokens.
- **`invalid_auth` / `token_revoked`** — The workspace uninstalled the app or tokens were rotated. **Re-authorize** from the connector settings.
- **Search actions fail but messaging works** — Search uses a user token (`xoxp-`). Ensure **User Token Scopes** (especially `search:read.*`) are configured in the Slack app and the user completed consent for user scopes.

### "Failed to initiate OAuth flow" error

The server has no secret to sign OAuth CSRF state tokens. This happens when:
- Using JWKS-based auth (Supabase CLI v2+) where `SUPABASE_JWT_SECRET` is not set
- `OAUTH_STATE_SECRET` was not explicitly configured

**Fix:** Set the `OAUTH_STATE_SECRET` environment variable:
```bash
OAUTH_STATE_SECRET=$(openssl rand -hex 32)
```

The server validates this at startup:
- **Production** (non-dev): a `🚨`/`🛑` fatal-error line aborts startup — look for `config error` in your JSON logs with `"env_var":"OAUTH_STATE_SECRET"`.
- **Development** (`MODE=development`): a `⚠️` warning is logged but startup continues.

### Token refresh failures

Check the server logs for refresh errors. Common causes:
- Refresh token expired (Google refresh tokens expire after 6 months of inactivity)
- OAuth app credentials rotated without updating the server
- Provider API downtime
