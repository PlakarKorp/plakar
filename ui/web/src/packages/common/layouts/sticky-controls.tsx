import clsx from "clsx";
import { ReactNode } from "react";

export type StickyControlsProps = {
  children: ReactNode;
};

export function StickyControls({ children }: StickyControlsProps) {
  return (
    <div
      className={clsx(
        "absolute top-0 right-0 left-0 z-5",
        "bg-primary border-primary border-b"
      )}
    >
      <div className="px-2 py-3 lg:px-4 2xl:container 2xl:mx-auto">
        <div className="flex items-center justify-end">{children}</div>
      </div>
    </div>
  );
}
