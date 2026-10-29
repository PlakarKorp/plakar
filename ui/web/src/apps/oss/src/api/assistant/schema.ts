import { z } from "zod";

export const AssistantStatusSchema = z.object({
  item: z.object({
    enabled: z.boolean(),
    provider: z.string().optional(),
    model: z.string().optional(),
    local: z.boolean().optional(),
    tools: z.array(z.string()).optional(),
  }),
});

export type AssistantStatus = z.infer<typeof AssistantStatusSchema>["item"];

// One server-sent event of an answer, keyed by the event name the server
// sends. Anything else on the stream is ignored.
export const AssistantEventSchema = z.discriminatedUnion("name", [
  z.object({
    name: z.literal("conversation"),
    data: z.object({ id: z.string() }),
  }),
  z.object({
    name: z.literal("text"),
    data: z.object({ text: z.string() }),
  }),
  z.object({
    name: z.literal("tool_use"),
    data: z.object({ id: z.string(), name: z.string() }),
  }),
  z.object({
    name: z.literal("tool_result"),
    data: z.object({
      id: z.string(),
      name: z.string(),
      input: z.record(z.string(), z.json()),
      output: z.string(),
      is_error: z.boolean(),
      truncated: z.boolean().optional(),
    }),
  }),
  z.object({
    name: z.literal("done"),
    data: z.object({
      stop_reason: z.string(),
      explanation: z.string().optional(),
    }),
  }),
  z.object({
    name: z.literal("error"),
    data: z.object({ message: z.string() }),
  }),
]);

export type AssistantEvent = z.infer<typeof AssistantEventSchema>;

export const AssistantErrorSchema = z.object({
  error: z.object({ message: z.string() }),
});
