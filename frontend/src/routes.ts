import { type ComponentType } from "react";
import { Dashboard } from "./pages/dashboard/Dashboard";
import { AgentConfigPage } from "./pages/agents/AgentConfigPage";
import { ConnectorConfigPage } from "./pages/agents/connectors/ConnectorConfigPage";
import { ActivityPage } from "./pages/activity/ActivityPage";
import { SettingsPage } from "./pages/settings/SettingsPage";
import { BillingPage } from "./pages/billing/BillingPage";

export interface RouteConfig {
  path: string;
  element: ComponentType;
}

/**
 * Authenticated app routes. Each entry maps a URL path to a page component.
 * Add new pages here — no need to touch App.tsx.
 *
 * Append new routes at the end to minimize merge conflicts.
 */
export const appRoutes: RouteConfig[] = [
  { path: "/", element: Dashboard },
  { path: "/agents/:agentId", element: AgentConfigPage },
  { path: "/agents/:agentId/connectors/:connectorId", element: ConnectorConfigPage },
  { path: "/activity", element: ActivityPage },
  { path: "/settings", element: SettingsPage },
  { path: "/billing", element: BillingPage },
];
