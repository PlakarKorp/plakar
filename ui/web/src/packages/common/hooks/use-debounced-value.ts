import { useDebouncedValue as usePacerDebouncedValue } from "@tanstack/react-pacer/debouncer";

export function useDebouncedValue<T>(value: T, wait: number) {
  const [debounced] = usePacerDebouncedValue(value, { wait });
  return debounced;
}
