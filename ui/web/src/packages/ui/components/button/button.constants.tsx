export const ButtonVariants = [
  "primary",
  "secondary",
  "tertiary",
  "error",
  "error-tertiary",
  "accent-tertiary",
  "accent-quaternary",
  "warning-quaternary",
  "error-quaternary",
  "success-quaternary",
] as const;

export type ButtonVariant = (typeof ButtonVariants)[number];

export const ButtonSizes = ["xs", "sm", "md"] as const;

export type ButtonSize = (typeof ButtonSizes)[number];
