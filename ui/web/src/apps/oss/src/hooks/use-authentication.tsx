import { useQuery } from "@tanstack/react-query";

import { getAccountMeQueryOptions } from "~/api/account/queries";

export function useAuthentication() {
  const query = useQuery(getAccountMeQueryOptions());
  const isAuthenticated = query.data !== null;

  return { isAuthenticated };
}
