import {
  Breadcrumbs,
  IntegrationDetails,
  IntegrationDocumentation,
} from "@plakar-ui/common/components";
import { Main } from "@plakar-ui/common/layouts";
import { useSuspenseQuery } from "@tanstack/react-query";
import { createLazyFileRoute } from "@tanstack/react-router";

import { getIntegrationQueryOptions } from "~/api/integrations/queries";
import type { Integration } from "~/api/integrations/schema";
import { IntegrationHeaderAdapter } from "~/components/integration-card/integration-adapters";
import { IntegrationAlertLoginRequired } from "~/components/integration-card/integration-alert-login-required";
import { useAuthentication } from "~/hooks/use-authentication";

export const Route = createLazyFileRoute(
  "/_layout/integrations/$integrationId"
)({
  component: RouteComponent,
});

type PageCrumbsProps = {
  integration: Integration;
};

function PageCrumbs({ integration }: PageCrumbsProps) {
  return (
    <Breadcrumbs>
      <Breadcrumbs.Crumb key="integrations" to="/integrations">
        Integrations
      </Breadcrumbs.Crumb>
      <Breadcrumbs.Crumb
        key="page"
        to="/integrations/$integrationId"
        params={{ integrationId: integration.id }}
      >
        {integration.display_name}
      </Breadcrumbs.Crumb>
    </Breadcrumbs>
  );
}

function RouteComponent() {
  const params = Route.useParams();
  const query = useSuspenseQuery(
    getIntegrationQueryOptions({ integration: params.integrationId })
  );
  const { repositoryInfo } = Route.useLoaderData();

  const integration = query.data;
  const { isAuthenticated } = useAuthentication();

  return (
    <Main breadcrumb={<PageCrumbs integration={integration} />}>
      <div className="flex flex-col gap-6">
        <IntegrationHeaderAdapter
          integration={integration}
          installDisabled={!isAuthenticated}
        />
        {!isAuthenticated && (
          <IntegrationAlertLoginRequired
            name={integration.id}
            version={integration.latest_version}
            os={repositoryInfo.item.os}
            arch={repositoryInfo.item.arch}
          />
        )}
        <IntegrationDocumentation
          documentation={integration.documentation}
          aside={
            <IntegrationDetails
              installedVersion={integration.installation.version}
              latestVersion={integration.latest_version}
              stage={integration.stage}
              types={integration.types}
            />
          }
        />
      </div>
    </Main>
  );
}
