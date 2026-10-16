import { RepositoryInfo } from "@plakar-ui/common/api/kloset";
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
      tooltip:
        "A chunking algorithm splits data into smaller pieces, or “chunks,” based on specific rules (e.g., fixed size or content-defined). This enables efficient deduplication by identifying and eliminating redundant data. Plakar currently uses FastCDC, a fast and content-aware chunking algorithm.",
    },
    {
      label: "Compression",
      value: config?.compression?.algorithm ?? "Not compressed",
      tooltip:
        "A compression algorithm reduces the size of data by encoding patterns and redundancies, helping save storage space and speed up transfers. Different algorithms offer trade-offs between speed, compression ratio, and resource usage. Plakar currently uses LZ4, a high-speed compression algorithm optimized for fast storage and retrieval.",
    },
    {
      label: "Encryption",
      value: config?.encryption?.data_algorithm ?? "Not encrypted",
      tooltip:
        "An encryption algorithm protects data by transforming it into an unreadable format, ensuring confidentiality and integrity. In encrypted repositories, the only way to access the data is with the correct passphrase — without it, the data remains completely inaccessible.",
    },
  ];
}
