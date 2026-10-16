import { Button } from "@plakar-ui/ui/components";

import { Error } from "./error";
import { GithubSendError } from "./github-send-error";

type DisplayInternalErrorProps = {
  reset: () => void;
};

export function InternalError({ reset }: DisplayInternalErrorProps) {
  const githubParams = new URLSearchParams({
    title: "API returned HTTP/500",
    body: "The API returned a HTTP/500 Internal Server Error response. To reproduce the issue, …",
  });

  return (
    <Error
      illustration="sad"
      title="Internal error"
      icon="exclamation-circle"
      content={
        <>
          The API encountered an unexpected HTTP/500 Internal Server Error. This
          shouldn't happen, and it's something we need to fix. We'd greatly
          appreciate it if you could submit a bug report with details to help us
          reproduce and resolve the issue. Thank you!
        </>
      }
      action={[
        <GithubSendError key="github-button" githubParams={githubParams} />,
        <Button key="reload-button" onPress={reset}>
          Reload
        </Button>,
      ]}
    />
  );
}
