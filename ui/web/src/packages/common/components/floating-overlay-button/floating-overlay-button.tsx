import { Icon, IconType } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { Button } from "react-aria-components";

type FloatingOverlayButtonProps = {
  icon: IconType;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  content: ReactNode;
  bottom?: string;
};

export function FloatingOverlayButton({
  icon,
  defaultOpen = false,
  onOpenChange,
  content,
  bottom,
}: FloatingOverlayButtonProps) {
  const [open, setOpen] = useState(defaultOpen);
  const rootRef = useRef<HTMLDivElement>(null);

  const handleClose = useCallback(() => {
    setOpen(false);
    onOpenChange?.(false);
  }, [onOpenChange]);

  const handleToggle = () => {
    setOpen((prev) => {
      const next = !prev;
      onOpenChange?.(next);
      return next;
    });
  };

  useEffect(() => {
    if (!open) return;

    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      handleClose();
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        handleClose();
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("touchstart", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("touchstart", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, handleClose]);

  return (
    <div
      ref={rootRef}
      className="fixed right-6 bottom-6 z-100 flex items-end justify-end md:right-8 md:bottom-8"
      style={{ bottom }}
    >
      <div
        aria-hidden={!open}
        inert={open ? undefined : true}
        className={clsx(
          "border-accent bg-primary text-primary absolute right-0 bottom-12 z-99 w-80 overflow-hidden rounded-xl rounded-br-none border shadow-xl transition-all duration-300 ease-out",
          open
            ? "translate-x-0 opacity-100"
            : "pointer-events-none translate-x-60 opacity-0"
        )}
      >
        {content}
      </div>

      <Button
        aria-label={open ? "Close overlay" : "Open overlay"}
        className={clsx(
          "relative z-100 flex p-3",
          "bg-accent text-on-brand cursor-pointer rounded-full shadow-xl",
          "hover:bg-accent-hover transition-transform duration-200 ease-in-out hover:scale-105",
          !open && "rounded-br-none"
        )}
        onPress={handleToggle}
      >
        <Icon type={open ? "close" : icon} size="md" />
      </Button>
    </div>
  );
}
