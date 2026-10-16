import { PropsWithChildren, useCallback, useState } from "react";
import { z } from "zod";

import { Position, SidePanelContext } from ".";
import { ValidPositions } from "./side-panel-provider.constants";
import { getLocalStorage, setLocalStorage } from "../../utils";

const PositionSchema = z.enum(ValidPositions);

type ProviderProps = PropsWithChildren;

export function SidePanelProvider({ children }: ProviderProps) {
  const [position, setPosition] = useState<Position>(() => {
    const saved = getLocalStorage("sidebar-position");
    const parsed = PositionSchema.safeParse(saved);
    if (parsed.success) {
      return parsed.data;
    }
    return "horizontal";
  });

  const moveSidebar = useCallback((newPosition: Position) => {
    setPosition(newPosition);
    setLocalStorage("sidebar-position", newPosition);
  }, []);

  return (
    <SidePanelContext.Provider
      value={{
        position,
        moveSidebar,
      }}
    >
      {children}
    </SidePanelContext.Provider>
  );
}
