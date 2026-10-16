import { Button, Card, Feed, EmptyList } from "@plakar-ui/ui/components";
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
  const status = {
    OK: "success",
    WARNING: "warning",
    FAILURE: "error",
    unknown: "default",
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
        <Feed loading={query.isPending}>
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
              query.data.items.map((report) => (
                <Feed.Item
                  key={report.id}
                  icon={icons[report.report_task.status]}
                  status={status[report.report_task.status]}
                  title={report.report_task.type.HumanString}
                  description={report.report_task.error_message || "Success"}
                  alt={report.timestamp.relativeToNow()}
                />
              ))
            ))}
        </Feed>
      </Card.Content>
    </Card>
  );
}
