import { ListIntegrationsParams } from "./schema";

export const integrationKeys = {
  all: () => ["api", "api.plakar.io", "integrations"] as const,
  lists: () => [...integrationKeys.all(), "list"] as const,
  list: (params: ListIntegrationsParams) =>
    [...integrationKeys.lists(), params] as const,
  detail: (id: string) => [...integrationKeys.all(), "get", id] as const,
};
