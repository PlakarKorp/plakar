import { useContext } from "react";

import { SidePanelContext } from ".";

export function useSidePanel() {
  const ctx = useContext(SidePanelContext);

  if (!ctx) {
    throw new Error("useSidePanel must be used within SidePanelProvider");
  }

  return ctx;
}
