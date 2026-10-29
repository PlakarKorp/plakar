import { RepositoryInfoQueryOptions } from "@plakar-ui/common/api/kloset";
import { Main } from "@plakar-ui/common/layouts";
import { createPaginationSearchSchema } from "@plakar-ui/common/utils";
import { EmptyList } from "@plakar-ui/ui/components";
import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import {
  IntegrationInstallationStatusFilterSchema,
  IntegrationTypeSchema,
} from "~/api/integrations/schema";
import { IntegrationAlertLoginRequired } from "~/components/integration-card/integration-alert-login-required";
import { IntegrationPagedGrid } from "~/components/integration-card/integration-paged-grid";
import { IntegrationShelf } from "~/components/integration-card/integration-shelf";
import { IntegrationsFilters } from "~/components/integration-card/integrations.filters";
import { ReviewUpdatesControls } from "~/components/integration-card/review-updates-controls";
import {
  shownCount,
  useIntegrationCounts,
} from "~/components/integration-card/use-integration-counts";
import { useAuthentication } from "~/hooks/use-authentication";

export const Route = createFileRoute("/_layout/integrations/")({
  validateSearch: createPaginationSearchSchema()
    .pick({ page: true })
    .extend({
      search: z.string().catch("").default(""),
      types: z.array(IntegrationTypeSchema).optional(),
      installation_status: IntegrationInstallationStatusFilterSchema.optional(),
    }),
  loader: async ({ context }) => {
    return context.queryClient.ensureQueryData(
      RepositoryInfoQueryOptions(context.klosetAPI, { repository: undefined })
    );
  },
  component: RouteComponent,
});

function RouteComponent() {
  const search = Route.useSearch();
  const repositoryInfo = Route.useLoaderData();
  const navigate = Route.useNavigate();
  const { isAuthenticated } = useAuthentication();
  const installDisabled = !isAuthenticated;

  const status = search.installation_status;
  const counts = useIntegrationCounts({
    search: search.search,
    types: search.types,
  });

  const isFiltered =
    search.search !== "" ||
    (search.types !== undefined && search.types.length > 0) ||
    status !== undefined;

  const isAll = status === undefined;
  const isInstalledView = status === "installed" || status === "upgradable";
  const showBuiltIn = isAll || status === "installed" || status === "built-in";
  const showAvailable = isAll || status === "not-installed";

  const reviewUpdates = !!counts.upgradable && (
    <ReviewUpdatesControls
      count={counts.upgradable}
      isDisabled={installDisabled}
      search={search.search}
      types={search.types}
    />
  );

  const pagination = {
    search: search.search,
    types: search.types,
    page: search.page,
    onPageChange: (page: number) =>
      navigate({ search: (previous) => ({ ...previous, page }) }),
    installDisabled,
  };

  const updateSearch = (patch: Partial<typeof search>) =>
    navigate({
      search: (previous) => ({ ...previous, page: 0, ...patch }),
      replace: true,
    });

  return (
    <Main
      icon="plug"
      iconBadgeType="accent"
      title="Integrations"
      description="Extend Plakar with integrations."
    >
      <div className="flex flex-col gap-6">
        {!isAuthenticated && (
          <IntegrationAlertLoginRequired
            os={repositoryInfo.item.os}
            arch={repositoryInfo.item.arch}
          />
        )}
        <IntegrationsFilters
          search={search.search}
          types={search.types}
          status={status}
          counts={counts}
          onSearchChange={(value) => updateSearch({ search: value })}
          onTypesChange={(types) => updateSearch({ types })}
          onStatusChange={(installation_status) =>
            updateSearch({ installation_status })
          }
          onClear={
            isFiltered
              ? () =>
                  updateSearch({
                    search: "",
                    types: undefined,
                    installation_status: undefined,
                  })
              : undefined
          }
        />

        {shownCount(counts, status) === 0 ? (
          <EmptyList
            size="sm"
            title={
              isFiltered
                ? "No integrations match your filters"
                : "No integrations found"
            }
            description={
              isFiltered
                ? "Try adjusting the search or clearing filters."
                : "Here, you should see all the integrations available for Plakar. Unfortunately, there are none at the moment."
            }
            illustrationType="juggles_nuts"
          />
        ) : (
          <>
            {isAll && (
              <IntegrationShelf
                title="Installed"
                status="installed"
                search={search.search}
                types={search.types}
                installDisabled={installDisabled}
                action={reviewUpdates}
              />
            )}
            {isInstalledView && (
              <IntegrationPagedGrid
                title={
                  status === "upgradable" ? "Updates available" : "Installed"
                }
                status={status}
                action={reviewUpdates}
                {...pagination}
              />
            )}
            {showBuiltIn && (
              <IntegrationShelf
                title="Built-in"
                status="built-in"
                search={search.search}
                types={search.types}
                installDisabled={installDisabled}
                isHiddenWhileLoading
              />
            )}
            {showAvailable && (
              <IntegrationPagedGrid
                title="Available"
                status="not-installed"
                {...pagination}
              />
            )}
          </>
        )}
      </div>
    </Main>
  );
}
