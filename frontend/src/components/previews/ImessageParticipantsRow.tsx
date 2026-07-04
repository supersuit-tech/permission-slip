import { cn } from "@/lib/utils";
import {
  formatImessageParticipants,
  parseImessageParticipants,
} from "@/lib/imessageParticipantsUtils";

export interface ImessageParticipantsRowProps {
  resourceDetails?: Record<string, unknown> | null;
  className?: string;
}

/**
 * Shows raw phone/email handles for a resolved iMessage chat in approval detail views.
 */
export function ImessageParticipantsRow({
  resourceDetails,
  className,
}: ImessageParticipantsRowProps) {
  const participants = parseImessageParticipants(resourceDetails);
  if (!participants) return null;

  return (
    <div
      className={cn(
        "overflow-hidden rounded-xl border bg-card p-4 shadow-sm text-sm",
        className,
      )}
      data-testid="imessage-participants-row"
    >
      <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
        Participants
      </p>
      <p className="mt-1 break-all">{formatImessageParticipants(participants)}</p>
    </div>
  );
}
