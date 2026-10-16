export const TypographyVariants = ["body", "heading"] as const;
export const TypographySizes = ["xl", "lg", "md", "sm", "xs"] as const;
export const TypographyWeights = ["normal", "medium", "semibold"] as const;

export type TypographyVariant = (typeof TypographyVariants)[number];
export type TypographySize = (typeof TypographySizes)[number];
export type TypographyWeight = (typeof TypographyWeights)[number];
