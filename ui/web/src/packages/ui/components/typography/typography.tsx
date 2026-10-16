import clsx from "clsx";
import { PropsWithChildren } from "react";
import type { ComponentProps, ElementType } from "react";
import { Heading } from "react-aria-components";

import {
  TypographyVariant,
  TypographySize,
  TypographyWeight,
} from "./typography.constants";
import { getTypographySizeClassName } from "./utils";

type TypographyProps = {
  as?: ElementType;
  variant?: TypographyVariant;
  size?: TypographySize;
  weight?: TypographyWeight;
  monospace?: boolean;
  className?: string;
};

type TypographyPropsWithAsProps<T extends ElementType> = PropsWithChildren<
  TypographyProps & Omit<ComponentProps<T>, keyof TypographyProps>
>;

const headingLevelFromSize: Record<TypographySize, 1 | 2 | 3 | 4 | 5> = {
  xl: 1,
  lg: 2,
  md: 3,
  sm: 4,
  xs: 5,
};

const headingLevelFromAs: Record<string, 1 | 2 | 3 | 4 | 5 | 6> = {
  h1: 1,
  h2: 2,
  h3: 3,
  h4: 4,
  h5: 5,
  h6: 6,
};

export function Typography<T extends ElementType>({
  as,
  variant = "body",
  size = "md",
  weight = "normal",
  monospace = false,
  className,
  children,
  ...props
}: TypographyPropsWithAsProps<T>) {
  const classes = clsx(
    // Font
    monospace
      ? "font-mono"
      : variant === "heading"
        ? "font-serif"
        : "font-sans",
    // Size
    getTypographySizeClassName(variant, size),
    // Weight
    {
      normal: "font-normal",
      medium: "font-medium",
      semibold: "font-semibold",
    }[weight],
    className
  );

  const shouldRenderHeading =
    (variant === "heading" && !as) ||
    (typeof as === "string" && /^h[1-6]$/.test(as));

  if (shouldRenderHeading) {
    return (
      <Heading
        level={
          (typeof as === "string" && headingLevelFromAs[as]) ||
          headingLevelFromSize[size]
        }
        className={classes}
        {...props}
      >
        {children}
      </Heading>
    );
  }

  const Tag = as ?? "p";

  return (
    <Tag className={classes} {...props}>
      {children}
    </Tag>
  );
}
