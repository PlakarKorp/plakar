import { ApiInfoSchema } from "./schema";

import { CallApi } from "~/api/http";

export async function getApiInfo() {
  const url = "/api/info";
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  try {
    return ApiInfoSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}
