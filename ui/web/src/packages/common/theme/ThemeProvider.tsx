import { PropsWithChildren, useEffect, useMemo, useState } from "react";

import { Mode, Scheme, ThemeContext } from ".";
import { useLocalStorage } from "../hooks";

function getSystemMode(): Scheme {
  if (typeof window === "undefined" || !window.matchMedia) {
    return "light";
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

export function ThemeProvider({ children }: PropsWithChildren) {
  const [mode, setMode] = useLocalStorage<Mode>("theme", "system");
  const [systemMode, setSystemMode] = useState<Scheme>(getSystemMode);

  useEffect(() => {
    if (
      mode !== "system" ||
      typeof window === "undefined" ||
      !window.matchMedia
    )
      return;

    const mediaQueries = window.matchMedia("(prefers-color-scheme: dark)");
    const update = () => setSystemMode(mediaQueries.matches ? "dark" : "light");
    update();

    mediaQueries.addEventListener?.("change", update);
    return () => mediaQueries.removeEventListener?.("change", update);
  }, [mode]);

  const effectiveMode = useMemo(
    () => (mode === "system" ? systemMode : mode),
    [mode, systemMode]
  );

  useEffect(() => {
    document.documentElement.classList.toggle("dark", effectiveMode === "dark");
  }, [effectiveMode]);

  return (
    <ThemeContext.Provider
      value={{
        value: effectiveMode,
        mode,
        setMode,
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
}
