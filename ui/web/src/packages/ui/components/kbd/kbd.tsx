import clsx from "clsx";
import { PropsWithChildren } from "react";

type KbdProps = PropsWithChildren<{
  className?: string;
}>;

export function Kbd({ children, className }: KbdProps) {
  return (
    <kbd
      className={clsx(
        "border-primary bg-secondary text-tertiary inline-flex h-5 min-w-5 items-center justify-center rounded border border-b-2 px-1 font-sans text-xs leading-none font-medium",
        className
      )}
    >
      {children}
    </kbd>
  );
}
