import { RefObject, useEffect, useRef, useState } from "react";

/**
 * Reports when an element first comes near the viewport, and stays true after
 * that: lazy content loads once and is not thrown away when scrolled past.
 */
export function useInView<T extends Element>(
  rootMargin = "400px"
): [RefObject<T | null>, boolean] {
  const ref = useRef<T>(null);
  const [inView, setInView] = useState(false);

  useEffect(() => {
    const element = ref.current;
    if (!element || inView) {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setInView(true);
          observer.disconnect();
        }
      },
      { rootMargin }
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [inView, rootMargin]);

  return [ref, inView];
}
