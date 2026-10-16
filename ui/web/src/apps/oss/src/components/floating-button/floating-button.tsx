import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";

import { FloatingDemoButton } from "./floating-demo-button";
import { FloatingSupportButton } from "./floating-support-button";

import { getApiInfoQueryOptions } from "~/api/info/queries";

export function FloatingButton() {
  const search = useSearch({ strict: false });
  const { data, isPending } = useQuery(getApiInfoQueryOptions());

  if (isPending) return null;

  const bottom = search.page !== undefined ? "4rem" : "2rem";

  return data?.demo_mode ? (
    <FloatingDemoButton bottom={bottom} />
  ) : (
    <FloatingSupportButton bottom={bottom} />
  );
}
