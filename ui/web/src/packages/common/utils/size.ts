export const SizeUnits = ["MB", "GB", "TB"] as const;

export type SizeUnit = (typeof SizeUnits)[number];

const BYTES_PER_UNIT: Record<SizeUnit, number> = {
  MB: 1e6,
  GB: 1e9,
  TB: 1e12,
};

export function getBytesFromSize(amount: number, unit: SizeUnit) {
  return amount * BYTES_PER_UNIT[unit];
}

export function getAmountFromBytes(bytes: number, unit: SizeUnit) {
  return bytes / BYTES_PER_UNIT[unit];
}

/**
 * Splits a byte count into the largest unit that keeps it above one, so a
 * threshold set in terabytes is read back in terabytes.
 */
export function getSizeFromBytes(bytes: number) {
  const unit =
    [...SizeUnits]
      .reverse()
      .find((candidate) => bytes >= BYTES_PER_UNIT[candidate]) ?? SizeUnits[0];

  return { amount: getAmountFromBytes(bytes, unit), unit };
}
