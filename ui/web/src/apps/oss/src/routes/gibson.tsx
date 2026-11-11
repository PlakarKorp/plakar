import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { Gibson } from "~/components/gibson/gibson";

// Outside the layout: the Gibson takes the whole screen.
export const Route = createFileRoute("/gibson")({
  validateSearch: z.object({
    snapshot: z.string().optional(),
    path: z.string().optional(),
  }),
  component: Component,
});

function Component() {
  const search = Route.useSearch();
  return <Gibson location={search} />;
}
