import { Alert, Button, Modal, Typography } from "@plakar-ui/ui/components";

type QuitDialogProps = {
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
  onConfirm: () => void;
  isPending?: boolean;
  error?: string;
};

export function QuitDialog({
  isOpen,
  onOpenChange,
  onConfirm,
  isPending = false,
  error,
}: QuitDialogProps) {
  return (
    <Modal
      isDismissable
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      aria-label="Quit Plakar?"
    >
      <Modal.Dialog className="max-w-md">
        <Modal.Header
          title="Quit Plakar?"
          description="This stops the plakar process serving this page."
        />
        <Modal.Content className="flex flex-col gap-4">
          <Typography size="sm" className="text-secondary">
            This page stops working for everyone using it. Your repository is
            not affected: run <code>plakar ui</code> again to come back.
          </Typography>

          {error && (
            <Alert variant="error" title="Unable to quit" size="sm">
              {error}
            </Alert>
          )}
        </Modal.Content>
        <Modal.Footer className="flex flex-wrap justify-end gap-2">
          <Button
            variant="secondary"
            onPress={() => onOpenChange(false)}
            isDisabled={isPending}
          >
            Stay
          </Button>
          <Button
            variant="primary"
            leftIcon="power-off"
            onPress={onConfirm}
            isPending={isPending}
          >
            Quit
          </Button>
        </Modal.Footer>
      </Modal.Dialog>
    </Modal>
  );
}
