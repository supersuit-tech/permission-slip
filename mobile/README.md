# Permission Slip — Mobile App

React Native mobile app built with [Expo](https://expo.dev). Lets users approve or deny permission requests via push notifications, biometric authentication, and deep linking.

## Prerequisites

- Node.js 20+
- [EAS CLI](https://docs.expo.dev/eas/) (included as a dev dependency — `npx eas-cli` works)
- An [Expo account](https://expo.dev/signup) (for device builds)
- Apple Developer account (for iOS device builds)
- The backend running locally, or `EXPO_PUBLIC_MOCK_AUTH=true` for UI-only development

## Quick Start

### 1. Install dependencies

```bash
cd mobile
npm install
```

### 2. Configure environment

```bash
cp .env.example .env
```

Fill in your values — see [Environment Variables](#environment-variables) below.

### 3. Start the dev server

```bash
npm start
```

> **Note:** Expo Go won't work because the app uses custom native modules (notifications, local authentication). You need a [development build](https://docs.expo.dev/develop/development-builds/introduction/).

## Development Builds

Build for iOS simulator:

```bash
npx eas-cli build --profile development --platform ios
```

Once the build completes, install it on your simulator and run `npm start` to connect.

For physical device builds, set `ios.simulator` to `false` in the `development` profile in `eas.json` and rebuild.

## Setting Up Your Own Expo Project

Contributors need their own Expo project to run device builds:

1. Create an Expo account at https://expo.dev/signup

2. Initialize your project:
   ```bash
   cd mobile
   npx eas-cli init
   ```

3. Add the project ID and your account details to `.env`:
   ```
   EXPO_PROJECT_ID=your-project-id
   EXPO_OWNER=your-expo-username
   APP_BUNDLE_ID=com.yourname.permissionslip
   ```

4. If using your own Supabase instance, update the env vars in the `eas.json` build profiles.

`app.config.ts` reads all of these from environment variables — no need to edit the config file directly.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `EXPO_PUBLIC_SUPABASE_URL` | Yes | — | Supabase project URL (`http://127.0.0.1:54321` for local) |
| `EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY` | Yes | — | Supabase anon/publishable key |
| `EXPO_PROJECT_ID` | For builds | — | EAS project ID (from `npx eas-cli init`) |
| `EXPO_OWNER` | For builds | — | Expo account username or org |
| `APP_BUNDLE_ID` | For builds | — | iOS bundle ID / Android package (must be unique to your account) |
| `EXPO_PUBLIC_API_BASE_URL` | No | `https://app.permissionslip.dev/api` | Backend API URL |
| `EXPO_PUBLIC_MOCK_AUTH` | No | `false` | `true` to test UI without a backend (`__DEV__` only) |

## Testing

```bash
npm test           # run all tests
npm test -- --ci   # CI mode (used by make mobile-test)
```

## Project Structure

```
src/
├── api/           # API client (generated from OpenAPI spec)
├── auth/          # Authentication context and screens
├── components/    # Shared UI components
├── hooks/         # Custom React hooks
├── lib/           # Supabase client, secure storage
├── navigation/    # React Navigation setup and deep linking
├── screens/       # Screen components (approvals, settings)
└── theme/         # Colors and design tokens
```

## Builds & Distribution

For full details on build profiles, OTA updates, code signing, and App Store submission, see [docs/mobile-builds.md](../docs/mobile-builds.md).
