import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useEffect } from "react";

import { InvalidateAPIProxifiedQueries } from "~/api/http";
import { getApiInfoQueryOptions } from "~/api/info/queries";

// Call /api/info at a regular interval to check if the API is authenticated,
// and update the local storage accordingly.
export function useApi(refetchInterval = 1000 * 10) {
  const query = useQuery({
    ...getApiInfoQueryOptions(),
    refetchInterval,
  });
  const queryClient = useQueryClient();
  const router = useRouter();

  // Invalidate the API queries when the authentication status changes.
  useEffect(() => {
    router.invalidate();
    InvalidateAPIProxifiedQueries(queryClient);
  }, [queryClient, router, query.data?.authenticated]);

  if (query.isError) {
    return { status: "error" } as const;
  }
  if (query.isPending) {
    return { status: "loading" } as const;
  }

  return {
    status: "ok",
    isAuthenticated: query.data.authenticated,
  };
}

type useWaitLoginStatusThenRedirectParams = {
  status: "logged_in" | "logged_out";
  returnUrl?: string;
};

export function useWaitLoginStatusThenRedirect({
  status,
  returnUrl,
}: useWaitLoginStatusThenRedirectParams) {
  const res = useApi(1000);
  const router = useRouter();

  useEffect(() => {
    if (res.status !== "ok") {
      return;
    }
    if (
      (status === "logged_in" && res.isAuthenticated) ||
      (status === "logged_out" && !res.isAuthenticated)
    ) {
      router.navigate({
        to: returnUrl || "/",
      });
    }
  }, [router, res, status, returnUrl]);
}
