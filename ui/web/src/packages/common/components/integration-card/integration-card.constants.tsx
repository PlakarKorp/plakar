export const IntegrationInstallationStatuses = [
  "installed",
  "not-installed",
  "built-in",
] as const;

export type IntegrationInstallationStatus =
  (typeof IntegrationInstallationStatuses)[number];

export const INTEGRATION_TYPE_TAGS = [
  { key: "storage", label: "Storage" },
  { key: "source", label: "Source" },
  { key: "destination", label: "Destination" },
  { key: "secret_provider", label: "Secret provider" },
  { key: "inventory", label: "Inventory" },
] as const;

export type IntegrationTypeKey = (typeof INTEGRATION_TYPE_TAGS)[number]["key"];

export type IntegrationTypeFlags = Partial<Record<IntegrationTypeKey, boolean>>;
