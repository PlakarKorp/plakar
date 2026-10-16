import { RepositoryInfoQueryOptions } from "@plakar-ui/common/api/kloset";
import { Main } from "@plakar-ui/common/layouts";
import { createFileRoute } from "@tanstack/react-router";

import { CardAlerting } from "~/components/dashboard/alerts";
import { CardCharts } from "~/components/dashboard/charts";
import { CardLatestSnapshots } from "~/components/dashboard/latest-snapshots";

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
      <div className="grid gap-3 lg:grid-cols-[2fr_1fr] lg:gap-4">
        <div className="flex flex-col gap-3 lg:gap-4">
          <CardCharts />
          <CardLatestSnapshots />
        </div>
        <div>
          <CardAlerting
            repositoryId={repositoryInfo.configuration.repository_id.Value}
          />
        </div>
      </div>
    </Main>
  );
}
