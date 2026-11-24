import { PropsWithChildren, ReactNode } from "react";
import { Dialog, Modal, ModalOverlay } from "react-aria-components";

import { IconButton, Typography } from "..";

type DrawerProps = PropsWithChildren<{
  open: boolean;
  setOpen: (open: boolean) => void;
  title: string;
  /** Stays at the bottom of the drawer, below the content that scrolls. */
  footer?: ReactNode;
}>;

export function Drawer({
  open,
  setOpen,
  title,
  footer,
  children,
}: DrawerProps) {
  return (
    <ModalOverlay
      isDismissable
      isOpen={open}
      onOpenChange={setOpen}
      className="text-primary fixed inset-0 z-50 bg-black/50 data-[entering]:animate-[modal-blur_300ms] data-[exiting]:animate-[modal-blur_300ms_reverse_ease-in]"
    >
      <Modal className="bg-overlay fixed inset-y-0 right-0 flex w-full max-w-2xl flex-col shadow-lg data-[entering]:animate-[modal-slide_300ms] data-[exiting]:animate-[modal-slide_300ms_reverse_ease-in]">
        <Dialog className="flex min-h-0 flex-1 flex-col outline-none">
          <div className="flex items-center justify-between gap-4 p-4">
            <Typography variant="heading" size="md" slot="title">
              {title}
            </Typography>
            <IconButton icon="close" slot="close" variant="tertiary" />
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
            {children}
          </div>
          {footer && (
            <div className="border-primary flex justify-end gap-2 border-t p-4">
              {footer}
            </div>
          )}
        </Dialog>
      </Modal>
    </ModalOverlay>
  );
}
