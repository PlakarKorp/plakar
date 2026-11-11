import { useQuery } from "@tanstack/react-query";
import { useLocation, useSearch } from "@tanstack/react-router";

import { FloatingDemoButton } from "./floating-demo-button";
import { FloatingSupportButton } from "./floating-support-button";

import { getApiInfoQueryOptions } from "~/api/info/queries";

export function FloatingButton() {
  const search = useSearch({ strict: false });
  const pathname = useLocation({ select: (location) => location.pathname });
  const { data, isPending } = useQuery(getApiInfoQueryOptions());

  // Nothing floats over the Gibson.
  if (isPending || pathname.startsWith("/gibson")) return null;

  const bottom = search.page !== undefined ? "4rem" : "2rem";

  return data?.demo_mode ? (
    <FloatingDemoButton bottom={bottom} />
  ) : (
    <FloatingSupportButton bottom={bottom} />
  );
}
