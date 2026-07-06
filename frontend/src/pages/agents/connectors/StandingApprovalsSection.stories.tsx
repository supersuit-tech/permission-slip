/**
 * Standing approvals on the connector config page — Storybook
 *
 * Mirrors `StandingApprovalsSection` table layout (name, action, constraints,
 * account, expiry, status) without API calls so Storybook stays offline.
 */
import type { Meta, StoryObj } from "@storybook/react";
import { Pencil, Plus, ShieldCheck, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";

function StandingApprovalsSectionLayoutMirror({
  variant,
}: {
  variant: "empty" | "populated";
}) {
  return (
    <Card>
      <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <ShieldCheck className="text-muted-foreground size-5" />
          <CardTitle>Standing Approvals</CardTitle>
        </div>
        {variant === "populated" && (
          <div className="flex flex-wrap items-center gap-2 self-start sm:self-center">
            <Button type="button" variant="outline" size="sm" disabled>
              Recommended Templates
            </Button>
            <Button type="button" size="sm" disabled>
              <Plus className="size-4" />
              Add Standing Approval
            </Button>
          </div>
        )}
      </CardHeader>
      <CardContent>
        {variant === "empty" ? (
          <div className="space-y-4 py-4 text-center">
            <Button size="lg" disabled>
              <Plus className="size-4" />
              Add Standing Approval
            </Button>
            <p className="text-muted-foreground mx-auto max-w-md text-sm">
              Every request from this agent will ask for your approval. Add a
              standing approval to pre-authorize trusted, repetitive actions.
            </p>
            <button
              type="button"
              disabled
              className="text-muted-foreground text-sm underline-offset-4"
            >
              Or start from a recommended template →
            </button>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg">
            <Table>
              <TableHeader>
                <TableRow className="border-none bg-primary hover:bg-primary">
                  <TableHead className="font-semibold text-primary-foreground">
                    Name
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Action
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Constraints
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Account
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Expires
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Status
                  </TableHead>
                  <TableHead className="w-[100px] font-semibold text-primary-foreground" />
                </TableRow>
              </TableHeader>
              <TableBody className="[&>tr:nth-child(even)]:bg-muted">
                <TableRow>
                  <TableCell>
                    <div>
                      <p className="font-medium">Create issues in webapp</p>
                      <p className="text-muted-foreground text-sm">
                        Auto-approve new bug reports
                      </p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div>
                      <p className="text-sm">Create Issue</p>
                      <p className="text-muted-foreground font-mono text-xs">
                        github.create_issue
                      </p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-1 text-xs">
                      <span className="text-muted-foreground font-mono">
                        repo:
                      </span>
                      <Badge variant="secondary" className="font-mono text-xs">
                        acme/webapp
                      </Badge>
                    </span>
                  </TableCell>
                  <TableCell className="text-sm">Acme GitHub</TableCell>
                  <TableCell className="text-sm">30d</TableCell>
                  <TableCell>
                    <Badge variant="success-soft" className="font-normal">
                      active
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button variant="ghost" size="sm" disabled aria-label="Edit">
                        <Pencil className="size-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive"
                        disabled
                        aria-label="Revoke"
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
                <TableRow className="bg-muted/40 text-muted-foreground [&_td]:opacity-90">
                  <TableCell>
                    <p className="font-medium">List open PRs</p>
                  </TableCell>
                  <TableCell>
                    <div>
                      <p className="text-sm">List Pull Requests</p>
                      <p className="font-mono text-xs">github.list_prs</p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs">Match all</span>
                  </TableCell>
                  <TableCell className="text-sm">Acme GitHub</TableCell>
                  <TableCell className="text-sm">Expired</TableCell>
                  <TableCell>
                    <Badge variant="outline" className="font-normal">
                      expired
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button variant="ghost" size="sm" disabled>
                        <Pencil className="size-4" />
                      </Button>
                      <Button variant="ghost" size="sm" disabled>
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

const meta = {
  title: "Pages/Agents/Connectors/Standing Approvals",
  component: StandingApprovalsSectionLayoutMirror,
  parameters: { layout: "padded" },
} satisfies Meta<typeof StandingApprovalsSectionLayoutMirror>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {
  args: { variant: "empty" },
};

export const WithRules: Story = {
  args: { variant: "populated" },
};
