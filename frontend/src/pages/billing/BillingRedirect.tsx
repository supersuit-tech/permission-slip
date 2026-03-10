import { Navigate, useSearchParams } from "react-router-dom";

/** Redirects /billing → /settings/billing, preserving query params (e.g. Stripe checkout callback). */
export function BillingRedirect() {
  const [searchParams] = useSearchParams();
  const qs = searchParams.toString();
  const target = qs ? `/settings/billing?${qs}` : "/settings/billing";
  return <Navigate to={target} replace />;
}
