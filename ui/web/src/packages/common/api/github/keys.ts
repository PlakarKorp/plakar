export const githubKeys = {
  all: () => ["api", "github"] as const,
  repoStars: (repo: string) =>
    [...githubKeys.all(), "repo_stars", repo] as const,
};
