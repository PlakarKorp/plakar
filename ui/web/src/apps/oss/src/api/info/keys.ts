export const infoKeys = {
  all: () => ["api", "plakar", "info"] as const,
  get: () => [...infoKeys.all(), "get"] as const,
};
