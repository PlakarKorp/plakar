import { useDebouncer } from "@tanstack/react-pacer/debouncer";
import { useEffect, useRef, useState } from "react";

const SEARCH_DEBOUNCE_MS = 300;

export function useDebouncedSearch(
  value: string,
  onSearch: (value: string) => void
) {
  const [localValue, setLocalValue] = useState(value);
  const emitted = useRef(value);

  const debouncer = useDebouncer(
    (next: string) => {
      emitted.current = next;
      onSearch(next);
    },
    { wait: SEARCH_DEBOUNCE_MS }
  );

  useEffect(() => {
    if (value === emitted.current) {
      return;
    }
    emitted.current = value;
    debouncer.cancel();
    setLocalValue(value);
  }, [value, debouncer]);

  const change = (next: string) => {
    setLocalValue(next);
    debouncer.maybeExecute(next);
  };

  const clear = () => {
    setLocalValue("");
    debouncer.cancel();
    emitted.current = "";
    onSearch("");
  };

  return { value: localValue, change, clear };
}
