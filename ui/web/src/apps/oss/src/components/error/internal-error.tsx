import {
  APINetworkError,
  APIResponseBadRequest,
  APIResponseForbiddenError,
  APIResponseInternalError,
  APIResponseNotFoundError,
  APIResponseUnauthorizedError,
} from "@plakar-ui/common/api/errors";
import {
  BadRequestError,
  Error,
  InternalError,
  NetworkError,
  NotFoundError,
  UnexpectedError,
} from "@plakar-ui/common/components";
import { useQueryClient } from "@tanstack/react-query";
import { type ErrorComponentProps, useRouter } from "@tanstack/react-router";

import { TokenForm } from "./token-form";

import { PLAKAR_URL } from "~/api/http";

export default function DefaultError({ error, reset }: ErrorComponentProps) {
  const router = useRouter();
  const queryClient = useQueryClient();

  const retry = async () => {
    await router.invalidate();
    queryClient.removeQueries();
    reset();
  };

  if (error instanceof APIResponseNotFoundError) {
    return <NotFoundError href={{ to: "/" }} />;
  }

  if (error instanceof APINetworkError) {
    return <NetworkError url={PLAKAR_URL} error={error} reset={retry} />;
  }

  if (error instanceof APIResponseInternalError) {
    return <InternalError reset={retry} />;
  }

  if (error instanceof APIResponseBadRequest) {
    return <BadRequestError error={error} reset={retry} />;
  }

  if (
    error instanceof APIResponseUnauthorizedError ||
    error instanceof APIResponseForbiddenError
  ) {
    return <DisplayPermissionError error={error} />;
  }

  const unexpected =
    error instanceof globalThis.Error
      ? error
      : new globalThis.Error(String(error));
  return <UnexpectedError error={unexpected} reset={retry} />;
}

type PermissionErrorProps = {
  error: APIResponseUnauthorizedError | APIResponseForbiddenError;
};

function DisplayPermissionError({ error }: PermissionErrorProps) {
  return (
    <Error
      illustration="walking"
      title="Invalid token"
      icon="md-vpn-key-off"
      content={
        <>
          The Plakar API returned a{" "}
          <span className="text-nowrap">
            HTTP/{error.code} {error.message}
          </span>{" "}
          response. It usually means that the token stored in the local storage
          of your browser is invalid or missing. Use the form below to enter a
          valid token.
        </>
      }
      action={<TokenForm />}
    />
  );
}
