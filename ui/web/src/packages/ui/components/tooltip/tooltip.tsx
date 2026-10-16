import clsx from "clsx";
import { JSX, ReactNode, RefObject } from "react";
import {
  OverlayArrow as RAOverlayArrow,
  Tooltip as RATooltip,
  TooltipTrigger as RATooltipTrigger,
  Focusable as RAFocusable,
} from "react-aria-components";

import { Typography } from "..";

type TooltipProps = {
  label: ReactNode;
  children: JSX.Element;
  maxWidth?: string | number;
  offset?: number;
  delay?: number;
  closeDelay?: number;
  isOpen?: boolean;
  triggerRef?: RefObject<Element | null>;
};

export function Tooltip({
  label,
  children,
  maxWidth,
  offset = 4,
  delay = 0,
  closeDelay = 200,
  isOpen,
  triggerRef,
}: TooltipProps) {
  return (
    <RATooltipTrigger delay={delay} closeDelay={closeDelay} isOpen={isOpen}>
      <RAFocusable>{children}</RAFocusable>

      <RATooltip
        className="bg-foreground-black text-on-brand rounded-lg px-3 py-2 shadow-md"
        style={{ maxWidth: maxWidth }}
        offset={offset}
        triggerRef={triggerRef}
      >
        <RAOverlayArrow>
          {({ placement }) => (
            <svg
              width={8}
              height={8}
              viewBox="0 0 8 8"
              className={clsx({
                "-rotate-90": placement === "left",
                "rotate-90": placement === "right",
                "rotate-180": placement === "bottom",
              })}
            >
              <path d="M0 0 L4 4 L8 0" paintOrder="stroke" />
            </svg>
          )}
        </RAOverlayArrow>

        <Typography as="div" size="xs" weight="medium">
          {label}
        </Typography>
      </RATooltip>
    </RATooltipTrigger>
  );
}
