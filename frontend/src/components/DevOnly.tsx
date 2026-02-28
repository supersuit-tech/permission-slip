import type { ReactNode } from "react";

interface DevOnlyProps {
  children: ReactNode;
}

export default function DevOnly({ children }: DevOnlyProps) {
  if (!import.meta.env.DEV) return null;
  return <>{children}</>;
}
