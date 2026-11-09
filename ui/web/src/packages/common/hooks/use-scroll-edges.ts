import { RefObject, useEffect, useRef, useState } from "react";

/**
 * Reports whether a scrollable element hides content above or below its
 * visible part, so the edge that cuts something off can say so.
 */
export function useScrollEdges<T extends HTMLElement>(): [
  RefObject<T | null>,
  { isCutTop: boolean; isCutBottom: boolean },
] {
  const ref = useRef<T>(null);
  const [edges, setEdges] = useState({ isCutTop: false, isCutBottom: false });

  useEffect(() => {
    const element = ref.current;
    if (!element) {
      return;
    }
    const update = () =>
      setEdges({
        isCutTop: element.scrollTop > 0,
        isCutBottom:
          element.scrollTop + element.clientHeight < element.scrollHeight - 1,
      });

    update();
    element.addEventListener("scroll", update, { passive: true });
    // The content grows when a group expands or entries load, without any
    // scroll.
    const observer = new ResizeObserver(update);
    observer.observe(element);
    Array.from(element.children).forEach((child) => observer.observe(child));

    return () => {
      element.removeEventListener("scroll", update);
      observer.disconnect();
    };
  }, []);

  return [ref, edges];
}
