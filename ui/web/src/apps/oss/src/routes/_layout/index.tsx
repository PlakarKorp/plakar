import { RepositoryInfoQueryOptions } from "@plakar-ui/common/api/kloset";
import { Main } from "@plakar-ui/common/layouts";
import { createFileRoute } from "@tanstack/react-router";

import { CardActivity } from "~/components/dashboard/activity";
import { CardAlerting } from "~/components/dashboard/alerts";
import { CardLatestSnapshots } from "~/components/dashboard/latest-snapshots";
import { DashboardStats } from "~/components/dashboard/stats";

export const Route = createFileRoute("/_layout/")({
  loader: async ({ context }) => {
    return context.queryClient.ensureQueryData(
      RepositoryInfoQueryOptions(context.klosetAPI, { repository: undefined })
    );
  },
  component: Component,
});

function Component() {
  const repositoryInfo = Route.useLoaderData().item;

  return (
    <Main
      icon="house"
      iconBadgeType="accent"
      title="Your dashboard"
      description="An overview of your data."
    >
      <div className="flex flex-col gap-3 lg:gap-4">
        <DashboardStats />
        <div className="grid items-start gap-3 lg:grid-cols-[minmax(0,1fr)_360px] lg:gap-4">
          <CardLatestSnapshots />
          <div className="flex flex-col gap-3 lg:gap-4">
            <CardActivity />
            <CardAlerting
              repositoryId={repositoryInfo.configuration.repository_id.Value}
            />
          </div>
        </div>
      </div>
    </Main>
  );
}
