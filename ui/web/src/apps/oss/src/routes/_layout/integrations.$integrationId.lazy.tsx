import {
  Breadcrumbs,
  IntegrationDocumentation,
} from "@plakar-ui/common/components";
import { Main } from "@plakar-ui/common/layouts";
import { useSuspenseQuery } from "@tanstack/react-query";
import { createLazyFileRoute } from "@tanstack/react-router";

import { getIntegrationQueryOptions } from "~/api/integrations/queries";
import type { Integration } from "~/api/integrations/schema";
import { IntegrationAlertLoginRequired } from "~/components/integration-card/integration-alert-login-required";
import { IntegrationCardAdapter } from "~/components/integration-card/integration-card-adapter";
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
      <IntegrationDocumentation
        documentation={integration.documentation}
        aside={
          <>
            {!isAuthenticated && (
              <IntegrationAlertLoginRequired
                name={query.data.id}
                version={query.data.latest_version}
                os={repositoryInfo.item.os}
                arch={repositoryInfo.item.arch}
              />
            )}

            <IntegrationCardAdapter
              integration={integration}
              installDisabled={!isAuthenticated}
            />
          </>
        }
      />
    </Main>
  );
}
