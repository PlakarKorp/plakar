import { APIResponseUnauthorizedError } from "@plakar-ui/common/api/errors";
import {
  getPaginationParams,
  PaginationParams,
} from "@plakar-ui/common/api/pagination";

import {
  AccountMeSchema,
  AccountServicesAlertingSchema,
  NotificationsListSchema,
  NotificationKnownStatus,
} from "./schema";

import { CallApi, catchify } from "~/api/http";

export async function getAccountMe() {
  const url = `/api/proxy/v1/account/me`;
  const res = await catchify(() => CallApi(url), APIResponseUnauthorizedError);

  if (res.error) {
    return null;
  }

  let data;
  try {
    data = await res.data.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.data.status} with an invalid JSON payload`
    );
  }
  return AccountMeSchema.parse(data);
}

export type GetAccountNotificationsParams = {
  repository: string;
  pagination: PaginationParams;
  status?: "unread";
};

export async function getAccountNotifications({
  repository,
  pagination,
  status,
}: GetAccountNotificationsParams) {
  const qs = new URLSearchParams({
    ...getPaginationParams(pagination),
    repository_id: repository,
    reverse: "1",
  });

  if (status) {
    qs.set("status", status);
  }

  const url = `/api/proxy/v1/account/notifications?${qs}`;
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
    notifications: NotificationsListSchema.parse(data),
  } as const;
}

export type SetNotificationStatusParams = {
  notificationIds: string[];
  status: NotificationKnownStatus;
};

export async function setNotificationStatus({
  notificationIds,
  status,
}: SetNotificationStatusParams) {
  const url = `/api/proxy/v1/account/notifications/set-status`;

  await CallApi(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ notification_ids: notificationIds, status }),
  });
}

export async function getAccountServicesAlerting() {
  const url = `/api/proxy/v1/account/services/alerting`;
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
    configuration: AccountServicesAlertingSchema.parse(data),
  } as const;
}

export type SetAccountServicesAlertingParams = {
  enabled: boolean;
  email_report: boolean;
};

export async function setAccountServicesAlerting({
  enabled,
  email_report,
}: SetAccountServicesAlertingParams) {
  const url = `/api/proxy/v1/account/services/alerting`;

  const res = await CallApi(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled, email_report }),
  });

  let data;
  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  return AccountServicesAlertingSchema.parse(data);
}
