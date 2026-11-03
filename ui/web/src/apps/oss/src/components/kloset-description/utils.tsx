import { RepositoryInfo } from "@plakar-ui/common/api/kloset";
import { REPOSITORY_CONFIG_TOOLTIPS } from "@plakar-ui/common/components";
import { Id, Skeleton, Tooltip } from "@plakar-ui/ui/components";

import { Item } from "./kloset-description";

const getSkeleton = () => <Skeleton variant="text" size="xs" />;

export function getDescriptionListItems({
  repositoryInfo,
  isPending,
}: {
  repositoryInfo?: RepositoryInfo;
  isPending: boolean;
}): Item[] {
  const config = repositoryInfo?.configuration;
  const location = repositoryInfo?.location;

  if (isPending) {
    return [
      { label: "ID", value: getSkeleton() },
      { label: "Location", value: getSkeleton() },
      { label: "Chunking", value: getSkeleton() },
      { label: "Compression", value: getSkeleton() },
      { label: "Encryption", value: getSkeleton() },
    ];
  }

  return [
    {
      label: "ID",
      value: (
        <Id
          value={config?.repository_id?.Value ?? "-"}
          size="xs"
          className="h-4"
        />
      ),
    },
    {
      label: "Location",
      value: location ? (
        <Tooltip label={location}>
          <span role="button">{location}</span>
        </Tooltip>
      ) : (
        "-"
      ),
    },
    {
      label: "Chunking",
      value: config?.chunking?.algorithm ?? "Chunking disabled",
      tooltip: REPOSITORY_CONFIG_TOOLTIPS.chunking,
    },
    {
      label: "Compression",
      value: config?.compression?.algorithm ?? "Not compressed",
      tooltip: REPOSITORY_CONFIG_TOOLTIPS.compression,
    },
    {
      label: "Encryption",
      value: config?.encryption?.data_algorithm ?? "Not encrypted",
      tooltip: REPOSITORY_CONFIG_TOOLTIPS.encryption,
    },
  ];
}
