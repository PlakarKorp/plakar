import { Icon, Id, TextHighlight, Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { PropsWithChildren } from "react";

import { getIntegrationIcon } from "../../api/enrich";

function ConfigurationCardRoot({ children }: PropsWithChildren) {
  return (
    <div className="bg-primary hover:bg-primary-hover border-primary flex items-center gap-4 rounded-lg border px-4 py-2">
      {children}
    </div>
  );
}

export type ConfigurationCardContentProps = {
  id?: string;
  name: string;
  integration: string;
  location?: string;
  withCopyButton?: boolean;
  maxNameWidth?: number;
  search?: string;
};

function ConfigurationCardContent({
  id,
  name,
  integration,
  location,
  withCopyButton = false,
  maxNameWidth,
  search,
}: ConfigurationCardContentProps) {
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div
        className={clsx(maxNameWidth ? "flex flex-col" : "grid grid-cols-1")}
      >
        <div className="flex items-center gap-2">
          <Icon type={getIntegrationIcon(integration)} size="md" />
          <Typography
            as="span"
            size="md"
            weight="medium"
            className="text-primary truncate"
            style={maxNameWidth ? { maxWidth: maxNameWidth } : undefined}
          >
            <TextHighlight text={name} search={search} />
          </Typography>
        </div>
        {location && (
          <Typography
            as="div"
            size="sm"
            className="text-tertiary truncate"
            monospace
          >
            {location}
          </Typography>
        )}
      </div>
      {id && (
        <Id
          value={id}
          size="sm"
          className="text-secondary h-5"
          withCopyButton={withCopyButton}
        />
      )}
    </div>
  );
}

export const ConfigurationCard = Object.assign(ConfigurationCardRoot, {
  Content: ConfigurationCardContent,
});
