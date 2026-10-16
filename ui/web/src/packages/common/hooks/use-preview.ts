import { createContext, useContext } from "react";

export type PreviewContextValue = {
  preview: boolean;
  setPreview: (value: boolean) => void;
};

export const PreviewContext = createContext<PreviewContextValue | null>(null);

export function usePreview(): PreviewContextValue {
  const ctx = useContext(PreviewContext);
  if (!ctx) {
    throw new Error("usePreview must be used within a PreviewProvider");
  }
  return ctx;
}
