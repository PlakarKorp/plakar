import { PollLoginParams } from "./api";

export const authKeys = {
  all: () => ["api", "plakar", "auth"] as const,
  poll: (params: PollLoginParams) =>
    [...authKeys.all(), "poll", params] as const,
};
