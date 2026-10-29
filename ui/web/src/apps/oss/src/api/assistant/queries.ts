import { queryOptions } from "@tanstack/react-query";

import { getAssistantStatus } from "./api";
import { assistantKeys } from "./keys";

// Whether the assistant is served is fixed for the life of the process.
export const getAssistantStatusQueryOptions = () =>
  queryOptions({
    queryKey: assistantKeys.status(),
    queryFn: getAssistantStatus,
    staleTime: Infinity,
  });
