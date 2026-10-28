import {
  ExchangeResponseSchema,
  LoginPollResponseSchema,
  LoginStartResponseSchema,
} from "~/api/auth/schema";
import { CallApi } from "~/api/http";

const parseJSON = async (url: string, res: Response) => {
  try {
    return await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }
};

const startLogin = async (url: string, body: object) => {
  const res = await CallApi(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return LoginStartResponseSchema.parse(await parseJSON(url, res));
};

export type SendEmailLoginMagicLinkParams = {
  email: string;
};

export const sendEmailLoginMagicLink = ({
  email,
}: SendEmailLoginMagicLinkParams) =>
  startLogin("/api/authentication/login/email", { email });

export const getGithubLoginUrl = () =>
  startLogin("/api/authentication/login/github", {});

export type PollLoginParams = {
  pollId: string;
  completionCode?: string;
};

export const pollLogin = async ({
  pollId,
  completionCode,
}: PollLoginParams) => {
  const url = "/api/authentication/login/poll";

  const res = await CallApi(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      poll_id: pollId,
      completion_code: completionCode,
    }),
  });
  return LoginPollResponseSchema.parse(await parseJSON(url, res));
};

export const logout = async () => {
  const url = "/api/authentication/logout";

  await CallApi(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });

  return null;
};

// Trade the one-time code plakar ui put in the launch URL for the API token.
export const exchangeLaunchCode = async (code: string) => {
  const url = "/api/authentication/exchange";

  const res = await CallApi(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code }),
  });
  return ExchangeResponseSchema.parse(await parseJSON(url, res));
};
