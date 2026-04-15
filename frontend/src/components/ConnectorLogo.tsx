import { Plug } from "lucide-react";
import DOMPurify, { type Config as DOMPurifyConfig } from "dompurify";
import { cn } from "@/lib/utils";

interface ConnectorLogoProps {
  name: string;
  logoSvg?: string | null;
  className?: string;
  size?: "sm" | "md" | "lg";
}

// Sanitize inline connector SVG strings before injecting them into the DOM.
// Connector logos are bundled at build time, but defense-in-depth against a
// compromised connector manifest, untrusted third-party connectors, or any
// future user-supplied logos requires stripping scripts, event handlers, and
// external references before rendering.
// DOMPurify's Config type expects mutable string[] fields, so this is a
// plain object rather than `as const`.
const SVG_SANITIZER_CONFIG: DOMPurifyConfig = {
  USE_PROFILES: { svg: true, svgFilters: true },
  // No MathML — connector logos are never math.
  // No <foreignObject> or <script> by default under the svg profile.
  FORBID_TAGS: ["script", "foreignObject"],
  FORBID_ATTR: ["onload", "onerror", "onclick", "href", "xlink:href"],
};

function sanitizeSVG(raw: string): string {
  return DOMPurify.sanitize(raw, SVG_SANITIZER_CONFIG);
}

const sizeClasses = {
  sm: "size-6",
  md: "size-8",
  lg: "size-10",
};

const fallbackTextSizes = {
  sm: "text-[9px]",
  md: "text-[11px]",
  lg: "text-sm",
};

function getInitials(name: string): string {
  const words = name.trim().split(/\s+/);
  const first = words[0] ?? "";
  const second = words[1] ?? "";
  if (words.length === 1) {
    return first.charAt(0).toUpperCase();
  }
  return (first.charAt(0) + second.charAt(0)).toUpperCase();
}

export function ConnectorLogo({
  name,
  logoSvg,
  className,
  size = "md",
}: ConnectorLogoProps) {
  const sizeClass = sizeClasses[size];

  if (logoSvg) {
    return (
      <div
        className={cn(
          "flex shrink-0 items-center justify-center overflow-hidden rounded",
          sizeClass,
          className,
        )}
        aria-hidden="true"
        dangerouslySetInnerHTML={{ __html: sanitizeSVG(logoSvg) }}
      />
    );
  }

  const initials = getInitials(name);

  if (initials) {
    return (
      <div
        className={cn(
          "bg-muted text-muted-foreground flex shrink-0 items-center justify-center rounded font-semibold",
          sizeClass,
          fallbackTextSizes[size],
          className,
        )}
        aria-hidden="true"
      >
        {initials}
      </div>
    );
  }

  return (
    <div
      className={cn(
        "bg-muted text-muted-foreground flex shrink-0 items-center justify-center rounded",
        sizeClass,
        className,
      )}
      aria-hidden="true"
    >
      <Plug className="size-[60%]" />
    </div>
  );
}
