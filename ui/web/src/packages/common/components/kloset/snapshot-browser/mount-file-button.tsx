import { Button } from "@plakar-ui/ui/components";

import {
  SnapshotType,
  useKlosetAPI,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";

import { useCreateMount } from "../../../api/kloset/mutations";
import { addToast } from "../../toast";

type MountFileButtonProps = {
  snapshot: SnapshotType;
  source: number;
  vfsEntry: VFSEntryType;
  onMount?: (mountId: string) => void;
  text?: string;
};

export function MountFileButton({
  snapshot,
  source,
  vfsEntry,
  onMount,
  text = "Mount",
}: MountFileButtonProps) {
  const { api, repository } = useKlosetAPI();
  const mountMutation = useCreateMount({ api, repository });

  if (!api.mount) {
    return null;
  }

  const handleMount = async () => {
    try {
      const resp = await mountMutation.mutateAsync({
        snapshot: snapshot.identifier.Value,
        source,
        path: vfsEntry.path,
      });

      addToast({
        content: {
          title: "Mount request successful",
          children:
            "The mount request was successful. The mount should be ready in a few moments.",
          variant: "success",
        },
      });
      onMount?.(resp.mount_id);
    } catch (error) {
      addToast({
        content: {
          title: "Error mounting file",
          children:
            error instanceof Error ? error.message : "An error occurred",
          variant: "error",
        },
      });
    }
  };

  return (
    <Button
      variant="secondary"
      leftIcon="link"
      onPress={handleMount}
      isPending={mountMutation.isPending}
    >
      {text}
    </Button>
  );
}
