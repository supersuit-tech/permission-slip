import { type ComponentType } from "react";
import { Dashboard } from "./pages/dashboard/Dashboard";
import { AgentConfigPage } from "./pages/agents/AgentConfigPage";
import { ConnectorConfigPage } from "./pages/agents/connectors/ConnectorConfigPage";
import { ActivityPage } from "./pages/activity/ActivityPage";
import { SettingsPage } from "./pages/settings/SettingsPage";
import { BillingPage } from "./pages/billing/BillingPage";

export interface RouteConfig {
  path: string;
  /** The page component type (not a React element). Used as `<Component />` in App.tsx. */
  Component: ComponentType;
}

/**
 * Authenticated app routes. Each entry maps a URL path to a page component.
 * Add new pages here — no need to touch App.tsx.
 *
 * Append new routes at the end to minimize merge conflicts.
 */
export const appRoutes: RouteConfig[] = [
  { path: "/", Component: Dashboard },
  { path: "/agents/:agentId", Component: AgentConfigPage },
  { path: "/agents/:agentId/connectors/:connectorId", Component: ConnectorConfigPage },
  { path: "/activity", Component: ActivityPage },
  { path: "/settings", Component: SettingsPage },
  { path: "/billing", Component: BillingPage },
];
