import { createContext, MouseEvent, useContext } from "react";

export const EntryMenuContext = createContext<{
  openPath: string | null;
  setOpenPath: (path: string | null) => void;
}>({ openPath: null, setOpenPath: () => {} });

export function useEntryMenu(path: string) {
  const { openPath, setOpenPath } = useContext(EntryMenuContext);
  return {
    isOpen: openPath === path,
    onOpenChange: (isOpen: boolean) => setOpenPath(isOpen ? path : null),
    onContextMenu: (event: MouseEvent) => {
      event.preventDefault();
      setOpenPath(path);
    },
  };
}
