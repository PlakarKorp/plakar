import { useCallback, useRef, useState } from "react";

import {
  forgetAssistantConversation,
  sendAssistantMessage,
} from "~/api/assistant/api";
import { AssistantEvent, AssistantPage } from "~/api/assistant/schema";

export type ToolCall = {
  id: string;
  name: string;
  status: "running" | "done";
  input?: string;
  output?: string;
  isError?: boolean;
  truncated?: boolean;
};

export type AssistantPart =
  { type: "text"; text: string } | { type: "tool"; call: ToolCall };

export type AssistantMessage = {
  id: number;
  role: "assistant";
  parts: AssistantPart[];
  // streaming while the answer arrives; stopped when the reader interrupted
  // it; error when the server or the network gave up on it.
  status: "streaming" | "done" | "stopped" | "error";
  error?: string;
  stopReason?: string;
  explanation?: string;
};

export type UserMessage = {
  id: number;
  role: "user";
  text: string;
};

export type ChatMessage = UserMessage | AssistantMessage;

function appendText(message: AssistantMessage, text: string): AssistantMessage {
  const last = message.parts[message.parts.length - 1];
  if (last?.type === "text") {
    return {
      ...message,
      parts: [
        ...message.parts.slice(0, -1),
        { ...last, text: last.text + text },
      ],
    };
  }
  return { ...message, parts: [...message.parts, { type: "text", text }] };
}

// A call still waiting for its result when the answer ends never gets one.
function settleToolCalls(message: AssistantMessage): AssistantMessage {
  return {
    ...message,
    parts: message.parts.map((part) =>
      part.type === "tool" && part.call.status === "running"
        ? { ...part, call: { ...part.call, status: "done" } }
        : part
    ),
  };
}

function applyEvent(
  message: AssistantMessage,
  event: AssistantEvent
): AssistantMessage {
  switch (event.name) {
    case "text":
      return appendText(message, event.data.text);
    case "tool_use":
      return {
        ...message,
        parts: [
          ...message.parts,
          {
            type: "tool",
            call: {
              id: event.data.id,
              name: event.data.name,
              status: "running",
            },
          },
        ],
      };
    case "tool_result":
      return {
        ...message,
        parts: message.parts.map((part) =>
          part.type === "tool" && part.call.id === event.data.id
            ? {
                ...part,
                call: {
                  ...part.call,
                  status: "done",
                  input: JSON.stringify(event.data.input, null, 2),
                  output: event.data.output,
                  isError: event.data.is_error,
                  truncated: event.data.truncated,
                },
              }
            : part
        ),
      };
    case "done":
      return {
        ...message,
        status: "done",
        stopReason: event.data.stop_reason,
        explanation: event.data.explanation,
      };
    case "error":
      return { ...message, status: "error", error: event.data.message };
    case "conversation":
      return message;
  }
}

/**
 * Holds one conversation with the assistant: the messages shown, and the
 * answer in progress. The server keeps the history; this side keeps its ID.
 */
export function useAssistantChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const conversationRef = useRef<string | undefined>(undefined);
  const abortRef = useRef<AbortController | null>(null);
  const nextIdRef = useRef(1);

  const updateAnswer = useCallback(
    (update: (message: AssistantMessage) => AssistantMessage) => {
      setMessages((previous) => {
        const last = previous[previous.length - 1];
        if (!last || last.role !== "assistant") {
          return previous;
        }
        return [...previous.slice(0, -1), update(last)];
      });
    },
    []
  );

  const send = useCallback(
    async (text: string, page?: AssistantPage) => {
      const message = text.trim();
      if (message === "" || abortRef.current) {
        return;
      }

      const controller = new AbortController();
      abortRef.current = controller;
      setIsStreaming(true);
      setMessages((previous) => [
        ...previous,
        { id: nextIdRef.current++, role: "user", text: message },
        {
          id: nextIdRef.current++,
          role: "assistant",
          parts: [],
          status: "streaming",
        },
      ]);

      try {
        await sendAssistantMessage({
          conversationId: conversationRef.current,
          message,
          page,
          signal: controller.signal,
          onEvent: (event) => {
            if (event.name === "conversation") {
              conversationRef.current = event.data.id;
              return;
            }
            updateAnswer((answer) => applyEvent(answer, event));
          },
        });
        // A stream that ends without done or error was cut by the network.
        updateAnswer((answer) =>
          answer.status === "streaming"
            ? {
                ...settleToolCalls(answer),
                status: "error",
                error: "The answer ended before it was complete.",
              }
            : answer
        );
      } catch (error) {
        if (controller.signal.aborted) {
          updateAnswer((answer) => ({
            ...settleToolCalls(answer),
            status: "stopped",
          }));
        } else {
          updateAnswer((answer) => ({
            ...settleToolCalls(answer),
            status: "error",
            error: error instanceof Error ? error.message : String(error),
          }));
        }
      } finally {
        abortRef.current = null;
        setIsStreaming(false);
      }
    },
    [updateAnswer]
  );

  const stop = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  const reset = useCallback(() => {
    abortRef.current?.abort();
    const conversationId = conversationRef.current;
    conversationRef.current = undefined;
    setMessages([]);
    if (conversationId) {
      void forgetAssistantConversation(conversationId);
    }
  }, []);

  return { messages, isStreaming, send, stop, reset };
}

export type AssistantChat = ReturnType<typeof useAssistantChat>;
