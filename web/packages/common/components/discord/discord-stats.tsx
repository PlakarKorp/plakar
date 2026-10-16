import { Icon, Skeleton, Typography } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import clsx from "clsx";

import { DiscordServerMembersQueryOptions } from "../../api/discord/queries";

type DiscordStatsProps = {
  invite: string;
  variant?: "default" | "accent";
};

export function DiscordStats({
  invite,
  variant = "default",
}: DiscordStatsProps) {
  const { data, isPending, isSuccess } = useQuery(
    DiscordServerMembersQueryOptions(invite)
  );

  return (
    <a
      target="_blank"
      href={`https://discord.gg/${invite}`}
      className={clsx(
        "flex min-w-[60px] gap-1 rounded-sm border px-2 py-1",
        variant === "default"
          ? "border-primary bg-secondary text-secondary hover:bg-secondary-hover"
          : "border-text-on-brand/50 text-on-brand"
      )}
    >
      <Icon type="discord" size="md" className="shrink-0" />

      {isPending ? (
        <Skeleton variant="text" size="xs" width="100%" />
      ) : isSuccess ? (
        <Typography variant="body" size="xs" monospace>
          {data}
        </Typography>
      ) : (
        <Typography variant="body" size="xs">
          Discord
        </Typography>
      )}
    </a>
  );
}
