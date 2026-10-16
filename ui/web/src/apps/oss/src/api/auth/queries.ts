import { queryOptions } from "@tanstack/react-query";

import { pollLogin, PollLoginParams } from "./api";
import { authKeys } from "./keys";

export const pollLoginQueryOptions = (params: PollLoginParams) =>
  queryOptions({
    queryKey: authKeys.poll(params),
    queryFn: () => pollLogin(params),
    // Every poll is a question to api.plakar.io, never a cached answer.
    gcTime: 0,
    retry: false,
  });
