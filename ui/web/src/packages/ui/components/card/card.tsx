import clsx from "clsx";
import {
  ComponentProps,
  createContext,
  PropsWithChildren,
  useContext,
  useState,
} from "react";

import { Button, Illustration, IllustrationType, Typography } from "..";

// A tint colours the header and gives the border the same hue, so a card
// reads as one block. A new tint sets both here.
const TINTS = {
  accent: {
    header: "bg-accent-quaternary",
    border: "border-accent-tertiary",
  },
} as const;

type CardTint = keyof typeof TINTS;

type CardProps = PropsWithChildren<{
  className?: string;
  defaultCollapsed?: boolean;
  isCollapsible?: boolean;
  isError?: boolean;
  /** Colours the header and the border. The header divider is full width. */
  tint?: CardTint;
}>;

const Context = createContext<{
  isCollapsible: boolean;
  isCollapsed: boolean;
  toggleCollapsed: () => void;
  tint?: CardTint;
}>({
  isCollapsible: false,
  isCollapsed: false,
  toggleCollapsed: () => {},
});

function CardRoot({
  children,
  defaultCollapsed = true,
  isCollapsible = false,
  isError = false,
  tint,
  className,
}: CardProps) {
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);

  const toggleCollapsed = () => setIsCollapsed((prev) => !prev);

  return (
    <Context.Provider
      value={{ isCollapsible, isCollapsed, toggleCollapsed, tint }}
    >
      <div
        className={clsx(
          "bg-primary rounded-xl border",
          isError
            ? "border-error"
            : tint
              ? TINTS[tint].border
              : "border-secondary",
          className
        )}
      >
        {children}
      </div>
    </Context.Provider>
  );
}

type CardHeaderProps = {
  prefix?: React.ReactNode;
  title: React.ReactNode;
  titleSuffix?: React.ReactNode;
  description?: React.ReactNode;
  suffix?: React.ReactNode;
  buttonProps?: ComponentProps<typeof Button>;
  isDividerFullWidth?: boolean;
  className?: string;
};

function CardHeader({
  prefix,
  title,
  titleSuffix,
  description,
  suffix,
  buttonProps,
  isDividerFullWidth = false,
  className,
}: CardHeaderProps) {
  const { isCollapsible, isCollapsed, toggleCollapsed, tint } =
    useContext(Context);
  const isOpen = !isCollapsible || !isCollapsed;

  return (
    <div
      className={clsx(
        "b flex flex-wrap items-center justify-between gap-2 py-2 md:py-4",
        isDividerFullWidth || tint ? "px-2 md:px-4" : "mx-2 md:mx-4",
        isOpen && "border-secondary border-b",
        // The card radius less its border, so the tint meets the border. A
        // header with nothing below it rounds its bottom too.
        tint && [
          TINTS[tint].header,
          "rounded-t-[calc(var(--radius-xl)-1px)] last:rounded-b-[calc(var(--radius-xl)-1px)] last:border-b-0",
        ],
        className
      )}
    >
      <div className="flex items-center gap-3">
        {prefix}

        <div>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <Typography
              as="h2"
              variant="body"
              size="md"
              weight="medium"
              className="text-primary"
            >
              {title}
            </Typography>
            {titleSuffix}
          </div>
          {description && (
            <Typography size="xs" className="text-tertiary">
              {description}
            </Typography>
          )}
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {buttonProps && (
          <Button size="sm" variant="tertiary" {...buttonProps} />
        )}

        {suffix}

        {isCollapsible && (
          <Button
            size="sm"
            variant="tertiary"
            leftIcon={isCollapsed ? "caret-down" : "caret-up"}
            onClick={toggleCollapsed}
            aria-label={isCollapsed ? "Expand card" : "Collapse card"}
          >
            {isCollapsed ? "Show" : "Hide"}
          </Button>
        )}
      </div>
    </div>
  );
}

type HighlightedCardHeaderProps = {
  title: string;
  description: string;
  illustration: IllustrationType;
};

function HighlightedCardHeader({
  title,
  description,
  illustration,
}: HighlightedCardHeaderProps) {
  return (
    <div className="from-accent to-accent/80 -mx-0.25 -mt-0.25 flex flex-col rounded-t-xl bg-gradient-to-r">
      <div className="relative -mt-2 mr-2 flex overflow-hidden pt-5 pb-4">
        <Illustration
          type={illustration}
          width={100}
          className="absolute top-0 left-1"
        />
        <div className="w-26 shrink-0" />
        <div>
          <Typography
            as="h2"
            variant="heading"
            size="sm"
            weight="medium"
            className="text-on-brand"
          >
            {title}
          </Typography>
          <Typography size="sm" className="text-on-brand-secondary">
            {description}
          </Typography>
        </div>
      </div>
    </div>
  );
}

type CardContentProps = PropsWithChildren<{
  padding?: "md" | "lg" | "xl";
  noPaddingX?: boolean;
  noPaddingY?: boolean;
  className?: string;
}>;

function CardContent({
  children,
  padding = "md",
  noPaddingX = false,
  noPaddingY = false,
  className,
}: CardContentProps) {
  const { isCollapsible, isCollapsed } = useContext(Context);

  if (isCollapsible && isCollapsed) {
    return null;
  }

  const classes = {
    md: clsx(!noPaddingX && "px-2 md:px-4", !noPaddingY && "py-2 md:py-4"),
    lg: clsx(!noPaddingX && "px-3 md:px-6", !noPaddingY && "py-3 md:py-6"),
    xl: clsx(!noPaddingX && "px-4 md:px-8", !noPaddingY && "py-4 md:py-8"),
  }[padding];

  return <div className={clsx(classes, className)}>{children}</div>;
}

function CardFooter({
  children,
  className,
}: PropsWithChildren<{ className?: string }>) {
  const { isCollapsible, isCollapsed } = useContext(Context);

  if (isCollapsible && isCollapsed) {
    return null;
  }

  return (
    <div
      className={clsx(
        "border-secondary bg-secondary rounded-b-xl border-t px-3 py-3 md:px-6 md:py-4",
        className
      )}
    >
      {children}
    </div>
  );
}

export const Card = Object.assign(CardRoot, {
  Header: CardHeader,
  HighlightedHeader: HighlightedCardHeader,
  Content: CardContent,
  Footer: CardFooter,
});
