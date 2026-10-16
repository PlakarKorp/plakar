import { createContext } from "react";

import { ValidPositions } from "./side-panel-provider.constants";

export type Position = (typeof ValidPositions)[number];

type SidePanelContextValue = {
  position: Position;
  moveSidebar: (newPosition: Position) => void;
};

export const SidePanelContext = createContext<SidePanelContextValue>({
  position: "horizontal",
  moveSidebar: () => {},
});
