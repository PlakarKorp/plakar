import { Card, DescriptionList } from "@plakar-ui/ui/components";

import {
  INTEGRATION_TYPE_TAGS,
  IntegrationTypeFlags,
} from "./integration-card.constants";

type IntegrationDetailsProps = {
  installedVersion?: string;
  latestVersion: string;
  stage: string;
  types: IntegrationTypeFlags;
};

export function IntegrationDetails({
  installedVersion,
  latestVersion,
  stage,
  types,
}: IntegrationDetailsProps) {
  const provides = INTEGRATION_TYPE_TAGS.filter(({ key }) => types[key])
    .map(({ label }) => label)
    .join(", ");

  return (
    <Card>
      <Card.Header title="Details" />
      <Card.Content>
        <DescriptionList size="sm">
          {installedVersion && (
            <DescriptionList.Item
              title="Installed"
              description={installedVersion}
            />
          )}
          <DescriptionList.Item title="Latest" description={latestVersion} />
          <DescriptionList.Item
            title="Stage"
            description={<span className="capitalize">{stage}</span>}
          />
          {provides && (
            <DescriptionList.Item
              title="Provides"
              description={provides}
              truncate={false}
            />
          )}
        </DescriptionList>
      </Card.Content>
    </Card>
  );
}
