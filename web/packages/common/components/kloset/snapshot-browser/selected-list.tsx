import { Button, Modal } from "@plakar-ui/ui/components";
import { SelectionControls } from "@plakar-ui/ui/components";
import { useBlocker } from "@tanstack/react-router";

import { Size } from "@plakar-ui/common/api/enrich";
import {
  EnrichedChildEntryType,
  SnapshotType,
} from "@plakar-ui/common/api/kloset";

import { DownloadButton } from "../..";

type SelectedListProps = {
  snapshot: SnapshotType;
  childEntries: EnrichedChildEntryType[];
  removeSelection: () => void;
};

export function SelectedList({
  snapshot,
  childEntries,
  removeSelection,
}: SelectedListProps) {
  const selectedDirectories = childEntries
    .filter((child) => child.file_info.mode.isDir())
    .sort((a, b) => a.file_info.name.localeCompare(b.file_info.name));
  const selectedFiles = childEntries
    .filter((child) => child.file_info.mode.isRegular())
    .sort((a, b) => a.file_info.name.localeCompare(b.file_info.name));

  const totalSize = Size(
    selectedDirectories.reduce(
      (acc, childEntry) => acc + (childEntry.summary?.Total.size.Value ?? 0),
      0
    ) +
      selectedFiles.reduce(
        (acc, childEntry) => acc + (childEntry.file_info.size.Value ?? 0),
        0
      )
  );

  return (
    <>
      <ConfirmExitModal
        numberOfItemsSelected={
          selectedDirectories.length + selectedFiles.length
        }
      />

      <SelectionControls
        text={`${childEntries.length} selected`}
        altText={totalSize.Pretty}
        onUnselect={removeSelection}
      >
        <DownloadButton
          snapshotId={snapshot.identifier.Value}
          paths={childEntries.map((child) => child.path)}
          variant="primary"
        >
          Download
        </DownloadButton>
      </SelectionControls>
    </>
  );
}

type ConfirmExitModalProps = {
  numberOfItemsSelected: number;
};

function ConfirmExitModal({ numberOfItemsSelected }: ConfirmExitModalProps) {
  const { proceed, reset, status } = useBlocker({
    shouldBlockFn: ({ current, next }) => {
      // Let the user leave the page if only one item is selected.
      if (numberOfItemsSelected < 2) {
        return false;
      }
      // If we change the page but stay in the same snapshot the selection is
      // preserved so let the user leave the page.
      if (current.pathname === next.pathname) {
        return false;
      }
      return true;
    },
    withResolver: true,
    // Disable the beforeunload event listener. We don't want to show the
    // browser's default message.
    enableBeforeUnload: false,
  });

  return (
    <Modal
      isDismissable
      isOpen={status === "blocked"}
      onOpenChange={() => reset?.()}
      aria-label="Leaving the page?"
    >
      <Modal.Dialog>
        <Modal.Header
          title="Leaving the page?"
          description={
            <>
              You have selected <strong>{numberOfItemsSelected}</strong> files
              and directories.
            </>
          }
        />
        <Modal.Content>
          If you leave this page, your selection will be lost.
        </Modal.Content>
        <Modal.Footer className="flex flex-wrap justify-end gap-2">
          <Button variant="secondary" onPress={() => reset?.()}>
            Stay here
          </Button>
          <Button variant="error" onPress={() => proceed?.()}>
            Leave the page
          </Button>
        </Modal.Footer>
      </Modal.Dialog>
    </Modal>
  );
}
