import { ReactNode } from "react";

type CountBadgeProps = {
  children: ReactNode;
};

export function CountBadge({ children }: CountBadgeProps) {
  return (
    <span className="bg-tertiary text-tertiary rounded-full px-1.5 font-sans text-xs font-semibold tabular-nums">
      {children}
    </span>
  );
}
