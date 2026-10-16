import { RepositoryInfoQueryOptions } from "@plakar-ui/common/api/kloset";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import { getApiInfoQueryOptions } from "~/api/info/queries";
import { Layout } from "~/layouts/layout";

export const Route = createFileRoute("/_layout")({
  loader: async ({ context }) => {
    context.queryClient.prefetchQuery(
      RepositoryInfoQueryOptions(context.klosetAPI, { repository: undefined })
    );
    return context.queryClient.ensureQueryData(getApiInfoQueryOptions());
  },
  component: LayoutRoute,
});

function LayoutRoute() {
  const apiInfo = Route.useLoaderData();

  return (
    <Layout repository={apiInfo.repository_id} version={apiInfo.version}>
      <Outlet />
    </Layout>
  );
}
