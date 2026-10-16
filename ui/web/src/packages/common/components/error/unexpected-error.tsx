import { Button } from "@plakar-ui/ui/components";

import { Error } from "./error";
import { GithubSendError } from "./github-send-error";

type UnexpectedErrorComponentProps = {
  error: Error;
  reset: () => void;
};

export function UnexpectedError({
  error,
  reset,
}: UnexpectedErrorComponentProps) {
  const githubParams = new URLSearchParams({
    title: "Unexpected error",
    body: `\`\`\`\n${error.name}: ${error.message}\n\n${error.stack}\`\`\``,
  });

  return (
    <Error
      illustration="sad"
      title="Oops, something went wrong (and it's our fault)"
      content={
        <>
          You should never have seen this page: we failed to catch an error and
          this fallback error is displayed. We would appreciate if you could
          fill a bug report with the error message below, and possibly
          everything we need to reproduce the issue.
        </>
      }
      action={[
        <GithubSendError key="github-button" githubParams={githubParams} />,
        <Button key="reload-button" onPress={reset}>
          Reload
        </Button>,
      ]}
      error={error}
    />
  );
}
