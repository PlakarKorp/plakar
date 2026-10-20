import { CallApi } from "~/api/http";

export const quit = async () => {
  await CallApi("/api/quit", { method: "POST" });
  return null;
};
