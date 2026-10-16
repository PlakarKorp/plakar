import clsx from "clsx";
import { PropsWithChildren, ReactNode } from "react";
import {
  Dialog as AriaDialog,
  ModalOverlayProps,
  Modal as AriaModal,
  DialogProps,
  ModalOverlay,
} from "react-aria-components";

import { IconButton, Typography } from "..";

type ModalProps = PropsWithChildren<
  ModalOverlayProps & Pick<DialogProps, "aria-label">
>;

function ModalRoot({
  children,
  className,
  "aria-label": ariaLabel,
  ...modalProps
}: ModalProps) {
  return (
    <ModalOverlay
      {...modalProps}
      className={clsx(
        "group fixed inset-0 z-50 flex items-center justify-center bg-black/50",
        "data-[entering]:animate-[modal-fade_200ms]",
        "data-[exiting]:animate-[modal-fade_150ms_reverse_ease-in]",
        className
      )}
    >
      <AriaModal className="max-w-screen">
        <AriaDialog
          aria-label={ariaLabel}
          className="m-1 rounded-xl shadow-xl group-data-[entering]:animate-[modal-zoom_300ms_cubic-bezier(0.175,0.885,0.32,1.275)]"
        >
          {children}
        </AriaDialog>
      </AriaModal>
    </ModalOverlay>
  );
}

function ModalDialog({
  children,
  className,
}: PropsWithChildren<{ className?: string }>) {
  return (
    <div
      className={clsx(
        "border-secondary bg-primary flex max-h-[90vh] flex-col rounded-xl border",
        className
      )}
    >
      {children}
    </div>
  );
}

type ModalHeaderProps = {
  prefix?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  className?: string;
};

function ModalHeader({
  prefix,
  title,
  description,
  className,
}: ModalHeaderProps) {
  return (
    <div
      className={clsx(
        "border-secondary flex items-start justify-between gap-2 border-b p-4",
        className
      )}
      slot="header"
    >
      <div className="flex items-center gap-3">
        {prefix && <div className="shrink-0">{prefix}</div>}
        <div className="flex min-w-0 flex-col">
          <Typography
            as="h2"
            size="lg"
            weight="medium"
            className="text-primary line-clamp-2 md:line-clamp-1"
            slot="title"
          >
            {title}
          </Typography>
          {description && (
            <Typography size="sm" className="text-tertiary line-clamp-2">
              {description}
            </Typography>
          )}
        </div>
      </div>
      <IconButton
        icon="close"
        variant="tertiary"
        aria-label="Close modal"
        slot="close"
        className="-my-1"
      />
    </div>
  );
}

function ModalContent({
  children,
  className,
}: PropsWithChildren<{ className?: string }>) {
  return (
    <div
      className={clsx("text-primary flex-1 overflow-y-auto p-4", className)}
      slot="content"
    >
      {children}
    </div>
  );
}

function ModalFooter({
  children,
  className,
}: PropsWithChildren<{ className?: string }>) {
  return (
    <div
      className={clsx("border-secondary border-t p-4 text-right", className)}
      slot="footer"
    >
      {children}
    </div>
  );
}

export const Modal = Object.assign(ModalRoot, {
  Dialog: ModalDialog,
  Header: ModalHeader,
  Content: ModalContent,
  Footer: ModalFooter,
});
