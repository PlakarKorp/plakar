import { Typography } from "@plakar-ui/ui/components";

import { EnrichedDate } from "../../api/enrich";

/** The time of a row in a table grouped by day, with how long ago it was. */
export function CreatedCell({ date }: { date: EnrichedDate }) {
  return (
    <span
      title={date.toHumanString()}
      className="flex min-w-0 items-baseline gap-2 whitespace-nowrap"
    >
      <Typography
        as="span"
        size="sm"
        weight="semibold"
        className="tabular-nums"
      >
        {date.format("HH:mm")}
      </Typography>
      <Typography
        as="span"
        size="xs"
        className="text-tertiary truncate max-md:hidden"
      >
        {date.relativeToNow()}
      </Typography>
    </span>
  );
}
