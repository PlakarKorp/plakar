import { describe, expect, it } from "vitest";

import { diffHunks, diffLines } from "./diff";

describe("diffLines", () => {
  it("returns only equal lines for identical inputs", () => {
    const lines = diffLines(["a", "b"], ["a", "b"]);
    expect(lines?.map((l) => l.op)).toEqual(["equal", "equal"]);
  });

  it("finds insertions and deletions with their line numbers", () => {
    expect(diffLines(["a", "b", "c"], ["a", "x", "c", "d"])).toEqual([
      { op: "equal", text: "a", a: 1, b: 1 },
      { op: "delete", text: "b", a: 2 },
      { op: "insert", text: "x", b: 2 },
      { op: "equal", text: "c", a: 3, b: 3 },
      { op: "insert", text: "d", b: 4 },
    ]);
  });

  it("gives up past the edit limit", () => {
    expect(diffLines(["a", "b", "c"], ["x", "y", "z"], 5)).toBeNull();
    expect(diffLines(["a", "b", "c"], ["x", "y", "z"], 6)).toHaveLength(6);
  });

  it("handles empty sides", () => {
    expect(diffLines([], ["a"])).toEqual([{ op: "insert", text: "a", b: 1 }]);
    expect(diffLines(["a"], [])).toEqual([{ op: "delete", text: "a", a: 1 }]);
  });
});

describe("diffHunks", () => {
  it("folds unchanged runs outside the context", () => {
    const a = Array.from({ length: 20 }, (_, i) => `line ${i}`);
    const b = [...a];
    b[10] = "changed";
    const hunks = diffHunks(diffLines(a, b) ?? [], 2);
    expect(hunks[0]).toEqual({ op: "skip", count: 8 });
    expect(hunks.at(-1)).toEqual({ op: "skip", count: 7 });
    expect(hunks.filter((h) => h.op !== "skip")).toHaveLength(6);
  });
});
