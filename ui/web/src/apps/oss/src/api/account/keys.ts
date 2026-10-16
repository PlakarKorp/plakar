export const accountKeys = {
  all: () => ["api", "api.plakar.io", "account"] as const,
  me: () => [...accountKeys.all(), "me"] as const,
  notifications: () => [...accountKeys.all(), "notifications"] as const,
  notificationsList: (repository: string, status?: string) =>
    [...accountKeys.notifications(), repository, status] as const,
  servicesAlerting: () =>
    [...accountKeys.all(), "services", "alerting"] as const,
};
