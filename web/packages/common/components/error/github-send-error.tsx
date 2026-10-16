import { Button } from "@plakar-ui/ui/components";

type GithubSendErrorProps = {
  githubParams: URLSearchParams;
};

export function GithubSendError({ githubParams }: GithubSendErrorProps) {
  return (
    <Button
      variant="secondary"
      leftIcon="github"
      href={`https://github.com/PlakarKorp/plakar/issues/new?${githubParams.toString()}`}
    >
      Send bug report
    </Button>
  );
}
