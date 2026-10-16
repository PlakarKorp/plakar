import { Alert, Button } from "@plakar-ui/ui/components";
import { LinkProps } from "react-aria-components";

import { VFSEntryType } from "@plakar-ui/common/api/kloset";

type SnapshotErrorsAlertProps = {
  vfsEntry: VFSEntryType;
  href: LinkProps["href"];
};

export function SnapshotErrorsAlert({
  vfsEntry,
  href,
}: SnapshotErrorsAlertProps) {
  const numberOfErrors = vfsEntry.summary?.Total.errors ?? "several";

  return (
    <Alert
      title="Unable to backup some files in this directory"
      variant="error"
      actions={
        <Button variant="secondary" href={href}>
          View Errors
        </Button>
      }
    >
      Backup completed with <strong>{numberOfErrors} errors</strong> in this
      directory and its subdirectories.
    </Alert>
  );
}
