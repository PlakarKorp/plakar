import clsx from "clsx";
import { ComponentProps } from "react";

export function Kbd({ className, ...props }: ComponentProps<"kbd">) {
  return (
    <kbd
      {...props}
      className={clsx(
        "bg-tertiary border-secondary text-tertiary inline-flex h-5 min-w-5 items-center justify-center rounded-md border border-b-2 px-1.5 font-sans text-xs leading-none font-semibold",
        className
      )}
    />
  );
}
