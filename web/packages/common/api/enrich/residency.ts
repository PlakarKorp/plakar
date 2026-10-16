// Maps an ISO 3166-1 alpha-2 code to its flag emoji by shifting each letter
// into the Unicode "regional indicator symbol" block -- the same trick used
// by every emoji-flag implementation, no lookup table needed. Rendering
// support varies by platform/font: most modern OSes show the flag glyph,
// some (older Windows, some Linux font stacks) fall back to showing the two
// letters in boxes, which just reads as a redundant echo of the code rather
// than broken.
export function alpha2ToEmoji(alpha2: string) {
  return String.fromCodePoint(
    ...alpha2
      .toUpperCase()
      .split("")
      .map((letter) => 127397 + letter.charCodeAt(0))
  );
}

const regionNames = new Intl.DisplayNames(["en"], { type: "region" });

export function alpha2ToCountryName(alpha2: string) {
  return regionNames.of(alpha2.toUpperCase()) ?? alpha2;
}

export function enrichResidency(alpha2: string | null) {
  return {
    alpha2: alpha2 || null,
    flagEmoji: alpha2 ? alpha2ToEmoji(alpha2) : null,
    name: alpha2 ? alpha2ToCountryName(alpha2) : null,
  };
}

export type EnrichedResidency = ReturnType<typeof enrichResidency>;
