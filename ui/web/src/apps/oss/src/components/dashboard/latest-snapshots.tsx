import {
  RepositorySnapshotsQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { Button, Card, EmptyList } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { ComponentProps } from "react";

import {
  RestorePointList,
  RestorePointListSkeleton,
} from "~/components/dashboard/restore-point-list/restore-point-list";

const PER_PAGE = 10;

const button = {
  children: "View all",
  href: {
    to: "/snapshots",
  },
  rightIcon: "arrow-up-right",
} as ComponentProps<typeof Button>;

export function CardLatestSnapshots() {
  const { api, repository } = useKlosetAPI();
  const { data, isPending, isError, refetch, error } = useQuery(
    RepositorySnapshotsQueryOptions(api, {
      repository,
      pagination: {
        page: 0,
        perPage: PER_PAGE,
      },
    })
  );

  const isEmpty = !data?.items || data?.items.length === 0;

  return (
    <Card className="overflow-hidden">
      <Card.Header
        title="Latest restore points"
        description={
          data &&
          `${data.total.toLocaleString("en-US")} ${data.total === 1 ? "restore point" : "restore points"} in this Kloset store.`
        }
        buttonProps={button}
        isDividerFullWidth={isPending || (!isError && !isEmpty)}
      />
      {isPending ? (
        <RestorePointListSkeleton rows={PER_PAGE} />
      ) : isError ? (
        <EmptyList
          illustrationType="sad"
          title="Unable to load the latest restore points"
          description={error.message}
          actions={[
            {
              children: "Retry",
              onPress: () => refetch(),
              variant: "secondary",
            },
          ]}
        />
      ) : isEmpty ? (
        <EmptyList
          illustrationType="default_nut"
          title="No restore points yet"
          description="There are no restore points available in this Kloset repository."
        />
      ) : (
        <RestorePointList
          points={data.items.map((snapshot) => ({
            id: snapshot.identifier.Value,
            createdAt: snapshot.timestamp,
            sources: snapshot.sources.map((source) => ({
              type: source.importer.type,
              origin: source.importer.origin,
              directory: source.importer.directory.Value,
            })),
            duration: snapshot.duration.toShortString(),
            size: snapshot.total.size.Pretty,
            items: snapshot.total.files,
            errors: snapshot.total.errors,
            link: {
              to: "/snapshots/$snapshotId/$source/browse",
              params: {
                snapshotId: snapshot.identifier.Value,
                source: 0,
              },
              search: {
                path: `${snapshot.sources[0].importer.directory.Value}`,
              },
            },
          }))}
        />
      )}
    </Card>
  );
}
