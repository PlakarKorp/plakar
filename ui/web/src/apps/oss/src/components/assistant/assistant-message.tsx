import {
  Alert,
  Markdown,
  Spinner,
  Tag,
  Typography,
} from "@plakar-ui/ui/components";

import { ToolCall } from "./tool-call";

import { ChatMessage } from "~/hooks/use-assistant-chat";

type AssistantMessageProps = {
  message: ChatMessage;
};

export function AssistantMessage({ message }: AssistantMessageProps) {
  if (message.role === "user") {
    return (
      <div className="flex justify-end">
        <Typography
          size="sm"
          className="bg-accent-tertiary text-primary max-w-[85%] rounded-xl rounded-br-sm px-3 py-2 whitespace-pre-wrap"
        >
          {message.text}
        </Typography>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {message.parts.map((part, index) =>
        part.type === "text" ? (
          <Markdown key={index} content={part.text} />
        ) : (
          <ToolCall key={part.call.id} call={part.call} />
        )
      )}

      {message.status === "streaming" && (
        <div className="text-tertiary flex items-center gap-2">
          <Spinner size="sm" className="h-auto" inheritColor />
          <Typography size="xs">
            {message.parts.length === 0 ? "Thinking…" : "Working…"}
          </Typography>
        </div>
      )}

      {message.status === "stopped" && (
        <Tag variant="default" size="xs" className="self-start">
          Stopped
        </Tag>
      )}

      {message.status === "error" && (
        <Alert variant="error" size="sm" title="The assistant could not answer">
          {message.error}
        </Alert>
      )}

      {message.status === "done" && message.explanation && (
        <Alert variant="warning" size="sm" title="The assistant declined">
          {message.explanation}
        </Alert>
      )}

      {message.status === "done" && message.stopReason === "max_tokens" && (
        <Alert variant="warning" size="sm" title="The answer was cut short">
          It reached the length limit. Ask for the rest, or a narrower question.
        </Alert>
      )}
    </div>
  );
}
