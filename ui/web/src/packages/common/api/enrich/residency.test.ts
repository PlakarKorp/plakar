import { expect, test } from "vitest";

import { enrichResidency } from "./residency";

// The last case is a code nobody can name: the backend rejects anything that is
// not a country, but it still has to render as something rather than throwing
// or coming back nameless.
test.each([
  ["FR", { alpha2: "FR", flagEmoji: "🇫🇷", name: "France" }],
  ["de", { alpha2: "de", flagEmoji: "🇩🇪", name: "Germany" }],
  ["XX", { alpha2: "XX", flagEmoji: "🇽🇽", name: "XX" }],
])("enriches %s", (code, expected) => {
  expect(enrichResidency(code)).toEqual(expected);
});

// The API omits the field when nobody up the containment chain declared one,
// which has to stay distinguishable from a country so nothing renders.
test("an undeclared residency carries no flag and no name", () => {
  expect(enrichResidency(null)).toEqual({
    alpha2: null,
    flagEmoji: null,
    name: null,
  });
});
