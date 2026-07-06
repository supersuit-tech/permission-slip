import { formatAbsoluteTime } from "@/lib/utils";
import {
  hasPartialEmailApprovalDetails,
  parseEmailApprovalDetails,
} from "@/lib/emailApprovalDetails";

interface EmailApprovalDetailsRowProps {
  actionType: string;
  resourceDetails?: Record<string, unknown> | null;
}

function DetailField({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[5.5rem_1fr] gap-x-3 gap-y-1 text-sm">
      <dt className="text-muted-foreground font-medium">{label}</dt>
      <dd className="min-w-0 break-words text-foreground">{value}</dd>
    </div>
  );
}

export function EmailApprovalDetailsRow({
  actionType,
  resourceDetails,
}: EmailApprovalDetailsRowProps) {
  const details = parseEmailApprovalDetails(actionType, resourceDetails);
  if (!hasPartialEmailApprovalDetails(details)) {
    return null;
  }

  const dateLabel =
    details?.date != null
      ? (() => {
          const formatted = formatAbsoluteTime(details.date);
          return formatted.length > 0 ? formatted : details.date;
        })()
      : null;

  return (
    <div
      className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm"
      data-testid="email-approval-details"
    >
      <dl className="space-y-2">
        {details?.from != null && <DetailField label="From" value={details.from} />}
        {details?.subject != null && <DetailField label="Subject" value={details.subject} />}
        {dateLabel != null && <DetailField label="Date" value={dateLabel} />}
      </dl>
    </div>
  );
}
