# Mobile Builds & Distribution

The mobile app uses [EAS (Expo Application Services)](https://docs.expo.dev/eas/) for building, distributing, and updating the React Native app.

## Prerequisites

1. An [Expo account](https://expo.dev/signup)
2. Node.js 20+
3. The EAS CLI (included as a dev dependency — `npx eas-cli` works out of the box)

## First-Time Setup

### 1. Link the Expo project

From the `mobile/` directory:

```bash
npx eas-cli init
```

This creates a project on Expo's servers and outputs an `EXPO_PROJECT_ID`. Add it (along with your Expo account name) to your `.env`:

```
EXPO_PROJECT_ID=your-project-id-here
EXPO_OWNER=your-expo-username
```

`app.config.ts` reads these env vars automatically — no manual edits needed.

### 2. Configure app signing

**iOS:**
- EAS Build manages provisioning profiles and certificates automatically on first build
- You'll be prompted to log in to your Apple Developer account
- Set these in `eas.json` under `submit.production.ios`:
  - `appleId`: Your Apple ID email
  - `ascAppId`: App Store Connect app ID (numeric)
  - `appleTeamId`: Your Apple Developer Team ID

**Android:**
- EAS generates a keystore automatically on first build
- For Google Play submission, create a service account key:
  1. Go to Google Play Console → Setup → API access
  2. Create a service account with "Release manager" permissions
  3. Download the JSON key file
  4. Store it **outside the repo** (e.g., `~/.config/eas/`) — the `.gitignore` blocks `service-account*.json` as a safety net, but credentials should never be in the repo tree
  5. Set the absolute path in `eas.json` under `submit.production.android.serviceAccountKeyPath`

### 3. Set up CI (optional)

For automated builds in GitHub Actions, add an `EXPO_TOKEN` secret:

1. Generate a token at https://expo.dev/accounts/[account]/settings/access-tokens
2. Add it as a repository secret named `EXPO_TOKEN` in GitHub

## Build Profiles

| Profile | Distribution | Use Case |
|---------|-------------|----------|
| `development` | Internal (simulator) | Local development with dev client |
| `preview` | Internal (device) | Testing on physical devices before release |
| `production` | App Store / Google Play | Production release builds |

## Common Commands

```bash
# Build for development (both platforms)
make mobile-build-dev

# Build for a single platform
make mobile-build-dev-ios
make mobile-build-dev-android

# Build preview for testers
make mobile-build-preview
make mobile-build-preview-ios
make mobile-build-preview-android

# Build for production release
make mobile-build-prod

# Submit to app stores
make mobile-submit

# Push an OTA update (no app store review needed)
make mobile-update
```

Or from the `mobile/` directory using npm scripts:

```bash
npm run eas-build:dev
npm run eas-build:preview
npm run eas-build:prod
npm run eas-submit
npm run eas-update
```

## OTA Updates

Over-the-air updates via `expo-updates` allow pushing JS bundle updates without going through app store review. Updates are routed by channel:

- `development` channel → development builds
- `preview` channel → preview builds
- `production` channel → production builds

To push an update:

```bash
make mobile-update
```

The `runtimeVersion` policy is set to `appVersion`, meaning OTA updates are only delivered to builds with a matching app version. If you change native code (new native modules, SDK upgrade), you must publish a new build — OTA updates only work for JS/asset changes.

### OTA Code Signing (recommended for production)

Expo supports code signing for OTA updates to prevent tampering. Without it, a compromised Expo account could push malicious updates to production devices. To enable:

1. Generate a key pair: `npx expo-updates codesigning:generate --key-output-directory keys --certificate-validity-duration-years 10`
2. Configure in `eas.json` under `build.production`:
   ```json
   "updates": {
     "codeSigningCertificate": "keys/certificate.pem",
     "codeSigningMetadata": {
       "keyid": "main",
       "alg": "rsa-v1_5-sha256"
     }
   }
   ```
3. Publish signed updates: `npx eas-cli update --channel production --code-signing-certificate keys/certificate.pem --code-signing-private-key keys/private-key.pem`

See [Expo code signing docs](https://docs.expo.dev/eas-update/code-signing/) for full details.

## Environment Variables

Each build profile can have its own environment variables set in `eas.json` under `build.<profile>.env`. At minimum, set:

- `EXPO_PUBLIC_SUPABASE_URL` — Supabase project URL
- `EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY` — Supabase anon key

For preview/production, update these values in `eas.json` to point at the correct Supabase project.
