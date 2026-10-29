import { IntegrationTileSkeleton } from "@plakar-ui/common/components";
import { Button, Skeleton } from "@plakar-ui/ui/components";
import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { ReactNode, useState } from "react";

import { listIntegrationsInfiniteQueryOptions } from "~/api/integrations/queries";
import type { Integration, IntegrationType } from "~/api/integrations/schema";
import { IntegrationTileAdapter } from "~/components/integration-card/integration-adapters";
import { TILE_GRID } from "~/components/integration-card/integration-grids";
import {
  IntegrationSectionError,
  IntegrationSectionHeader,
} from "~/components/integration-card/integration-section";
import { isUpgradable } from "~/components/integration-card/use-integration-actions";

const COLLAPSED_TILES = 6;
const PER_PAGE = 50;

function upgradableFirst(integrations: Integration[]) {
  return [...integrations].sort(
    (a, b) => Number(isUpgradable(b)) - Number(isUpgradable(a))
  );
}

type IntegrationShelfProps = {
  title: string;
  status: "installed" | "built-in";
  search: string;
  types: IntegrationType[] | undefined;
  installDisabled: boolean;
  action?: ReactNode;
  isHiddenWhileLoading?: boolean;
};

export function IntegrationShelf({
  title,
  status,
  search,
  types,
  installDisabled,
  action,
  isHiddenWhileLoading = false,
}: IntegrationShelfProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const query = useInfiniteQuery({
    ...listIntegrationsInfiniteQueryOptions({
      search,
      types,
      installationStatus: status,
      perPage: PER_PAGE,
    }),
    placeholderData: keepPreviousData,
  });

  const pages = query.data?.pages ?? [];
  const total = pages[0]?.total;
  const integrations = upgradableFirst(pages.flatMap((page) => page.items));
  const visible = isExpanded
    ? integrations
    : integrations.slice(0, COLLAPSED_TILES);
  const hiddenCount = integrations.length - visible.length;

  if (
    (query.isSuccess && total === 0) ||
    (isHiddenWhileLoading && query.isPending)
  ) {
    return null;
  }

  return (
    <section className="flex flex-col gap-3" aria-label={title}>
      <IntegrationSectionHeader title={title} total={total} action={action} />

      {query.isPending ? (
        <>
          <div className={TILE_GRID}>
            {Array.from({ length: COLLAPSED_TILES }, (_, index) => (
              <IntegrationTileSkeleton key={index} />
            ))}
          </div>
          <div className="flex justify-center">
            <Skeleton variant="rectangle" width={136} height={36} />
          </div>
        </>
      ) : query.isError ? (
        <IntegrationSectionError
          title={title}
          error={query.error}
          onRetry={() => query.refetch()}
        />
      ) : (
        <div className={TILE_GRID}>
          {visible.map((integration) => (
            <IntegrationTileAdapter
              key={integration.id}
              integration={integration}
              installDisabled={installDisabled}
            />
          ))}
        </div>
      )}

      {!isExpanded && hiddenCount > 0 && (
        <ShelfToggle onPress={() => setIsExpanded(true)}>
          Show {hiddenCount} more
        </ShelfToggle>
      )}
      {isExpanded && query.hasNextPage && (
        <ShelfToggle
          onPress={() => query.fetchNextPage()}
          isPending={query.isFetchingNextPage}
        >
          Load more
        </ShelfToggle>
      )}
    </section>
  );
}

function ShelfToggle({
  children,
  onPress,
  isPending = false,
}: {
  children: ReactNode;
  onPress: () => void;
  isPending?: boolean;
}) {
  return (
    <div className="flex items-center gap-3">
      <span className="bg-border-secondary h-px flex-1" />
      <Button
        variant="tertiary"
        rightIcon="chevron-down"
        onPress={onPress}
        isPending={isPending}
      >
        {children}
      </Button>
      <span className="bg-border-secondary h-px flex-1" />
    </div>
  );
}
