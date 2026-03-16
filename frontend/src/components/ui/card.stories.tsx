import type { Meta, StoryObj } from "@storybook/react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "./card";
import { Button } from "./button";
import { Badge } from "./badge";

const meta: Meta<typeof Card> = {
  title: "UI/Card",
  component: Card,
  parameters: { layout: "padded" },
};
export default meta;
type Story = StoryObj<typeof Card>;

export const Simple: Story = {
  render: () => (
    <Card className="w-[380px]">
      <CardHeader>
        <CardTitle>Card Title</CardTitle>
        <CardDescription>Card description goes here.</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-sm">Card content area.</p>
      </CardContent>
    </Card>
  ),
};

export const ApprovalCard: Story = {
  render: () => (
    <Card className="w-[380px]">
      <CardHeader>
        <CardTitle>Delete Calendar Event</CardTitle>
        <CardDescription>google_calendar.delete_event</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-2">
          <Badge variant="warning-soft">Medium Risk</Badge>
          <Badge variant="info-soft">Pending</Badge>
        </div>
        <p className="text-muted-foreground mt-3 text-sm">
          Agent wants to delete event{" "}
          <code className="bg-muted rounded px-1 py-0.5 text-xs">
            fifrnnp6iai8qi5klOhl9cspms
          </code>{" "}
          from the primary calendar.
        </p>
      </CardContent>
      <CardFooter className="gap-2 justify-end">
        <Button variant="outline" size="sm">
          Deny
        </Button>
        <Button size="sm">Approve</Button>
      </CardFooter>
    </Card>
  ),
};

export const WithFooter: Story = {
  render: () => (
    <Card className="w-[380px]">
      <CardHeader>
        <CardTitle>Settings</CardTitle>
        <CardDescription>Manage your notification preferences.</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-sm">
          You will receive push notifications for all pending approvals.
        </p>
      </CardContent>
      <CardFooter className="justify-end gap-2">
        <Button variant="outline">Cancel</Button>
        <Button>Save</Button>
      </CardFooter>
    </Card>
  ),
};
