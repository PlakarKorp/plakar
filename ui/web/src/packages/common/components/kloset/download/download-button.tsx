import { Button, ButtonProps, Menu } from "@plakar-ui/ui/components";

import { useDownloadSnapshot } from "./use-download-snapshot";

type DownloadButtonProps = {
  snapshotId: string;
  paths: string[];
} & ButtonProps;

export function DownloadButton({
  snapshotId,
  paths,
  ...buttonProps
}: DownloadButtonProps) {
  const { downloadSnapshot, isPending } = useDownloadSnapshot({
    snapshotId,
    paths,
  });

  return (
    <Button
      onPress={downloadSnapshot}
      isPending={isPending}
      leftIcon="download"
      {...buttonProps}
    />
  );
}

type DownloadMenuItemProps = {
  snapshotId: string;
  paths: string[];
};

export function DownloadMenuItem({ snapshotId, paths }: DownloadMenuItemProps) {
  const { downloadSnapshot } = useDownloadSnapshot({
    snapshotId,
    paths,
  });

  return (
    <Menu.Item
      label="Download"
      onPress={downloadSnapshot}
      leftIcon="download"
    />
  );
}
