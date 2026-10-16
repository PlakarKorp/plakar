import { useSearch } from "@tanstack/react-router";

import { useBreakpoint } from ".";
import { useSidePanel } from "../layouts/side-panel";

export function useForceCollapse() {
  const { position } = useSidePanel();
  const { preview } = useSearch({ strict: false }) as { preview?: string };
  const { is3xlDown } = useBreakpoint();

  return !!preview && position === "horizontal" && is3xlDown;
}
