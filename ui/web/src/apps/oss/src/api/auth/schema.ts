import { z } from "zod";

// A sign-in started through the plakar backend. The token is only released
// against the completion code shown to whoever completes the sign-in, so the
// page keeps the poll ID and polls with the code the user types.
export const LoginStartResponseSchema = z.object({
  URL: z.string().optional().default(""),
  poll_id: z.string(),
});

export type LoginStartResponse = z.infer<typeof LoginStartResponseSchema>;

export const LoginPollResponseSchema = z.object({
  status: z.enum(["pending", "code-required", "invalid-code", "ok"]),
});
