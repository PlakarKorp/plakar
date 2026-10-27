import { Button, Modal, Typography } from "@plakar-ui/ui/components";

type RemoveIntegrationModalProps = {
  displayName: string;
  version: string;
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
  onConfirm: () => void;
};

export function RemoveIntegrationModal({
  displayName,
  version,
  isOpen,
  onOpenChange,
  onConfirm,
}: RemoveIntegrationModalProps) {
  return (
    <Modal
      isDismissable
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      aria-label={`Remove ${displayName}?`}
    >
      <Modal.Dialog className="max-w-md">
        <Modal.Header
          title={`Remove ${displayName}?`}
          description={`Version ${version} will be uninstalled.`}
        />
        <Modal.Content>
          <Typography size="sm" className="text-secondary">
            Anything that uses this integration stops working until it is
            installed again.
          </Typography>
        </Modal.Content>
        <Modal.Footer className="flex flex-wrap justify-end gap-2">
          <Button variant="secondary" onPress={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="error"
            leftIcon="trash"
            onPress={() => {
              onConfirm();
              onOpenChange(false);
            }}
          >
            Remove
          </Button>
        </Modal.Footer>
      </Modal.Dialog>
    </Modal>
  );
}
