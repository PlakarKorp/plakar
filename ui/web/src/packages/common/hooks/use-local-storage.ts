import { useState, useEffect } from "react";

import { getLocalStorage, setLocalStorage } from "../utils";

export function useLocalStorage<T>(key: string, defaultValue: T) {
  const [state, setState] = useState<T>(() => {
    try {
      const stored = getLocalStorage<T>(key);
      return stored !== null ? stored : defaultValue;
    } catch {
      return defaultValue;
    }
  });

  useEffect(() => {
    try {
      setLocalStorage<T>(key, state);
    } catch {
      // Do nothing if localStorage is not available or quota exceeded
    }
  }, [key, state]);

  return [state, setState] as const;
}
