import { Button, Modal } from "@plakar-ui/ui/components";
import std_path from "path";

import { DiffSide, TextDiff } from "./text-diff";

type DiffModalProps = {
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
  source: number;
  before: DiffSide;
  after: DiffSide;
};

/**
 * Opens the line diff of two versions of a file in a dialog.
 */
export function DiffModal({
  isOpen,
  onOpenChange,
  source,
  before,
  after,
}: DiffModalProps) {
  return (
    <Modal
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      isDismissable
      aria-label={`Changes to ${std_path.basename(after.path)}`}
    >
      <Modal.Dialog className="w-[min(1100px,95vw)]">
        <Modal.Header
          title={std_path.basename(after.path)}
          description={after.path}
        />
        <Modal.Content>
          {isOpen && <TextDiff source={source} before={before} after={after} />}
        </Modal.Content>
        {(before.href || after.href) && (
          <Modal.Footer className="flex flex-wrap justify-end gap-2">
            {[before, after].map(
              (side) =>
                side.href && (
                  <Button
                    key={side.snapshot}
                    variant="secondary"
                    leftIcon="arrow-up-right"
                    href={side.href}
                  >
                    Open {side.label}
                  </Button>
                )
            )}
          </Modal.Footer>
        )}
      </Modal.Dialog>
    </Modal>
  );
}
