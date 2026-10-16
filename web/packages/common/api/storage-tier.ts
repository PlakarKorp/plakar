export const StorageTiers = ["hot", "cold"] as const;
export type StorageTier = (typeof StorageTiers)[number];
