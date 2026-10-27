import { IconButton, Menu } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { ComponentProps } from "react";
import { MenuTrigger, Popover } from "react-aria-components";

import { EnrichedChildEntryType } from "@plakar-ui/common/api/kloset";
import { DownloadMenuItem } from "@plakar-ui/common/components";

import { useEntryMenu } from "./use-entry-menu";
import { useEntryURL } from "./use-entry-url";

type EntryMenuProps = {
  snapshot: string;
  source: number;
  entry: EnrichedChildEntryType;
  isSelected: boolean;
  onToggleSelected: () => void;
  onOpen: () => void;
  revealOnHover?: boolean;
  size?: ComponentProps<typeof IconButton>["size"];
  className?: string;
};

export function EntryMenu({
  snapshot,
  source,
  entry,
  isSelected,
  onToggleSelected,
  onOpen,
  revealOnHover = true,
  size = "sm",
  className,
}: EntryMenuProps) {
  const { isOpen, onOpenChange } = useEntryMenu(entry.path);
  const isDirectory = entry.file_info.mode.isDir();

  return (
    <MenuTrigger isOpen={isOpen} onOpenChange={onOpenChange}>
      <IconButton
        icon="ellipsis-vertical"
        aria-label={`Actions for ${entry.file_info.name}`}
        variant="tertiary"
        size={size}
        className={clsx(
          revealOnHover &&
            !isOpen &&
            "opacity-0 group-hover:opacity-100 data-focus-visible:opacity-100 pointer-coarse:opacity-100",
          className
        )}
      />
      <Popover placement="bottom end" offset={4}>
        <Menu>
          <Menu.Item
            label="Open"
            leftIcon={isDirectory ? "folder-open" : "eye"}
            onAction={onOpen}
          />
          {isDirectory ? (
            <DownloadMenuItem snapshotId={snapshot} paths={[entry.path]} />
          ) : (
            <FileDownloadMenuItem
              snapshot={snapshot}
              source={source}
              path={entry.path}
            />
          )}
          <Menu.Item
            label={isSelected ? "Deselect" : "Select"}
            leftIcon="check"
            onAction={onToggleSelected}
          />
        </Menu>
      </Popover>
    </MenuTrigger>
  );
}

function FileDownloadMenuItem({
  snapshot,
  source,
  path,
}: {
  snapshot: string;
  source: number;
  path: string;
}) {
  const { getUrl, isColdStorage } = useEntryURL({ snapshot, source, path });

  return (
    <Menu.Item
      label="Download"
      leftIcon="download"
      altText={isColdStorage ? "Cold storage" : undefined}
      href={getUrl?.({ download: true })}
      download
      isDisabled={!getUrl}
    />
  );
}
