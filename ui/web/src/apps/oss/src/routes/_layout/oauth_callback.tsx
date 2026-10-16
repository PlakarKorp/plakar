import { Spinner } from "@plakar-ui/ui/components";
import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { useWaitLoginStatusThenRedirect } from "~/hooks/use-api";

export const Route = createFileRoute("/_layout/oauth_callback")({
  validateSearch: z.object({
    returnUrl: z.string().default("").catch(""),
  }),
  component: Component,
});

function Component() {
  const { returnUrl } = Route.useSearch();

  useWaitLoginStatusThenRedirect({ status: "logged_in", returnUrl });
  return <Spinner />;
}
