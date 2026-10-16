// Root key shared by all queries proxied through the plakar backend to api.plakar.io.
export const ossProxyKeys = {
  all: () => ["api", "api.plakar.io"] as const,
};
