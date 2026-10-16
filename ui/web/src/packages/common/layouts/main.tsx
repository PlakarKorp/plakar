import {
  Typography,
  ProgressBar,
  Tab,
  TabItem,
  IconType,
} from "@plakar-ui/ui/components";
import clsx from "clsx";
import { PropsWithChildren, ReactNode, useRef } from "react";

import { IconBadge, IconBadgeType } from "../components";
import { useScrollPosition } from "../hooks";
import { StickyControls } from "./sticky-controls";

export type MainProps = PropsWithChildren<{
  title?: ReactNode;
  heading?: ReactNode;
  description?: ReactNode;
  summary?: ReactNode;
  isLoading?: boolean;
  breadcrumb?: ReactNode;
  controls?: ReactNode;
  className?: string;
  tabItems?: TabItem[];
  selectedTab?: string | null;
  tabsAction?: ReactNode;
  tabsAsTitle?: boolean;
  variant?: "main" | "sub";
  icon?: IconType;
  iconBadgeType?: IconBadgeType;
}>;

export function Main({
  title,
  heading,
  description,
  summary,
  isLoading = false,
  breadcrumb,
  controls,
  children,
  className,
  tabItems,
  selectedTab,
  tabsAction,
  tabsAsTitle = false,
  variant = "main",
  icon,
  iconBadgeType,
}: MainProps) {
  const controlsRef = useRef<HTMLDivElement>(null);
  const { isScrolledTo } = useScrollPosition(controlsRef, { offset: 12 });

  const tabs = (tabItems || tabsAction) && (
    <div className="flex items-center gap-2">
      {tabItems && (
        <Tab
          variant={tabsAsTitle ? "pill" : "underlined"}
          ariaLabel="tabs"
          items={tabItems}
          selected={selectedTab}
        />
      )}

      {tabsAction}
    </div>
  );

  const stickyHeading =
    title ?? tabItems?.find((tab) => tab.id === selectedTab)?.title;

  const titleBlock = heading ? (
    <>
      <Typography as={variant === "main" ? "h1" : "h2"} className="sr-only">
        {title}
      </Typography>
      {heading}
    </>
  ) : (
    <div className="flex items-center gap-3">
      {icon && <IconBadge icon={icon} type={iconBadgeType} />}

      <div className={clsx(variant === "sub" && "min-h-9.5")}>
        {tabsAsTitle ? (
          tabs
        ) : (
          <>
            <Typography
              as={variant === "main" ? "h1" : "h2"}
              variant="heading"
              size="md"
              weight="semibold"
              className="text-primary"
            >
              {title}
            </Typography>
            <Typography
              as="div"
              variant="body"
              size="sm"
              className="text-secondary"
            >
              {description}
            </Typography>
          </>
        )}
      </div>
    </div>
  );

  return (
    <div
      id="main_scroll"
      className={clsx(
        "flex flex-col",
        variant === "main" && "h-full overflow-auto",
        className
      )}
    >
      <ProgressBar isLoading={isLoading} className="fixed top-0 left-0 z-50" />

      {controls && isScrolledTo && (
        <StickyControls>
          <div className="flex w-full items-center justify-between gap-2">
            <Typography
              as="span"
              variant="heading"
              size="md"
              weight="semibold"
              className="text-primary hidden sm:block"
            >
              {stickyHeading}
            </Typography>

            <div className="w-full sm:w-auto">{controls}</div>
          </div>
        </StickyControls>
      )}

      {(title || heading || breadcrumb || controls || tabs) && (
        <div
          className={clsx(
            {
              main: clsx(
                "bg-primary border-secondary border-b px-2 pt-4 lg:px-4",
                (!tabItems || tabsAsTitle) && "pb-4"
              ),
              sub: "bg-secondary mb-4",
            }[variant]
          )}
        >
          <div className="flex flex-col gap-2 2xl:container 2xl:mx-auto">
            {breadcrumb}

            {(title || heading || controls || (tabsAsTitle && tabs)) && (
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                {titleBlock}

                {controls && (
                  <div ref={controlsRef} className="ml-auto w-full sm:w-auto">
                    {controls}
                  </div>
                )}
              </div>
            )}

            {summary && <div className="py-2">{summary}</div>}

            {!tabsAsTitle && tabs}
          </div>
        </div>
      )}

      <div
        className={clsx(
          "flex-1 2xl:container 2xl:mx-auto",
          {
            main: "px-2 py-4 lg:px-4",
            sub: "",
          }[variant]
        )}
      >
        {children}
      </div>
    </div>
  );
}
