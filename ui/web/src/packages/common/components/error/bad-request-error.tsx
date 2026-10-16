import { Button } from "@plakar-ui/ui/components";

import { Error } from "./error";
import { APIResponseBadRequest } from "../../api/errors";

type DisplayBadRequestErrorProps = {
  reset: () => void;
  error: APIResponseBadRequest;
};

export function BadRequestError({ error, reset }: DisplayBadRequestErrorProps) {
  let prettyError = error.responsePayload;

  try {
    prettyError = JSON.stringify(JSON.parse(error.responsePayload), null, 2);
  } catch {
    // Do nothing, keep the original error.
  }

  return (
    <Error
      illustration="sad"
      icon="error-circle"
      title="Bad Request"
      content={
        <>
          The API returned a HTTP/400 Bad Request response. This usually means
          that the request was malformed or invalid. If you did not craft the
          URL manually, please report this issue to us.
        </>
      }
      error={prettyError}
      action={<Button onPress={reset}>Reload</Button>}
    />
  );
}
