import clsx from "clsx";
import React, {
  PointerEvent,
  ReactElement,
  cloneElement,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  Popover as AriaPopover,
  OverlayArrow,
  type PressEvent,
  type ButtonProps,
} from "react-aria-components";

import { Typography } from "../typography/typography";

type PopoverProps = {
  label: React.ReactNode;
  children: ReactElement<ButtonProps>;
  maxWidth?: string | number;
};

export function Popover({ label, children, maxWidth }: PopoverProps) {
  const triggerRef = useRef<HTMLDivElement | null>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [isTouchMode, setIsTouchMode] = useState(false);
  const [isFocusMode, setIsFocusMode] = useState(false);

  const handlePress = (e: PressEvent) => {
    if (e.pointerType === "touch") {
      setIsTouchMode(true);
      setIsOpen(true);
    }
  };

  const handlePointerEnter = (e: PointerEvent) => {
    if (e.pointerType === "mouse") {
      setIsOpen(true);
      setIsTouchMode(false);
    }
  };

  const handlePointerLeave = (e: PointerEvent) => {
    if (e.pointerType === "mouse") {
      setIsOpen(false);
    }
  };

  useEffect(() => {
    if (!isOpen) return;

    const handleScroll = () => {
      setIsOpen(false);
      setIsTouchMode(false);
    };

    document.addEventListener("scroll", handleScroll, { capture: true });
    return () => {
      document.removeEventListener("scroll", handleScroll, { capture: true });
    };
  }, [isOpen]);

  return (
    <>
      {/* Trigger wrapper */}
      <div
        ref={triggerRef}
        className="flex w-fit max-w-full"
        onPointerEnter={handlePointerEnter}
        onPointerLeave={handlePointerLeave}
        onFocus={() => {
          if (!isOpen && !isTouchMode) {
            setIsFocusMode(true);
            setIsTouchMode(false);
            setIsOpen(true);
          }
        }}
        onBlur={() => {
          if (isFocusMode && !isTouchMode) setIsOpen(false);
        }}
      >
        {cloneElement(children, {
          onPress: (event: PressEvent) => {
            children.props.onPress?.(event);
            handlePress(event);
          },
        })}
      </div>

      {/* Touch-only underlay to catch outside taps and avoid triggering links/buttons */}
      {isOpen && isTouchMode && (
        <div
          className="fixed inset-0 z-[49]"
          aria-hidden="true"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setIsOpen(false);
            setIsTouchMode(false);
          }}
        />
      )}

      {/* Popover */}
      <AriaPopover
        isOpen={isOpen}
        isNonModal
        offset={4}
        triggerRef={triggerRef}
        onPointerEnter={handlePointerEnter}
        onPointerLeave={handlePointerLeave}
        className="bg-primary text-primary border-secondary z-[50] rounded-lg border px-3 py-2 shadow-md outline-none"
        style={{ maxWidth }}
        placement="top"
      >
        {({ placement }) => (
          <>
            {/* Bridges trigger/popover gap */}
            <div
              className="absolute -bottom-[5px] left-0 h-2 w-full"
              aria-hidden="true"
              role="presentation"
            />
            <div
              className="absolute -top-[5px] left-0 h-2 w-full"
              aria-hidden="true"
              role="presentation"
            />

            <OverlayArrow>
              <svg
                width={8}
                height={8}
                viewBox="0 0 8 8"
                className={clsx("fill-background-primary", {
                  "rotate-180": placement === "bottom",
                })}
              >
                <path
                  d="M0 0 L4 4 L8 0"
                  stroke="var(--color-border-secondary)"
                  strokeWidth="2"
                  paintOrder="stroke"
                />
              </svg>
            </OverlayArrow>

            <Typography
              as="div"
              size="xs"
              weight={typeof label === "string" ? "medium" : undefined}
            >
              {label}
            </Typography>
          </>
        )}
      </AriaPopover>
    </>
  );
}
