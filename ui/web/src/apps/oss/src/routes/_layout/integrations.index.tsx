import { RepositoryInfoQueryOptions } from "@plakar-ui/common/api/kloset";
import {
  IntegrationPagedGrid,
  IntegrationShelf,
  IntegrationsFilters,
  ReviewUpdatesButton,
  ReviewUpdatesModal,
  sumCounts,
  useIntegrationCounts,
} from "@plakar-ui/common/components";
import { Main } from "@plakar-ui/common/layouts";
import { createPaginationSearchSchema } from "@plakar-ui/common/utils";
import { EmptyList } from "@plakar-ui/ui/components";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { z } from "zod";

import { useInstallIntegration } from "~/api/integrations/mutations";
import {
  IntegrationInstallationStatusFilterSchema,
  IntegrationType,
  IntegrationTypeSchema,
} from "~/api/integrations/schema";
import { IntegrationAlertLoginRequired } from "~/components/integration-card/integration-alert-login-required";
import { integrationCatalog } from "~/components/integration-card/integration-catalog";
import { useAuthentication } from "~/hooks/use-authentication";

const COUNTED_STATUSES = IntegrationInstallationStatusFilterSchema.options;

const TYPE_ITEMS: Array<{ id: IntegrationType; label: string }> = [
  { id: "storage", label: "Storage" },
  { id: "source", label: "Source" },
  { id: "destination", label: "Destination" },
];

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
  const catalog = integrationCatalog(installDisabled);
  const install = useInstallIntegration();
  const [isReviewOpen, setIsReviewOpen] = useState(false);

  const status = search.installation_status;
  const filters = { search: search.search, types: search.types };
  const counts = useIntegrationCounts(catalog, COUNTED_STATUSES, filters);
  const installedCount = sumCounts(
    counts.byStatus.installed,
    counts.byStatus["built-in"]
  );
  const allCount = sumCounts(installedCount, counts.byStatus["not-installed"]);
  const shownCount =
    status === undefined
      ? allCount
      : status === "installed"
        ? installedCount
        : counts.byStatus[status];
  const upgradableCount = counts.byStatus.upgradable;

  const isFiltered =
    search.search !== "" ||
    (search.types !== undefined && search.types.length > 0) ||
    status !== undefined;

  const isAll = status === undefined;
  const isInstalledView = status === "installed" || status === "upgradable";
  const showBuiltIn = isAll || status === "installed" || status === "built-in";
  const showAvailable = isAll || status === "not-installed";

  const reviewUpdates = !!upgradableCount && (
    <ReviewUpdatesButton
      count={upgradableCount}
      isDisabled={installDisabled}
      onPress={() => setIsReviewOpen(true)}
    />
  );

  const pagination = {
    catalog,
    ...filters,
    page: search.page,
    onPageChange: (page: number) =>
      navigate({ search: (previous) => ({ ...previous, page }) }),
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
          onSearchChange={(value) => updateSearch({ search: value })}
          status={status}
          allCount={allCount}
          statusItems={[
            {
              id: "installed",
              label: "Installed",
              count: installedCount,
              dot: "success",
            },
            {
              id: "upgradable",
              label: "Updates available",
              count: upgradableCount,
              dot: "warning",
            },
            {
              id: "not-installed",
              label: "Not installed",
              count: counts.byStatus["not-installed"],
            },
          ]}
          onStatusChange={(installation_status) =>
            updateSearch({ installation_status })
          }
          types={search.types}
          typeItems={TYPE_ITEMS}
          onTypesChange={(types) => updateSearch({ types })}
          shown={shownCount}
          total={counts.unfiltered}
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

        {shownCount === 0 ? (
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
                catalog={catalog}
                title="Installed"
                status="installed"
                action={reviewUpdates}
                {...filters}
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
                catalog={catalog}
                title="Built-in"
                status="built-in"
                isHiddenWhileLoading
                {...filters}
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
        {isReviewOpen && (
          <ReviewUpdatesModal
            catalog={catalog}
            upgradableStatus="upgradable"
            onUpdate={async (integration) => {
              await install.mutateAsync({
                integrationId: integration.id,
                version: integration.latest_version,
              });
            }}
            onClose={() => setIsReviewOpen(false)}
            {...filters}
          />
        )}
      </div>
    </Main>
  );
}
