import { Button, Skeleton } from "@plakar-ui/ui/components";
import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { Fragment, ReactNode, useState } from "react";

import {
  IntegrationCatalog,
  IntegrationCatalogFilters,
  IntegrationCatalogItem,
  integrationShelfQueryOptions,
  isUpgradable,
} from "./integration-catalog";
import {
  IntegrationSectionError,
  IntegrationSectionHeader,
} from "./integration-section";
import { IntegrationTileSkeleton } from "./integration-skeletons";

const COLLAPSED_TILES = 6;

function upgradableFirst<T extends IntegrationCatalogItem>(integrations: T[]) {
  return [...integrations].sort(
    (a, b) => Number(isUpgradable(b)) - Number(isUpgradable(a))
  );
}

type IntegrationShelfProps<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
> = IntegrationCatalogFilters<Type> & {
  catalog: IntegrationCatalog<T, Type, Status>;
  title: string;
  status: Status;
  action?: ReactNode;
  isHiddenWhileLoading?: boolean;
};

export function IntegrationShelf<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
>({
  catalog,
  title,
  status,
  search,
  types,
  action,
  isHiddenWhileLoading = false,
}: IntegrationShelfProps<T, Type, Status>) {
  const [isExpanded, setIsExpanded] = useState(false);
  const query = useInfiniteQuery({
    ...integrationShelfQueryOptions(catalog, status, { search, types }),
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
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3">
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
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3">
          {visible.map((integration) => (
            <Fragment key={integration.id}>
              {catalog.renderTile(integration)}
            </Fragment>
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
