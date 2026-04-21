/**
 * Connector credentials checklist — Storybook
 *
 * Mirrors `ConnectorInstancesSection` (checkboxes, default badge, Make default)
 * without API calls so Storybook stays offline.
 */
import type { Meta, StoryObj } from "@storybook/react";
import { Settings, Star } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";

function ConnectorInstancesSectionLayoutMirror() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Credentials for this agent</CardTitle>
        <CardDescription>
          Choose which stored credentials this agent may use for Slack. One
          credential is the default for approvals when no specific instance is
          selected.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex min-w-0 flex-1 items-start gap-3">
              <Checkbox id="story-oauth-1" checked className="mt-1" disabled />
              <div className="min-w-0 flex-1">
                <label htmlFor="story-oauth-1" className="cursor-pointer font-medium">
                  Slack OAuth — Acme Workspace
                </label>
                <p className="text-muted-foreground text-xs">Connected 3/1/2026</p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2 sm:justify-end">
              <Badge variant="secondary">Default</Badge>
              <Button type="button" variant="outline" size="sm" disabled>
                <Star className="size-3.5" />
                Make default
              </Button>
            </div>
          </div>

          <div className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex min-w-0 flex-1 items-start gap-3">
              <Checkbox id="story-oauth-2" checked className="mt-1" disabled />
              <div className="min-w-0 flex-1">
                <label htmlFor="story-oauth-2" className="cursor-pointer font-medium">
                  Slack OAuth — Sales Workspace
                </label>
                <p className="text-muted-foreground text-xs">Connected 3/2/2026</p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2 sm:justify-end">
              <Button type="button" variant="outline" size="sm">
                <Star className="size-3.5" />
                Make default
              </Button>
            </div>
          </div>
        </div>

        <div className="mt-4 flex justify-end">
          <Button type="button" variant="outline" size="sm">
            <Settings className="size-3" />
            Manage credentials
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

const meta: Meta<typeof ConnectorInstancesSectionLayoutMirror> = {
  title: "Agents/ConnectorInstancesSection",
  component: ConnectorInstancesSectionLayoutMirror,
  parameters: { layout: "padded" },
};
export default meta;

type Story = StoryObj<typeof ConnectorInstancesSectionLayoutMirror>;

export const MultiCredentialChecklist: Story = {
  render: () => (
    <div className="max-w-xl">
      <ConnectorInstancesSectionLayoutMirror />
    </div>
  ),
};
