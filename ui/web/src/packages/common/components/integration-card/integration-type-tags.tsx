import { Tag } from "@plakar-ui/ui/components";

import {
  INTEGRATION_TYPE_TAGS,
  IntegrationTypeFlags,
} from "./integration-card.constants";

export function IntegrationTypeTags({
  types,
}: {
  types: IntegrationTypeFlags;
}) {
  const typeTags = INTEGRATION_TYPE_TAGS.filter(({ key }) => types[key]);

  if (typeTags.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {typeTags.map(({ key, label }) => (
        <Tag key={key} size="sm" radius="sqrt">
          {label}
        </Tag>
      ))}
    </div>
  );
}
