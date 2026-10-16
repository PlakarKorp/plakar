export const discordKeys = {
  all: () => ["api", "discord"] as const,
  serverStats: (invite: string) =>
    [...discordKeys.all(), "server_stats", invite] as const,
};
