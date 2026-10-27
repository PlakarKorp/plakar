import { APIResponseUnauthorizedError } from "@plakar-ui/common/api/errors";
import {
  getPaginationParams,
  PaginationParams,
} from "@plakar-ui/common/api/pagination";

import { ReportsListSchema } from "./schema";

import { CallApi, catchify } from "~/api/http";

export type GetReportingReportsParams = {
  repository: string;
  pagination: PaginationParams;
};

export async function getReportingReportsList({
  repository,
  pagination,
}: GetReportingReportsParams) {
  const qs = new URLSearchParams({
    repository_id: repository,
    reverse: "true",
    ...getPaginationParams(pagination),
  });
  const url = `/api/proxy/v1/reporting/reports?${qs}`;

  const res = await catchify(() => CallApi(url), APIResponseUnauthorizedError);

  if (res.error) {
    return { __type: "unauthenticated" } as const;
  }

  let data;
  try {
    data = await res.data.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.data.status} with an invalid JSON payload`
    );
  }
  return {
    __type: "ok",
    ...ReportsListSchema.parse(data),
  } as const;
}
