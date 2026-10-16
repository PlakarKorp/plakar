/**
 * Ensures exhaustive handling of discriminated unions and literal union types.
 *
 * This function is intended to be used in code paths that should be
 * unreachable if all possible variants have been handled correctly.
 *
 * TypeScript will emit a type error whenever a new union member is added
 * and not explicitly handled before reaching `assertNever()`.
 *
 * Example with `switch`:
 *
 * ```ts
 * type Value = "abc" | "def"
 *
 * function render(v: Value) {
 *   switch (v) {
 *     case "abc":
 *       return "ABC"
 *
 *     case "def":
 *       return "DEF"
 *
 *     default:
 *       return assertNever(v)
 *   }
 * }
 * ```
 *
 * If `Value` later becomes:
 *
 * ```ts
 * type Value = "abc" | "def" | "ghi"
 * ```
 *
 * TypeScript will report an error at:
 *
 * ```ts
 * assertNever(v)
 * ```
 *
 * because `v` is no longer `never`.
 */
export function assertNever(x: never): never {
  throw new Error(`Unexpected value: ${x}`);
}
