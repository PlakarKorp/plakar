import { TypographySize, TypographyVariant } from "./typography.constants";

export function getTypographySizeClassName(
  variant: TypographyVariant,
  size: TypographySize
) {
  return variant === "heading"
    ? {
        xl: "text-4xl",
        lg: "text-3xl",
        md: "text-2xl",
        sm: "text-xl",
        xs: "text-lg",
      }[size]
    : {
        xl: "text-xl",
        lg: "text-lg",
        md: "text-base",
        sm: "text-sm",
        xs: "text-xs",
      }[size];
}
