import { AlertTriangle } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import type { RiskLevel } from "./RiskBadge";
import { riskCardClass } from "./RiskBadge";
import {
  effectiveRiskLevel,
  unrestrictedApprovalWarning,
} from "@/lib/unrestrictedApprovalCopy";

interface UnrestrictedApprovalConfirmProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  riskLevel?: RiskLevel | string | null;
  disabled?: boolean;
  id?: string;
}

export function UnrestrictedApprovalConfirm({
  checked,
  onCheckedChange,
  riskLevel,
  disabled,
  id = "confirm-unrestricted",
}: UnrestrictedApprovalConfirmProps) {
  const risk = effectiveRiskLevel(riskLevel);

  return (
    <div className={`space-y-3 rounded-lg border p-3 ${riskCardClass(risk)}`}>
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 size-4 shrink-0" />
        <p className="text-sm">{unrestrictedApprovalWarning(riskLevel)}</p>
      </div>
      <div className="flex items-start gap-2">
        <Checkbox
          id={id}
          checked={checked}
          disabled={disabled}
          onCheckedChange={(value) => onCheckedChange(value === true)}
        />
        <Label htmlFor={id} className="text-sm font-normal leading-snug">
          I understand this approves any parameters and will not prompt me
        </Label>
      </div>
    </div>
  );
}
