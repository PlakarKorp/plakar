import { Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";

import { JOB_STATUSES } from "./job-status.constants";

type Status = (typeof JOB_STATUSES)[number];

type JobStatusProps = {
  status: Status;
};

export function JobStatus({ status }: JobStatusProps) {
  return (
    <Typography
      as="span"
      size="sm"
      className={clsx(
        "flex items-center gap-1.5 capitalize",
        {
          succeeded: "text-success",
          failed: "text-error",
          running: "text-accent animate-pulse",
          queued: "text-accent animate-pulse",
          canceled: "text-warning",
        }[status]
      )}
      weight="medium"
    >
      <span
        className={clsx(
          "h-2.5 w-2.5 rounded-full",
          {
            succeeded: "bg-success",
            failed: "bg-error",
            running: "bg-accent animate-pulse",
            queued: "bg-accent animate-pulse",
            canceled: "bg-warning",
          }[status]
        )}
      />
      {status}
    </Typography>
  );
}
