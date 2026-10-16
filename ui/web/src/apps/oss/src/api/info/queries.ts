import { queryOptions } from "@tanstack/react-query";

import { getApiInfo } from "./api";
import { infoKeys } from "./keys";

export const getApiInfoQueryOptions = () =>
  queryOptions({
    queryKey: infoKeys.get(),
    queryFn: getApiInfo,
  });
