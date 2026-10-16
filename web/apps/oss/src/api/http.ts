import {
  APINetworkError,
  APIResponseBadRequest,
  APIResponseForbiddenError,
  APIResponseInternalError,
  APIResponseNotFoundError,
  APIResponseNotImplementedError,
  APIResponseUnauthorizedError,
} from "@plakar-ui/common/api/errors";
import { getLocalStorage } from "@plakar-ui/common/utils";
import { QueryClient } from "@tanstack/react-query";

import { ossProxyKeys } from "./keys";

export const PLAKAR_URL = import.meta.env.VITE_PLAKAR_URL ?? "";

export async function CallApi(input: RequestInfo | URL, init?: RequestInit) {
  const url = `${PLAKAR_URL}${input}`;
  const method = init?.method ?? "GET";

  const storageToken = getLocalStorage<string>("plakar_token");

  // Ensure the token is a valid alphanumeric string with dashes.
  const token =
    storageToken && /^[A-Za-z0-9-]+$/.test(storageToken) ? storageToken : "";

  if (token) {
    init = {
      ...init,
      headers: {
        ...init?.headers,
        Authorization: `Bearer ${token}`,
      },
    };
  }

  try {
    const resp = await fetch(url, init);

    switch (resp.status) {
      case 400: {
        const responsePayload = await resp.text();
        throw new APIResponseBadRequest(method, url, responsePayload);
      }
      case 401:
        throw new APIResponseUnauthorizedError(method, url);
      case 403:
        throw new APIResponseForbiddenError(method, url);
      case 404:
        throw new APIResponseNotFoundError(method, url);
      case 500:
        throw new APIResponseInternalError(method, url);
      case 501:
        throw new APIResponseNotImplementedError(method, url);
    }
    return resp;
  } catch (error) {
    if (error instanceof TypeError && error.message.includes("NetworkError")) {
      throw new APINetworkError(method, url, "Network error", error);
    }
    throw error;
  }
}

/**
 * Catchify calls the promise and returns specific error types if they occur.
 * @param fn The function to call
 * @param only The types of errors to catch and return
 *
 * Example:
 * ```ts
 * const { data, error } = await catchify(() => CallApi("/api/endpoint"), APIResponseNotFoundError);
 * ```
 */
export async function catchify<
  T,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  U extends (new (...args: any[]) => unknown)[],
>(fn: () => Promise<T>, ...only: [...U]) {
  try {
    return { data: await fn(), error: null };
  } catch (error) {
    if (!only.some((ErrorType) => error instanceof ErrorType)) {
      throw error;
    }
    return { data: null, error: error as (typeof only)[number] };
  }
}

// Invalidate all the queries of the proxified API, which all require to be authenticated.
export async function InvalidateAPIProxifiedQueries(queryClient: QueryClient) {
  return queryClient.invalidateQueries({
    queryKey: ossProxyKeys.all(),
  });
}
