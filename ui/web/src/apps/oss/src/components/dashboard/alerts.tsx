import {
  Button,
  Card,
  EmptyList,
  Icon,
  Skeleton,
  Typography,
} from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { ComponentProps } from "react";

import { listReportingReportsQueryOptions } from "~/api/reporting/queries";

type CardReportsProps = {
  repositoryId: string;
};

export function CardAlerting({ repositoryId }: CardReportsProps) {
  const query = useQuery(
    listReportingReportsQueryOptions({
      repository: repositoryId,
      page: 0,
      perPage: 5,
    })
  );

  const handleRetry = () => {
    query.refetch();
  };

  const icons = {
    OK: "check",
    WARNING: "exclamation",
    FAILURE: "close",
    unknown: "cloud",
  } as const;
  const iconClasses = {
    OK: "text-success",
    WARNING: "text-warning",
    FAILURE: "text-error",
    unknown: "text-tertiary",
  } as const;
  const statusLabels = {
    OK: "Success",
    WARNING: "Warning",
    FAILURE: "Failure",
    unknown: "Unknown status",
  } as const;
  const isAuthenticated = query.data?.__type !== "unauthenticated";
  const button = !isAuthenticated
    ? ({
        children: "Sign in to view",
        href: {
          to: "/login",
        },
        rightIcon: "arrow-up-right",
      } as ComponentProps<typeof Button>)
    : ({
        children: "View all",
        href: {
          to: "/alerting",
        },
        rightIcon: "arrow-up-right",
      } as ComponentProps<typeof Button>);

  return (
    <Card>
      <Card.Header title="Latest alerting reports" buttonProps={button} />

      <Card.Content>
        <div className="flex flex-col">
          {query.isPending &&
            [0, 1, 2, 3, 4].map((row) => (
              <Skeleton key={row} variant="text" size="sm" className="my-2" />
            ))}

          {query.error && (
            <EmptyList
              illustrationType="sad"
              title="Unable to load the latest alerting reports"
              description={query.error.message}
              actions={[
                {
                  children: "Retry",
                  onPress: handleRetry,
                  variant: "secondary",
                },
              ]}
            />
          )}

          {query.isSuccess &&
            (query.data.__type === "unauthenticated" ? (
              <EmptyList
                illustrationType="emperor"
                title="Sign in to enable Alerts"
                description="Get notified when backups, restores, checks, or maintenance tasks complete or fail."
                actions={[
                  {
                    children: "Sign in",
                    href: {
                      to: "/login",
                    },
                    variant: "primary",
                  },
                ]}
              />
            ) : query.data.items.length === 0 ? (
              <EmptyList
                illustrationType="waving"
                title="You don't have any alerting reports yet"
                description="Enable alerting to start receiving reports automatically
                  whenever you run backups, restores, checks, or maintenance
                  tasks."
              />
            ) : (
              <ol>
                {query.data.items.map((report) => (
                  <li
                    key={report.id}
                    className="flex h-8 min-w-0 items-center gap-2.5"
                  >
                    <Icon
                      type={icons[report.report_task.status]}
                      size="sm"
                      className={iconClasses[report.report_task.status]}
                    />
                    <Typography
                      as="span"
                      size="sm"
                      className="text-primary shrink-0 capitalize"
                    >
                      {report.report_task.type.HumanString}
                    </Typography>
                    <Typography
                      as="span"
                      size="sm"
                      className="text-tertiary min-w-0 flex-1 truncate"
                    >
                      {report.report_task.error_message ||
                        statusLabels[report.report_task.status]}
                    </Typography>
                    <Typography
                      as="span"
                      size="xs"
                      className="text-tertiary shrink-0"
                    >
                      {report.timestamp.relativeToNow()}
                    </Typography>
                  </li>
                ))}
              </ol>
            ))}
        </div>
      </Card.Content>
    </Card>
  );
}
