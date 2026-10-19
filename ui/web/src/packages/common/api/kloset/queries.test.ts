import { describe, expect, it } from "vitest";

import { searchNameFilter } from "./queries";

describe("searchNameFilter", () => {
  it("turns a bare word into a substring glob for indexed searches", () => {
    expect(searchNameFilter("rock", { indexed: true })).toBe("*rock*");
  });

  it("keeps globs as typed", () => {
    expect(searchNameFilter("*.jpg", { indexed: true })).toBe("*.jpg");
    expect(searchNameFilter("IMG_10?.jpg", { indexed: true })).toBe(
      "IMG_10?.jpg"
    );
  });

  it("leaves plain searches alone: they match substrings already", () => {
    expect(searchNameFilter("rock", { indexed: false })).toBe("rock");
    expect(searchNameFilter("", { indexed: true })).toBe("");
  });
});
