import { queryOptions } from "@tanstack/react-query";

import { githubKeys } from "./keys";

export const GithubRepoStarsQueryOptions = (repo: string) =>
  queryOptions({
    queryKey: githubKeys.repoStars(repo),
    queryFn: async () => {
      const response = await fetch(`https://api.github.com/repos/${repo}`);
      const data = await response.json();

      if (data.stargazers_count === undefined) {
        throw new Error("Invalid response from GitHub API");
      }

      return data.stargazers_count as number;
    },

    // Be gentle with the GitHub API, and avoid refetching too often.
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
