import { IntegrationCardSkeleton } from "@plakar-ui/common/components";
import { Pagination } from "@plakar-ui/ui/components";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import clsx from "clsx";
import { ReactNode, useEffect } from "react";

import { listIntegrationsQueryOptions } from "~/api/integrations/queries";
import type {
  IntegrationInstallationStatusFilter,
  IntegrationType,
} from "~/api/integrations/schema";
import { IntegrationCardAdapter } from "~/components/integration-card/integration-adapters";
import { CARD_GRID } from "~/components/integration-card/integration-grids";
import {
  IntegrationSectionError,
  IntegrationSectionHeader,
} from "~/components/integration-card/integration-section";

const PER_PAGE = 24;
const SKELETON_CARDS = 6;

type IntegrationPagedGridProps = {
  title: string;
  status: IntegrationInstallationStatusFilter;
  search: string;
  types: IntegrationType[] | undefined;
  page: number;
  onPageChange: (page: number) => void;
  installDisabled: boolean;
  action?: ReactNode;
};

export function IntegrationPagedGrid({
  title,
  status,
  search,
  types,
  page,
  onPageChange,
  installDisabled,
  action,
}: IntegrationPagedGridProps) {
  const query = useQuery({
    ...listIntegrationsQueryOptions({
      search,
      types,
      installationStatus: status,
      page,
      perPage: PER_PAGE,
    }),
    placeholderData: keepPreviousData,
  });
  const total = query.data?.total;
  const lastPage =
    total === undefined || query.isPlaceholderData
      ? undefined
      : Math.max(Math.ceil(total / PER_PAGE) - 1, 0);

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
        <div className={CARD_GRID}>
          {Array.from({ length: SKELETON_CARDS }, (_, index) => (
            <IntegrationCardSkeleton key={index} />
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
          <div className={clsx(CARD_GRID, query.isFetching && "animate-pulse")}>
            {query.data.items.map((integration) => (
              <IntegrationCardAdapter
                key={integration.id}
                integration={integration}
                installDisabled={installDisabled}
              />
            ))}
          </div>
          <Pagination
            currentPage={page}
            total={{ count: query.data.total, perPage: PER_PAGE }}
            onPrev={page > 0 ? () => onPageChange(page - 1) : undefined}
            onNext={
              (page + 1) * PER_PAGE < query.data.total
                ? () => onPageChange(page + 1)
                : undefined
            }
          />
        </>
      )}
    </section>
  );
}
