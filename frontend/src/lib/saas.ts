/**
 * Whether this instance is running as the hosted SaaS product.
 * When false (the default for self-hosted deployments), company-specific
 * pages like /terms, /privacy, /cookies, and /support are excluded.
 */
export const isSaas = import.meta.env.VITE_PERMISSION_SLIP_SAAS === "true";
