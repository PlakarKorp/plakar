import {
  IconType,
  Id,
  TagOverflowList,
  Typography,
} from "@plakar-ui/ui/components";

import { ConnectorCard } from "../connector-card/connector-card";

type SnapshotCardProps = {
  id: string;
  sources: {
    type: string;
    origin: string;
    directory: string;
    icon: IconType;
  }[];
  size?: string;
  itemsLength?: number;
  tags?: string[];
  dateModified?: string;
  timeSince?: string;
};

export function SnapshotCard({
  id,
  sources = [],
  size,
  itemsLength = 0,
  tags = [],
  dateModified,
  timeSince,
}: SnapshotCardProps) {
  return (
    <div className="flex w-full items-center gap-6 max-sm:flex-wrap max-sm:items-start max-sm:gap-x-3 max-sm:gap-y-1">
      <span className="flex shrink-0 items-center">
        <Id
          value={id}
          className="text-secondary"
          size="sm"
          withCopyButton={false}
        />
      </span>

      <span className="min-w-0 grow space-y-0.5 truncate">
        {sources && (
          <>
            <ConnectorCard.Connector
              type="source"
              name={sources[0].origin}
              integrationIcon={sources[0].icon}
              size="sm"
            />
            <Typography size="sm" className="text-secondary">
              {sources[0].directory}
            </Typography>
          </>
        )}
      </span>

      <span className="min-w-0">
        <TagOverflowList
          items={tags}
          variant="blue_borderless"
          radius="sqrt"
          size="sm"
          maxRows={2}
        />
      </span>

      <span className="flex shrink-0 flex-col items-end gap-0.5 whitespace-nowrap max-sm:hidden">
        <Typography size="sm" className="text-secondary">
          {dateModified}
        </Typography>
        <Typography size="sm" className="text-tertiary">
          {timeSince}
        </Typography>
      </span>

      <span className="flex shrink-0 flex-col items-end gap-0.5 whitespace-nowrap max-sm:hidden">
        <Typography size="sm" className="text-secondary">
          {size}
        </Typography>
        <Typography size="sm" className="text-tertiary">
          {itemsLength} {itemsLength > 1 ? "items" : "item"}
        </Typography>
      </span>
    </div>
  );
}
