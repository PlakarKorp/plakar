import { PropsWithChildren } from "react";

import { useLocalStorage } from "./use-local-storage";
import { PreviewContext } from "./use-preview";

export function LocalStoragePreviewProvider({ children }: PropsWithChildren) {
  const [preview, setPreview] = useLocalStorage("preview", false);

  return (
    <PreviewContext.Provider value={{ preview, setPreview }}>
      {children}
    </PreviewContext.Provider>
  );
}
