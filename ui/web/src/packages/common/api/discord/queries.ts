import { queryOptions } from "@tanstack/react-query";

import { discordKeys } from "./keys";

export const DiscordServerMembersQueryOptions = (invite: string) =>
  queryOptions({
    queryKey: discordKeys.serverStats(invite),
    queryFn: async () => {
      const response = await fetch(`https://discord.com/api/invites/${invite}`);
      const data = await response.json();
      const count = data?.profile?.member_count;

      if (count === undefined) {
        throw new Error("Invalid response from Discord API");
      }
      return count as number;
    },

    // Be gentle with the Discord API, and avoid refetching too often.
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
