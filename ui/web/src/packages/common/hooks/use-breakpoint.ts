import { useEffect, useState } from "react";

type Breakpoint = "base" | "sm" | "md" | "lg" | "xl" | "2xl" | "3xl";

const BREAKPOINTS: Record<Breakpoint, number> = {
  base: 0,
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  "2xl": 1536,
  "3xl": 1920,
};

// Determine the current Tailwind CSS breakpoint.
export function useBreakpoint() {
  const [current, setCurrent] = useState<Breakpoint>("base");

  useEffect(() => {
    const queries = Object.entries(BREAKPOINTS)
      .filter(([bp]) => bp !== "base")
      .map(([bp, min]) => ({
        bp: bp as Breakpoint,
        query: window.matchMedia(`(min-width: ${min}px)`),
      }));

    const handleChange = () => {
      const active = queries
        .filter(({ query }) => query.matches)
        .map(({ bp }) => bp);

      setCurrent(active[active.length - 1] ?? "base");
    };

    handleChange();

    queries.forEach(({ query }) =>
      query.addEventListener("change", handleChange)
    );
    return () => {
      queries.forEach(({ query }) =>
        query.removeEventListener("change", handleChange)
      );
    };
  }, []);

  const width = BREAKPOINTS[current];

  return {
    breakpoint: current,
    isSmDown: width < BREAKPOINTS.sm,
    isMdDown: width < BREAKPOINTS.md,
    isLgDown: width < BREAKPOINTS.lg,
    isXlDown: width < BREAKPOINTS.xl,
    is2xlDown: width < BREAKPOINTS["2xl"],
    is3xlDown: width < BREAKPOINTS["3xl"],
    isSmUp: width >= BREAKPOINTS.sm,
    isMdUp: width >= BREAKPOINTS.md,
    isLgUp: width >= BREAKPOINTS.lg,
    isXlUp: width >= BREAKPOINTS.xl,
    is2xlUp: width >= BREAKPOINTS["2xl"],
    is3xlUp: width >= BREAKPOINTS["3xl"],
  };
}
