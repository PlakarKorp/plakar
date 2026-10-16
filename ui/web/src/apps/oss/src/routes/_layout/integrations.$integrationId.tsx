import { RepositoryInfoQueryOptions } from "@plakar-ui/common/api/kloset";
import { createFileRoute } from "@tanstack/react-router";

import { getIntegrationQueryOptions } from "~/api/integrations/queries";

export const Route = createFileRoute("/_layout/integrations/$integrationId")({
  loader: async ({ context, params }) => {
    context.queryClient.prefetchQuery(
      getIntegrationQueryOptions({ integration: params.integrationId })
    );

    const repositoryInfo = await context.queryClient.ensureQueryData(
      RepositoryInfoQueryOptions(context.klosetAPI, { repository: undefined })
    );

    return {
      repositoryInfo,
    };
  },
});
