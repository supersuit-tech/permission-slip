import { Badge } from "@/components/ui/badge";
import { Asterisk } from "lucide-react";

export function UnrestrictedBadge() {
  return (
    <Badge variant="warning-soft" className="font-normal">
      <Asterisk className="size-3" />
      Unrestricted
    </Badge>
  );
}
