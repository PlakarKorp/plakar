export const IllustrationTypes = [
  "default",
  "default_nut",
  "juggles_nuts",
  "walking",
  "proud",
  "waving",
  "sad",
  "shrug",
  "emperor",
] as const;

export type IllustrationType = (typeof IllustrationTypes)[number];
