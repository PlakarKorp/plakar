/**
 * Returns a normalized version object with raw and "pretty" representations.
 * @param value The raw version string (e.g. "v1.0.3-devel.c88dc6076910...")
 * @returns An object with:
 * - `Value`: the original input string
 * - `Pretty`: a cleaned-up version string where the commit hash is shortened to the first 7 characters.
 * - `Channel`: keeps only "vX.Y.Z[-tag]" (drops commit).
 * @example
 * Version("v1.2.3") ➝ { Value: "v1.2.3", Pretty: "v1.2.3", Channel: "v1.2.3" }
 * Version("v1.2.3-rc.1.abc1234567890") ➝ { Value: ..., Pretty: "v1.2.3-rc.1.abc1234", Channel: "v1.2.3-rc.1" }
 */
export function Version(value: string) {
  const matches = value.match(/^(v?[^-]+)-(.*)\.([a-f0-9]{7,})$/);
  if (matches) {
    return {
      Value: value,
      Pretty: `${matches[1]}-${matches[2]}.${matches[3].substring(0, 7)}`,
      Channel: `${matches[1]}-${matches[2]}`,
    };
  }

  return {
    Value: value,
    Pretty: value,
    Channel: value,
  };
}

export type EnrichedVersion = ReturnType<typeof Version>;
