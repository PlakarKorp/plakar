export const CardVariants = ["primary", "secondary"] as const;
export type CardVariant = (typeof CardVariants)[number];
