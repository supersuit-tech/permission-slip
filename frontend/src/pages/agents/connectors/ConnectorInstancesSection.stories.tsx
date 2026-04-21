/**
 * Connector instances layout — Storybook
 *
 * Mirrors the structure of `ConnectorInstancesSection` (cards, default badge,
 * credential row) without API calls so Storybook stays offline.
 */
import type { Meta, StoryObj } from "@storybook/react";
import { Plus, Settings, Star, Trash2, Unplug } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";

const selectClassName =
  "border-input bg-background flex h-9 w-full rounded-md border px-3 py-1 text-sm";

function ConnectorInstancesSectionLayoutMirror() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Connector instances</CardTitle>
        <CardDescription>
          Connect via OAuth (recommended) or use an API key as an alternative.
          Each instance can use its own credential.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="rounded-lg border p-4">
            <div className="mb-3 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="font-medium">Engineering</p>
                  <Badge variant="secondary">Default</Badge>
                </div>
                <p className="text-muted-foreground mt-1 text-xs">
                  Added 3/1/2026, 12:00:00 PM
                </p>
              </div>
            </div>
            <div className="rounded-md border border-dashed p-3">
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Label className="text-sm font-medium">Credential</Label>
                  <Badge
                    variant="secondary"
                    className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                  >
                    Assigned
                  </Badge>
                </div>
                <p className="text-muted-foreground text-sm">
                  This instance uses the selected credential or OAuth connection.
                </p>
                <Button type="button" variant="ghost" size="sm" className="h-8 px-2">
                  <Unplug className="size-3.5" />
                  Disconnect
                </Button>
                <select className={selectClassName} defaultValue="oauth:oconn_1">
                  <option value="oauth:oconn_1">
                    Slack OAuth — Acme Workspace (connected 3/1/2026)
                  </option>
                </select>
              </div>
            </div>
          </div>

          <div className="rounded-lg border p-4">
            <div className="mb-3 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="font-medium">Sales</p>
                </div>
                <p className="text-muted-foreground mt-1 text-xs">
                  Added 3/2/2026, 12:00:00 PM
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button type="button" variant="outline" size="sm">
                  <Star className="size-3.5" />
                  Make default
                </Button>
                <Button type="button" variant="outline" size="sm">
                  <Trash2 className="size-3.5" />
                  Remove instance
                </Button>
              </div>
            </div>
            <div className="rounded-md border border-dashed p-3">
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Label className="text-sm font-medium">Credential</Label>
                  <Badge variant="destructive">Not set</Badge>
                </div>
                <p className="text-muted-foreground text-sm">
                  Select a credential so this instance can run actions.
                </p>
                <select className={selectClassName} defaultValue="">
                  <option value="">Select a credential…</option>
                  <option value="oauth:oconn_2">
                    Slack OAuth — Sales Workspace (connected 3/2/2026)
                  </option>
                </select>
              </div>
            </div>
          </div>

          <Button type="button" variant="outline" size="sm" className="w-full sm:w-auto">
            <Plus className="size-4" />
            Add another
          </Button>
        </div>

        <div className="mt-4 flex justify-end">
          <Button variant="outline" size="sm">
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

export const MultiInstanceLayout: Story = {
  render: () => (
    <div className="max-w-xl">
      <ConnectorInstancesSectionLayoutMirror />
    </div>
  ),
};
