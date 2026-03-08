# OAuth Setup Guide

Permission Slip uses OAuth 2.0 to connect with Google, Microsoft, Meta (Facebook/Instagram), Linear, and X (Twitter) services. This guide covers how to configure OAuth for both hosted and self-hosted deployments.

## Overview

Permission Slip supports two modes for OAuth provider credentials:

1. **Platform-level (pre-configured)**: The server has Google/Microsoft client credentials set via environment variables. Users can connect their accounts immediately.
2. **BYOA (Bring Your Own App)**: Users provide their own OAuth client credentials through the Settings UI. Required for self-hosted deployments or custom providers.

## Environment Variables

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

### OAuth Infrastructure

| Variable | Description | Default |
|---|---|---|
| `OAUTH_REDIRECT_BASE_URL` | Base URL for OAuth callbacks (e.g., `https://app.example.com/api`) | Falls back to `BASE_URL` |
| `OAUTH_STATE_SECRET` | HMAC secret for signing OAuth CSRF state tokens | Falls back to `SUPABASE_JWT_SECRET` |
| `OAUTH_REFRESH_INTERVAL` | Interval for background token refresh job | `10m` |

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
- Google: `https://your-domain.com/api/v1/oauth/google/callback`
- Microsoft: `https://your-domain.com/api/v1/oauth/microsoft/callback`
- Meta: `https://your-domain.com/api/v1/oauth/meta/callback`
- Linear: `https://your-domain.com/api/v1/oauth/linear/callback`
- X: `https://your-domain.com/api/v1/oauth/x/callback`

If using `OAUTH_REDIRECT_BASE_URL`, the callback URL is `{OAUTH_REDIRECT_BASE_URL}/v1/oauth/{provider}/callback`.

### Token refresh failures

Check the server logs for refresh errors. Common causes:
- Refresh token expired (Google refresh tokens expire after 6 months of inactivity)
- OAuth app credentials rotated without updating the server
- Provider API downtime
