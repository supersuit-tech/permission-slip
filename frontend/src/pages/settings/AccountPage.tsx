import { PaymentMethodSection } from "./PaymentMethodSection";
import { DangerZoneSection } from "./DangerZoneSection";

export function AccountPage() {
  return (
    <div className="space-y-10">
      <PaymentMethodSection />
      <DangerZoneSection />
    </div>
  );
}
