import { RepositoryInfoQueryOptions } from "@plakar-ui/common/api/kloset";
import {
  AppTableFeatures,
  appTableFeatures,
  PaginatedTable,
} from "@plakar-ui/common/components";
import { Main } from "@plakar-ui/common/layouts";
import { createPaginationSearchSchema } from "@plakar-ui/common/utils";
import {
  Alert,
  Button,
  EmptyList,
  Icon,
  IconType,
  Spinner,
  Typography,
} from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
  createColumnHelper,
  useTable as useReactTable,
} from "@tanstack/react-table";
import clsx from "clsx";

import { getAccountServicesAlertingQueryOptions } from "~/api/account/queries";
import { listReportingReportsQueryOptions } from "~/api/reporting/queries";
import type { ReportType } from "~/api/reporting/schema";
import { AlertingLogin } from "~/components/alerting/alerting-login";

export const Route = createFileRoute("/_layout/alerting")({
  validateSearch: createPaginationSearchSchema(),
  loader: async ({ context }) => {
    return context.queryClient.ensureQueryData(
      RepositoryInfoQueryOptions(context.klosetAPI, { repository: undefined })
    );
  },
  component: Component,
});

function Component() {
  const navigate = Route.useNavigate();
  const search = Route.useSearch();
  const repositoryInfo = Route.useLoaderData().item;
  const query = useQuery(getAccountServicesAlertingQueryOptions());

  if (query.data?.__type === "unauthenticated") {
    return <AlertingLogin />;
  }

  return (
    <Main
      icon="list-check"
      iconBadgeType="accent"
      title="Alerting"
      description="Follow the activity of your Kloset store."
    >
      <div className="flex flex-col gap-4">
        {query.isPending ? (
          <Spinner />
        ) : query.isError ? (
          <EmptyList
            illustrationType="sad"
            title="Unable to load alerting configuration"
            description={query.error.message}
            actions={[
              {
                children: "Retry",
                onPress: () => query.refetch(),
                variant: "secondary",
              },
            ]}
          />
        ) : (
          query.data.__type === "ok" && (
            <>
              {!query.data.configuration.enabled && (
                <Alert
                  title="Alerting is disabled"
                  variant="warning"
                  actions={[
                    <Button
                      key="settings-button"
                      variant="secondary"
                      href={{
                        to: "/settings",
                      }}
                    >
                      Go to settings
                    </Button>,
                  ]}
                >
                  The alerting service is currently disabled. You can enable it
                  in the settings.
                </Alert>
              )}

              <AlertsTable
                repository={repositoryInfo.configuration.repository_id.Value}
                page={search.page}
                perPage={search.per_page}
                onPageChange={(page: number, per_page: number) =>
                  navigate({
                    search: (prev) => ({
                      ...prev,
                      page: prev.per_page !== per_page ? 0 : page,
                      per_page,
                    }),
                  })
                }
              />
            </>
          )
        )}
      </div>
    </Main>
  );
}

type AlertsTableProps = {
  repository: string;
  page: number;
  perPage: number;
  onPageChange: (page: number, per_page: number) => void;
};

function AlertsTable({
  repository,
  page,
  perPage,
  onPageChange,
}: AlertsTableProps) {
  const query = useQuery({
    ...listReportingReportsQueryOptions({
      repository,
      page,
      perPage,
    }),
    // AlertsTable is always rendered in the context of an authenticated user,
    // but typescript doesn't know that. Let's assert it.
    select: (data) => {
      if (data.__type === "unauthenticated") {
        throw new Error("You should be authenticated to list reports");
      }
      return data;
    },
  });

  const columnHelper = createColumnHelper<AppTableFeatures, ReportType>();

  const paginationState = { pageIndex: page, pageSize: perPage };

  const table = useReactTable({
    features: appTableFeatures,
    data: query.isSuccess && query.data.__type === "ok" ? query.data.items : [],
    columns: columnHelper.columns([
      columnHelper.display({
        id: "status",
        meta: {
          cellClassName: "md:w-[20%]",
        },
        header: () => "Status",
        cell: (info) => {
          const report = info.row.original;
          const statusIcon = (
            {
              OK: "success-circle",
              WARNING: "warning-triangle",
              FAILURE: "error-circle",
              unknown: "info-circle",
            } as Record<string, IconType>
          )[report.report_task.status];
          const color = {
            OK: "text-success",
            WARNING: "text-warning",
            FAILURE: "text-error",
            unknown: "",
          }[report.report_task.status];

          return (
            <span className="flex flex-col gap-1">
              <Typography
                as="span"
                size="md"
                className={clsx("flex items-center gap-1 capitalize", color)}
              >
                <Icon type={statusIcon} size="md" />
                {info.row.original.report_task.type.HumanString}
              </Typography>

              <Typography as="span" className="text-secondary" size="sm">
                {info.row.original.report_task.duration.toHumanString() ? (
                  <>
                    Completed in{" "}
                    {info.row.original.report_task.duration.toHumanString()}
                  </>
                ) : (
                  <>Completed almost instantly</>
                )}
              </Typography>
            </span>
          );
        },
      }),
      columnHelper.accessor("report_task.name", {
        meta: {
          cellClassName: "md:w-[20%]",
        },
        header: () => "Name",
        cell: (info) => {
          const value = info.getValue();
          return (
            <Typography as="span" className="text-primary">
              {value === "@agent" || value === "@agentless" ? "" : value}
            </Typography>
          );
        },
      }),
      columnHelper.display({
        meta: {
          cellClassName: "md:w-[20%]",
        },
        id: "timeline",
        header: () => "Timeline",
        cell: (info) => {
          return (
            <div className="flex items-center gap-2">
              <div className="flex flex-col gap-1">
                <Typography
                  as="span"
                  className="text-primary whitespace-nowrap"
                >
                  {info.row.original.timestamp.relativeToNow()}
                </Typography>

                <Typography as="span" className="text-secondary" size="sm">
                  {info.row.original.timestamp.toHumanString()}
                </Typography>
              </div>
            </div>
          );
        },
      }),
      columnHelper.display({
        id: "output",
        header: () => "Output",
        cell: (info) => {
          return (
            <>
              {info.row.original.report_task.status !== "OK" && (
                <Typography as="span" className="text-primary" size="sm">
                  {info.row.original.report_task.error_message}
                </Typography>
              )}
            </>
          );
        },
      }),
    ]),
    state: {
      pagination: paginationState,
    },
    enableRowSelection: false,
    enableMultiRowSelection: false,
    getRowId: (row) => row.id,
    manualPagination: true,
    onPaginationChange: (pagination) => {
      const state =
        typeof pagination === "function"
          ? pagination(paginationState)
          : pagination;

      onPageChange(state.pageIndex, state.pageSize);
    },
  });

  return (
    <PaginatedTable<ReportType>
      query={query}
      table={table}
      ariaLabel="Reports table"
      variant="cards"
      skeleton="two-rows"
    />
  );
}
