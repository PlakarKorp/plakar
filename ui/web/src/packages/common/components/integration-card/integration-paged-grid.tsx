import { Pagination } from "@plakar-ui/ui/components";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import clsx from "clsx";
import { Fragment, ReactNode, useEffect } from "react";

import {
  IntegrationCatalog,
  IntegrationCatalogFilters,
  INTEGRATION_GRID_PAGE_SIZE,
  IntegrationCatalogItem,
  integrationGridQueryOptions,
} from "./integration-catalog";
import {
  IntegrationSectionError,
  IntegrationSectionHeader,
} from "./integration-section";
import { IntegrationCardSkeleton } from "./integration-skeletons";

const SKELETON_CARDS = 6;

type IntegrationPagedGridProps<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
> = IntegrationCatalogFilters<Type> & {
  catalog: IntegrationCatalog<T, Type, Status>;
  title: string;
  status: Status;
  page: number;
  onPageChange: (page: number) => void;
  action?: ReactNode;
};

export function IntegrationPagedGrid<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
>({
  catalog,
  title,
  status,
  search,
  types,
  page,
  onPageChange,
  action,
}: IntegrationPagedGridProps<T, Type, Status>) {
  const query = useQuery({
    ...integrationGridQueryOptions(catalog, status, { search, types }, page),
    placeholderData: keepPreviousData,
  });
  const total = query.data?.total;
  const lastPage =
    total === undefined || query.isPlaceholderData
      ? undefined
      : Math.max(Math.ceil(total / INTEGRATION_GRID_PAGE_SIZE) - 1, 0);

  useEffect(() => {
    if (lastPage !== undefined && page > lastPage) {
      onPageChange(lastPage);
    }
  }, [page, lastPage, onPageChange]);

  if (query.isSuccess && total === 0) {
    return null;
  }

  return (
    <section className="flex flex-col gap-3" aria-label={title}>
      <IntegrationSectionHeader title={title} total={total} action={action} />

      {query.isPending ? (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: SKELETON_CARDS }, (_, index) => (
            <IntegrationCardSkeleton
              key={index}
              hasDocumentation={catalog.detailHref !== undefined}
            />
          ))}
        </div>
      ) : query.isError ? (
        <IntegrationSectionError
          title={title}
          error={query.error}
          onRetry={() => query.refetch()}
        />
      ) : (
        <>
          <div
            className={clsx(
              "grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3",
              query.isFetching && "animate-pulse"
            )}
          >
            {query.data.items.map((integration) => (
              <Fragment key={integration.id}>
                {catalog.renderCard(integration)}
              </Fragment>
            ))}
          </div>
          <Pagination
            currentPage={page}
            total={{
              count: query.data.total,
              perPage: INTEGRATION_GRID_PAGE_SIZE,
            }}
            onPrev={page > 0 ? () => onPageChange(page - 1) : undefined}
            onNext={
              (page + 1) * INTEGRATION_GRID_PAGE_SIZE < query.data.total
                ? () => onPageChange(page + 1)
                : undefined
            }
          />
        </>
      )}
    </section>
  );
}
