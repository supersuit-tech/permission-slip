import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
} from "@/components/ui/table";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { ActionRow } from "./ConnectorActionsSection";

interface ConnectorActionsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  actions: ConnectorAction[];
}

export function ConnectorActionsDialog({
  open,
  onOpenChange,
  actions,
}: ConnectorActionsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Available Actions</DialogTitle>
          <DialogDescription>
            All actions this connector supports. Configure how your agent uses
            them by adding action configurations.
          </DialogDescription>
        </DialogHeader>
        {actions.length === 0 ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            No actions defined for this connector.
          </p>
        ) : (
          <div className="overflow-hidden rounded-lg">
            <Table>
              <TableHeader>
                <TableRow className="border-none bg-primary hover:bg-primary">
                  <TableHead className="font-semibold text-primary-foreground">
                    Action
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Risk
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Parameters
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="[&>tr:nth-child(even)]:bg-muted">
                {actions.map((action) => (
                  <ActionRow key={action.action_type} action={action} />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
