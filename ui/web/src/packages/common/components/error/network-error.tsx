import { Alert, Button, Terminal } from "@plakar-ui/ui/components";
import { ReactNode } from "react";

import { Error } from "./error";
import { APINetworkError } from "../../api/errors";

type NetworkErrorProps = {
  url?: string;
  reset: () => void;
  error: APINetworkError;
  /** How to start the API with CORS enabled, defaults to `plakar ui`. */
  corsInstructions?: ReactNode;
};

export function NetworkError({
  url = "",
  error,
  reset,
  corsInstructions,
}: NetworkErrorProps) {
  let devaddr = "…";
  let errorUrl;

  try {
    errorUrl = new URL(error.url);
  } catch {
    // Ignore. Happens when the API is not running and the UI is run with `plakar ui`.
  }

  if (url !== "") {
    devaddr = new URL(url).host;
  }

  return (
    <Error
      illustration="sad"
      title="Network error"
      content={
        <>
          {url ? (
            <>
              Unable to establish a connection to the API{" "}
              {errorUrl && (
                <>
                  at{" "}
                  <span className="text-nowrap">
                    {errorUrl.protocol}//{errorUrl.host}
                  </span>
                </>
              )}
              . Please verify that the API service is running and accessible at
              the specified URL.
            </>
          ) : (
            <>
              Unable to establish a connection to the API. Please verify that
              the API service is still running.
            </>
          )}

          {import.meta.env.DEV && (
            <Alert
              variant="warning"
              title="Possible CORS Configuration Issue"
              className="mt-8"
            >
              <p>
                If the API is running, you might be experiencing a CORS issue.
                To enable CORS, start the API as follows:
              </p>

              {corsInstructions ?? (
                <Terminal
                  title="Run Plakar UI with CORS enabled"
                  className="mt-2"
                >
                  <Terminal.Command
                    prompt="$ "
                    command={`plakar ui -cors -addr ${devaddr} -no-spawn -no-auth`}
                  />
                </Terminal>
              )}
            </Alert>
          )}
        </>
      }
      action={<Button onPress={reset}>Reload</Button>}
    />
  );
}
