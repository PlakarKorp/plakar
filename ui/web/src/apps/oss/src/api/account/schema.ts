import { Date } from "@plakar-ui/common/api/enrich";
import { z } from "zod";

export const AccountMeSchema = z.object({
  user_id: z.string(),
  organization_id: z.string(),
  auth_provider: z.string(),
  name: z.string(),
  avatar_url: z.string(),
});

const notificationLevels = ["info", "warning", "error", "critical"] as const;

type NotificationKnownLevel = (typeof notificationLevels)[number];
type NotificationLevel = NotificationKnownLevel | "unknown";

const notificationStatuses = ["unread", "read", "pinned", "discarded"] as const;

export type NotificationKnownStatus = (typeof notificationStatuses)[number];
type NotificationStatus = NotificationKnownStatus | "unknown";

const NotificationSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  level: z.string().transform((value): NotificationLevel => {
    return notificationLevels.includes(value as NotificationKnownLevel)
      ? (value as NotificationKnownLevel)
      : "unknown";
  }),
  type: z.string(),
  status: z.string().transform((value): NotificationStatus => {
    return notificationStatuses.includes(value as NotificationKnownStatus)
      ? (value as NotificationKnownStatus)
      : "unknown";
  }),
  code: z.string(),
  message: z.string(),
  details: z.any(),
  created_at: z.string().transform(Date),
  updated_at: z.string().transform(Date),
  flags: z.number().transform((flags) => ({
    isBanner: !!(flags & 0b0001),
  })),
});

export const NotificationsListSchema = z.object({
  total: z.number(),
  items: z.array(NotificationSchema),
});

export const AccountServicesAlertingSchema = z.object({
  enabled: z.boolean(),
  email_report: z.boolean(),
});

export type AlertingSettings = z.infer<typeof AccountServicesAlertingSchema>;
