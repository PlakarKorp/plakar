import { expect, test } from "vitest";

import { getBytesFromSize, getSizeFromBytes } from "./size";

test("converts an amount and its unit to bytes", () => {
  expect(getBytesFromSize(2, "TB")).toBe(2e12);
  expect(getBytesFromSize(1.5, "GB")).toBe(1.5e9);
});

test("reads a byte count back in the unit it was set in", () => {
  expect(getSizeFromBytes(2e12)).toEqual({ amount: 2, unit: "TB" });
  expect(getSizeFromBytes(1.5e9)).toEqual({ amount: 1.5, unit: "GB" });
  expect(getSizeFromBytes(500e6)).toEqual({ amount: 500, unit: "MB" });
});

test("falls back to the smallest unit below its own scale", () => {
  expect(getSizeFromBytes(1_400)).toEqual({ amount: 0.0014, unit: "MB" });
});
