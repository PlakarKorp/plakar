import { getPaginationParams } from "@plakar-ui/common/api/pagination";

import {
  IntegrationOperationResultSchema,
  IntegrationSchema,
  IntegrationsListSchema,
  type ListIntegrationsParams,
} from "./schema";

import { CallApi } from "~/api/http";

export async function getIntegration({
  integrationId,
}: {
  integrationId: string;
}) {
  const url = `/api/proxy/v1/integration/${integrationId}`;
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
    return IntegrationSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function listIntegrations({
  search,
  types,
  installationStatus,
  page,
  perPage,
}: ListIntegrationsParams) {
  const qs = new URLSearchParams(getPaginationParams({ page, perPage }));
  if (search) {
    qs.set("search", search);
  }
  for (const t of types ?? []) {
    qs.append("type", t);
  }
  if (installationStatus) {
    qs.set("installation_status", installationStatus);
  }
  const url = `/api/proxy/v1/integration?${qs}`;
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
    return IntegrationsListSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function installIntegration({
  integrationId,
  version,
}: {
  integrationId: string;
  version: string;
}) {
  const url = `/api/integrations/install`;

  const res = await CallApi(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id: integrationId, version }),
  });

  let data;
  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  let payload;
  try {
    payload = IntegrationOperationResultSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }

  if (payload.status === "failed") {
    throw new Error(
      payload.messages.map((message) => message.message).join("\n")
    );
  }

  return payload;
}

export async function removeIntegration({
  integrationId,
}: {
  integrationId: string;
}) {
  const url = `/api/integrations/${integrationId}`;

  const res = await CallApi(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
  });

  let data;
  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  let payload;
  try {
    payload = IntegrationOperationResultSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }

  if (payload.status === "failed") {
    throw new Error(
      payload.messages.map((message) => message.message).join("\n")
    );
  }

  return payload;
}
