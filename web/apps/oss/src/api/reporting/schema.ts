import { Date, Duration } from "@plakar-ui/common/api/enrich";
import { z } from "zod";

const TaskSchema = z.object({
  type: z
    .union([
      z.literal("maintenance"),
      z.literal("backup"),
      z.literal("check"),
      z.literal("restore"),
      z.literal("sync"),
      z.literal("rm"),
      z.literal("unknown"),
    ])
    .catch("unknown")
    .transform((s) => ({
      Value: s,
      HumanString: {
        maintenance: "maintenance",
        backup: "backup",
        check: "check",
        restore: "restore",
        sync: "sync",
        rm: "remove",
        unknown: "unknown",
      }[s],
    })),
  name: z.string(),
  start_time: z.string().transform(Date),
  duration: z.number().transform(Duration),
  status: z
    .union([
      z.literal("OK"),
      z.literal("FAILURE"),
      z.literal("WARNING"),
      z.literal("unknown"),
    ])
    .catch("unknown"),
  error_code: z.number(),
  error_message: z.string(),
});

const RepositorySchema = z.object({
  name: z.string(),
});

export const ReportSchema = z.object({
  id: z.string(),
  timestamp: z.string().transform(Date),
  report_task: TaskSchema,
  report_repository: RepositorySchema,
});

export type ReportType = z.infer<typeof ReportSchema>;

export const ReportsListSchema = z.object({
  total: z.number(),
  items: z.array(ReportSchema),
});
