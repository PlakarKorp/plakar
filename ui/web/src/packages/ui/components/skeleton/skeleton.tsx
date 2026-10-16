import clsx from "clsx";

import { getTypographySizeClassName } from "..";
import { getIconSizeClassName } from "..";
import { IconSize } from "../icon/icon.constants";
import {
  TypographySizes,
  TypographyVariants,
} from "../typography/typography.constants";

type SkeletonProps = {
  as?: "div" | "span";
  className?: string;
} & (
  | {
      variant: "text";
      type?: (typeof TypographyVariants)[number];
      size?: (typeof TypographySizes)[number];
      width?: number | string;
    }
  | {
      variant: "icon";
      size: IconSize;
    }
  | {
      variant: "circle";
      width: number | string;
    }
  | {
      variant: "rectangle";
      width: number | string;
      height: number | string;
    }
);

export function Skeleton({
  as: Element = "div",
  className,
  ...props
}: SkeletonProps) {
  let styles;

  switch (props.variant) {
    case "text":
      styles = {
        width: props.width,
      };
      break;
    case "circle":
      styles = {
        width: props.width,
        paddingTop: props.width,
      };
      break;
    case "rectangle":
      styles = {
        width: props.width,
        height: props.height,
      };
      break;
  }

  return (
    <Element
      role="status"
      className={clsx(
        "bg-disable animate-pulse",
        Element === "span" && "block",

        props.variant === "text" &&
          clsx(
            "rounded-md before:content-[' ']",
            getTypographySizeClassName(props.type ?? "body", props.size ?? "md")
          ),

        props.variant === "icon" && getIconSizeClassName(props.size),

        props.variant === "circle" && "h-auto rounded-full",
        props.variant === "rectangle" && "rounded-md",
        className
      )}
      style={styles}
    />
  );
}
