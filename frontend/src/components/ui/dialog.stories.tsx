import type { Meta, StoryObj } from "@storybook/react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./dialog";
import { Button } from "./button";
import { Badge } from "./badge";

const meta: Meta<typeof Dialog> = {
  title: "UI/Dialog",
  component: Dialog,
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof Dialog>;

export const Default: Story = {
  render: () => (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline">Open Dialog</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Are you sure?</DialogTitle>
          <DialogDescription>
            This action cannot be undone. This will permanently delete the event
            from your calendar.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline">Cancel</Button>
          <Button variant="destructive">Delete</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  ),
};

export const ApprovalDialog: Story = {
  render: () => (
    <Dialog defaultOpen>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Calendar Event</DialogTitle>
          <DialogDescription>
            Chiedobot wants to perform an action
          </DialogDescription>
        </DialogHeader>
        <div className="flex items-center gap-2">
          <Badge variant="warning-soft">Pending</Badge>
          <span className="text-muted-foreground text-sm">9:30 AM</span>
        </div>
        <div className="bg-muted rounded-lg p-3">
          <p className="text-sm">
            Delete event{" "}
            <code className="bg-background rounded px-1.5 py-0.5 text-xs font-medium">
              fifrnnp6iai8qi5klOhl9cspms
            </code>
          </p>
        </div>
        <DialogFooter>
          <Button variant="outline" className="flex-1">
            Deny
          </Button>
          <Button className="flex-1">Approve</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  ),
};
