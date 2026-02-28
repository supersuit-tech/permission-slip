import { PolicyLayout } from "./PolicyLayout";

export function TermsOfServicePage() {
  return (
    <PolicyLayout title="Terms of Service">
      <p className="text-muted-foreground">
        Our terms of service are coming soon. This page will cover acceptable
        use, account termination, liability limitations, dispute resolution, IP
        ownership, beta disclaimers, payment and subscription terms, and our
        refund and cancellation policy.
      </p>
    </PolicyLayout>
  );
}
