import clsx from "clsx";

import { Typography } from "..";
import { StepBadgeVariant, StepBadgeSize } from "./step-badge.constants";

type StepBadgeProps = {
  /** The rank of the step in its sequence. */
  step: number;
  variant?: (typeof StepBadgeVariant)[number];
  size?: (typeof StepBadgeSize)[number];
  className?: string;
};

/**
 * The rank of one step of a form that shows all of its steps at once, meant to
 * sit beside the title it numbers. It says where in an order something falls,
 * not how far along the order is — a stepper is what tracks progress.
 */
export function StepBadge({
  step,
  variant = "default",
  size = "md",
  className,
}: StepBadgeProps) {
  return (
    <div
      className={clsx(
        "flex shrink-0 items-center justify-center rounded-md",
        {
          default: "bg-accent-heavy text-on-accent",
          accent: "bg-accent text-on-brand",
          muted: "bg-tertiary text-tertiary",
        }[variant],
        { sm: "h-5 w-5", md: "h-6 w-6" }[size],
        className
      )}
    >
      <Typography as="span" size="xs" weight="semibold" monospace>
        {step}
      </Typography>
    </div>
  );
}
