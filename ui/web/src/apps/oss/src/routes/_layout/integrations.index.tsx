import { RepositoryInfoQueryOptions } from "@plakar-ui/common/api/kloset";
import { Main } from "@plakar-ui/common/layouts";
import { createPaginationSearchSchema } from "@plakar-ui/common/utils";
import { EmptyList, Pagination, Spinner } from "@plakar-ui/ui/components";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import clsx from "clsx";
import { z } from "zod";

import { listIntegrationsQueryOptions } from "~/api/integrations/queries";
import {
  IntegrationInstallationStatusFilterSchema,
  IntegrationTypeSchema,
} from "~/api/integrations/schema";
import { OssApiToCard } from "~/components/integration-card/field-mapping";
import { IntegrationAlertLoginRequired } from "~/components/integration-card/integration-alert-login-required";
import { IntegrationCardAdapter } from "~/components/integration-card/integration-card-adapter";
import { IntegrationsFilters } from "~/components/integration-card/integrations.filters";
import { useAuthentication } from "~/hooks/use-authentication";

export const Route = createFileRoute("/_layout/integrations/")({
  validateSearch: createPaginationSearchSchema(30).extend({
    search: z.string().catch("").default(""),
    types: z.array(IntegrationTypeSchema).optional(),
    installation_status: IntegrationInstallationStatusFilterSchema.optional(),
  }),

  loaderDeps: ({ search }) => {
    return {
      page: search.page,
      per_page: search.per_page,
      search: search.search,
      types: search.types,
      installation_status: search.installation_status,
    };
  },

  loader: async ({ context, deps }) => {
    context.queryClient.prefetchQuery(
      listIntegrationsQueryOptions({
        page: deps.page,
        perPage: deps.per_page,
        search: deps.search,
        types: deps.types,
        installationStatus: deps.installation_status,
      })
    );
    return context.queryClient.ensureQueryData(
      RepositoryInfoQueryOptions(context.klosetAPI, { repository: undefined })
    );
  },
  component: RouteComponent,
});

function RouteComponent() {
  const search = Route.useSearch();

  const repositoryInfo = Route.useLoaderData();

  const { isAuthenticated } = useAuthentication();
  const query = useQuery({
    ...listIntegrationsQueryOptions({
      page: search.page,
      perPage: search.per_page,
      search: search.search,
      types: search.types,
      installationStatus: search.installation_status,
    }),
    placeholderData: keepPreviousData,
  });
  const integrations = query.data?.items ?? [];
  const total = query.data?.total ?? 0;

  const hasActiveFilters =
    search.search !== "" ||
    (search.types !== undefined && search.types.length > 0) ||
    search.installation_status !== undefined;

  const highlightedTypes = search.types?.map((t) => OssApiToCard[t]);

  const navigate = Route.useNavigate();

  const onPrev =
    search.page > 0
      ? () => {
          navigate({ search: (prev) => ({ ...prev, page: search.page - 1 }) });
        }
      : undefined;

  const onNext =
    (search.page + 1) * search.per_page < total
      ? () => {
          navigate({ search: (prev) => ({ ...prev, page: search.page + 1 }) });
        }
      : undefined;

  return (
    <Main
      icon="plug"
      iconBadgeType="accent"
      title="Integrations"
      description="Extend Plakar with integrations."
    >
      {query.isError ? (
        <EmptyList
          illustrationType="sad"
          title="Failed to load integrations"
          description={query.error?.message}
          actions={[
            {
              children: "Retry",
              onPress: () => query.refetch(),
              variant: "secondary",
            },
          ]}
        />
      ) : (
        <div className="flex flex-col gap-6">
          {!isAuthenticated && (
            <IntegrationAlertLoginRequired
              os={repositoryInfo.item.os}
              arch={repositoryInfo.item.arch}
            />
          )}
          <IntegrationsFilters />
          {query.isPending ? (
            <Spinner />
          ) : query.isSuccess && total === 0 ? (
            hasActiveFilters ? (
              <EmptyList
                title="No integrations match your filters"
                description="Try adjusting the search or clearing filters."
                illustrationType="juggles_nuts"
              />
            ) : (
              <EmptyList
                title="No integrations found"
                description="Here, you should see all the integrations available for Plakar. Unfortunately, there are none at the moment."
                illustrationType="juggles_nuts"
              />
            )
          ) : (
            <>
              <div
                className={clsx(
                  "grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3",
                  query.isFetching && "animate-pulse"
                )}
              >
                {integrations.map((integration) => (
                  <IntegrationCardAdapter
                    key={integration.id}
                    integration={integration}
                    documentationProps={{
                      href: {
                        to: "/integrations/$integrationId",
                        params: { integrationId: integration.id },
                      },
                    }}
                    installDisabled={!isAuthenticated}
                    highlightedTypes={highlightedTypes}
                  />
                ))}
              </div>

              <Pagination
                currentPage={search.page}
                total={{
                  count: total,
                  perPage: search.per_page,
                }}
                onPrev={onPrev}
                onNext={onNext}
              />
            </>
          )}
        </div>
      )}
    </Main>
  );
}
