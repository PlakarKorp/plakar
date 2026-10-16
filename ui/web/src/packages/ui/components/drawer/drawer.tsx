import { PropsWithChildren } from "react";
import { Dialog, Modal, ModalOverlay } from "react-aria-components";

import { IconButton, Typography } from "..";

type DrawerProps = PropsWithChildren<{
  open: boolean;
  setOpen: (open: boolean) => void;
  title: string;
}>;

export function Drawer({ open, setOpen, title, children }: DrawerProps) {
  return (
    <ModalOverlay
      isDismissable
      isOpen={open}
      onOpenChange={setOpen}
      className="text-primary fixed inset-0 z-50 bg-black/50 data-[entering]:animate-[modal-blur_300ms] data-[exiting]:animate-[modal-blur_300ms_reverse_ease-in]"
    >
      <Modal className="bg-primary fixed inset-y-0 right-0 w-full max-w-2xl overflow-y-scroll p-4 shadow-lg data-[entering]:animate-[modal-slide_300ms] data-[exiting]:animate-[modal-slide_300ms_reverse_ease-in]">
        <Dialog>
          <div className="mb-4 flex items-center justify-between gap-4">
            <Typography variant="heading" size="md" slot="title">
              {title}
            </Typography>
            <IconButton icon="close" slot="close" variant="tertiary" />
          </div>
          {children}
        </Dialog>
      </Modal>
    </ModalOverlay>
  );
}
