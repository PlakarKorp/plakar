import { APIResponseNotFoundError } from "@plakar-ui/common/api/errors";
import {
  ServerSentEventsParser,
  ServerSentEvent,
} from "@plakar-ui/common/utils";

import {
  AssistantErrorSchema,
  AssistantEvent,
  AssistantEventSchema,
  AssistantPage,
  AssistantStatusSchema,
} from "./schema";

import { CallApi, catchify } from "~/api/http";

export async function getAssistantStatus() {
  const url = "/api/assistant";
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
    return AssistantStatusSchema.parse(data).item;
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

type SendAssistantMessageParams = {
  conversationId?: string;
  message: string;
  page?: AssistantPage;
  signal: AbortSignal;
  onEvent: (event: AssistantEvent) => void;
};

/**
 * Sends a message and relays the events of the answer as the server streams
 * them. Resolves once the stream ends, which the server closes with a done or
 * an error event.
 */
export async function sendAssistantMessage({
  conversationId,
  message,
  page,
  signal,
  onEvent,
}: SendAssistantMessageParams) {
  const res = await CallApi("/api/assistant/messages", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      conversation_id: conversationId,
      message,
      page,
    }),
    signal,
  });

  if (!res.ok) {
    throw new Error(await describeError(res));
  }
  if (!res.body) {
    throw new Error("The assistant sent no answer.");
  }

  const handle = (event: ServerSentEvent) => {
    let data;
    try {
      data = JSON.parse(event.data);
    } catch {
      return;
    }
    const parsed = AssistantEventSchema.safeParse({ name: event.name, data });
    if (parsed.success) {
      onEvent(parsed.data);
    }
  };

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  const parser = new ServerSentEventsParser();
  for (;;) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    parser.push(decoder.decode(value, { stream: true })).forEach(handle);
  }
  const last = parser.flush();
  if (last) {
    handle(last);
  }
}

async function describeError(res: Response) {
  const fallback = `The assistant failed (HTTP ${res.status}).`;
  try {
    return AssistantErrorSchema.parse(await res.json()).error.message;
  } catch {
    return fallback;
  }
}

/** Drops a conversation on the server; one it no longer holds is fine. */
export async function forgetAssistantConversation(conversationId: string) {
  await catchify(
    () =>
      CallApi(`/api/assistant/conversations/${conversationId}`, {
        method: "DELETE",
      }),
    APIResponseNotFoundError
  );
}
