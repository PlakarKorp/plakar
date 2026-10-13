import clsx from "clsx";

import { StatusDotVariant } from "./status-dot.constants";

type StatusDotProps = {
  variant: (typeof StatusDotVariant)[number];
  pending?: boolean;
  isOnAccent?: boolean;
};

export function StatusDot({
  variant,
  pending = false,
  isOnAccent = false,
}: StatusDotProps) {
  return (
    <span
      className={clsx(
        "inline-block h-2 w-2 shrink-0 rounded-full",
        isOnAccent
          ? variant === "neutral"
            ? "border-[1.5px] border-white"
            : "bg-white"
          : {
              success: "bg-success",
              error: "bg-error",
              warning: "bg-warning",
              neutral: "text-disable border-[1.5px] border-current",
            }[variant],
        pending && "animate-pulse"
      )}
    />
  );
}
