import { useEffect, useState } from "react";

// useSpinner returns a boolean indicating whether to show a spinner based on
// the `loaded` state.
// If `loaded` is true, the spinner will not be shown.
// If `loaded` is false, the spinner will be shown after a specified delay.
//
// This is useful for scenarios where you want to show a loading spinner while
// data is being fetched, but avoid showing it immediately if the data is
// quickly available to avoid flickering.
export function useSpinner(loaded: boolean) {
  const delay = 100; // ms
  const [showSpinner, setShowSpinner] = useState(loaded);

  useEffect(() => {
    if (loaded) {
      setShowSpinner(false);
      return;
    }

    const timer = setTimeout(() => {
      setShowSpinner(true);
    }, delay);

    return () => clearTimeout(timer);
  }, [loaded, delay]);

  return showSpinner;
}
