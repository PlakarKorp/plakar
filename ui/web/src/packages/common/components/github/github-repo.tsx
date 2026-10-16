import { Icon, Skeleton, Typography } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import clsx from "clsx";

import { GithubRepoStarsQueryOptions } from "../../api/github/queries";

type GithubRepoProps = {
  repo: string;
  variant?: "default" | "accent";
};

export function GithubRepo({ repo, variant = "default" }: GithubRepoProps) {
  const { data, isPending, isSuccess } = useQuery(
    GithubRepoStarsQueryOptions(repo)
  );

  return (
    <a
      target="_blank"
      href={`https://github.com/${repo}`}
      className={clsx(
        "flex min-w-[60px] gap-1 rounded-sm border px-2 py-1",
        variant === "default"
          ? "border-primary bg-secondary text-secondary hover:bg-secondary-hover"
          : "border-text-on-brand/50 text-on-brand"
      )}
    >
      <Icon type="github" size="md" className="shrink-0" />

      {isPending ? (
        <Skeleton variant="text" size="xs" width="100%" />
      ) : isSuccess ? (
        <Typography variant="body" size="xs" monospace>
          {data}
        </Typography>
      ) : (
        <Typography variant="body" size="xs">
          GitHub
        </Typography>
      )}
    </a>
  );
}
