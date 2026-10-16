import { useEffect, useState, RefObject } from "react";

type Options = {
  offset?: number;
};

export function useScrollPosition(
  targetRef: RefObject<HTMLElement | null>,
  { offset = 0 }: Options = {}
) {
  const [isScrolledTo, setIsScrolledTo] = useState(false);
  const [isScrolledPast, setIsScrolledPast] = useState(false);

  useEffect(() => {
    if (!targetRef.current) return;

    const handleScroll = () => {
      if (!targetRef.current) return;

      const rect = targetRef.current.getBoundingClientRect();
      setIsScrolledTo(rect.top <= offset);
      setIsScrolledPast(rect.bottom <= offset);
    };

    const scrollableElement = document.querySelector("#main_scroll") ?? window;

    scrollableElement.addEventListener("scroll", handleScroll, {
      passive: true,
    });

    return () => {
      scrollableElement.removeEventListener("scroll", handleScroll);
    };
  }, [targetRef, offset]);

  return { isScrolledTo, isScrolledPast };
}
