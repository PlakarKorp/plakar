import {
  RepositorySnapshotsQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { useBreakpoint } from "@plakar-ui/common/hooks";
import { Button, Card, EmptyList } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { ComponentProps } from "react";

import { Snapshots } from "~/components/dashboard/snapshot-cards/snapshot-cards";

const PER_PAGE = 10;

const button = {
  children: "View all",
  href: {
    to: "/snapshots",
  },
  rightIcon: "arrow-up-right",
} as ComponentProps<typeof Button>;

export function CardLatestSnapshots() {
  const { isMdUp } = useBreakpoint();
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
    <Card>
      <Card.Header title="Latest restore points" buttonProps={button} />
      <Card.Content noPaddingY>
        {isPending ? (
          <Snapshots>
            {[...Array(PER_PAGE).keys()].map((i) => (
              <Snapshots.RowSkeleton
                key={i}
                withSources={isMdUp}
                withSize={isMdUp}
              />
            ))}
          </Snapshots>
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
          <Snapshots>
            {data.items.map((snapshot) => {
              return (
                <Snapshots.Row
                  key={snapshot.identifier.Value}
                  id={snapshot.identifier.Value}
                  sources={
                    isMdUp
                      ? snapshot.sources.map((source) => ({
                          type: source.importer.type,
                          origin: source.importer.origin,
                          directory: source.importer.directory.Value,
                        }))
                      : undefined
                  }
                  timeSince={snapshot.timestamp.relativeToNow()}
                  dateModified={snapshot.timestamp.toHumanString()}
                  size={isMdUp ? snapshot.total.size.Pretty : undefined}
                  itemsLength={isMdUp ? snapshot.total.files : undefined}
                  errorsLength={snapshot.total.errors}
                  snapshotLink={{
                    to: "/snapshots/$snapshotId/$source/browse",
                    params: {
                      snapshotId: snapshot.identifier.Value,
                      source: 0,
                    },
                    search: {
                      path: `${snapshot.sources[0].importer.directory.Value}`,
                    },
                  }}
                />
              );
            })}
          </Snapshots>
        )}
      </Card.Content>
    </Card>
  );
}
