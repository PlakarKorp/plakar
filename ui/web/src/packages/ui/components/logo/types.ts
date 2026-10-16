export const LogoVariants = ["icon", "full"] as const;

export type LogoVariant = (typeof LogoVariants)[number];
