export const assistantKeys = {
  all: () => ["api", "plakar", "assistant"] as const,
  status: () => [...assistantKeys.all(), "status"] as const,
};
