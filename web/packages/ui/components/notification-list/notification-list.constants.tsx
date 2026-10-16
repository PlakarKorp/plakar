export const NotificationSevrities = ["default", "warning", "error"] as const;

export type NotificationSeverity = (typeof NotificationSevrities)[number];
