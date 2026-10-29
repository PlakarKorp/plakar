import clsx from "clsx";
import { ReactNode } from "react";

type CountBadgeProps = {
  children: ReactNode;
  isOnAccent?: boolean;
};

export function CountBadge({ children, isOnAccent = false }: CountBadgeProps) {
  return (
    <span
      className={clsx(
        "rounded-full px-1.5 font-sans text-xs font-semibold tabular-nums",
        isOnAccent ? "text-on-brand bg-white/20" : "bg-tertiary text-tertiary"
      )}
    >
      {children}
    </span>
  );
}
