import clsx from "clsx";
import { useRef, useState } from "react";

import { Content, ContentProps } from "./content";
import { Header } from "./mobile/header";
import { useBreakpoint } from "../../hooks";
import { useForceCollapse } from "../../hooks/use-force-collapse";

export type NavigationProps = Pick<
  ContentProps,
  | "navItems"
  | "subNavItems"
  | "user"
  | "version"
  | "plan"
  | "isDemo"
  | "controls"
  | "scope"
  | "footer"
  | "logo"
  | "isOffline"
>;

export function Navigation(contentProps: NavigationProps) {
  const { isLgUp, isXlUp } = useBreakpoint();
  const forceCollapse = useForceCollapse();

  const [shouldExpand, setShouldExpand] = useState(true);
  const [shouldHoverExpand, setShouldHoverExpand] = useState(false);
  const isPopoverOpenRef = useRef(false);

  if (!isLgUp) {
    return <Header {...contentProps} />;
  }

  const canExpand = !forceCollapse && isXlUp;
  const isPinned = shouldExpand && canExpand;
  const isExpanded = isPinned || shouldHoverExpand;
  const isFloating = shouldHoverExpand && !isPinned;
  const isAnimated = !forceCollapse;

  const handleMouseEnter = () => {
    setShouldHoverExpand(true);
  };
  const handleMouseLeave = () => {
    if (isPopoverOpenRef.current) return;
    setShouldHoverExpand(false);
  };
  const handlePopoverOpenChange = (isOpen: boolean) => {
    isPopoverOpenRef.current = isOpen;
  };
  const handleToggle = () => {
    setShouldExpand((prev) => !prev);
    setShouldHoverExpand(false);
  };

  return (
    <>
      <aside
        className={clsx(
          "bg-primary border-secondary scrollbar-hide absolute z-50 h-full overflow-y-auto border-r p-4",
          isExpanded ? "w-70" : "w-18",
          isFloating && "shadow-xl"
        )}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        <Content
          {...contentProps}
          isExpanded={isExpanded}
          isFloating={isFloating}
          canExpand={canExpand}
          handleToggle={handleToggle}
          onPopoverOpenChange={handlePopoverOpenChange}
        />
      </aside>
      <div
        className={clsx(
          "h-full",
          isExpanded && !isFloating ? "w-70" : "w-18",
          isAnimated && "transition-[width] duration-150 ease-out"
        )}
      />
    </>
  );
}
export type { NavigationScopeProps } from "./content";
