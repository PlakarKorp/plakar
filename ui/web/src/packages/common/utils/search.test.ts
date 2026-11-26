import { describe, expect, it } from "vitest";

import { foldSearchText, matchSearchToken, parseSearchQuery, rankSearchables, scoreSearchable } from "./search";

const item = (id: string, title: string, extra = {}) => ({
  id,
  title,
  ...extra,
});

describe("foldSearchText", () => {
  it("strips accents and keeps the offsets of the original", () => {
    expect(foldSearchText("Événement")).toEqual({
      text: "evenement",
      offsets: [0, 1, 2, 3, 4, 5, 6, 7, 8],
    });
  });
});

describe("matchSearchToken", () => {
  it("prefers an exact match over a prefix", () => {
    expect(matchSearchToken("stores", "Stores")?.score).toBeGreaterThan(
      matchSearchToken("stor", "Stores")!.score
    );
  });

  it("marks the start of a later word", () => {
    expect(matchSearchToken("prov", "Secret providers")?.ranges).toEqual([[7, 11]]);
  });

  it("matches initials", () => {
    expect(matchSearchToken("sp", "Secret providers")?.ranges).toEqual([
      [0, 1],
      [7, 8],
    ]);
  });

  it("matches across camel case and separators", () => {
    expect(matchSearchToken("db", "postgres-db-eu")?.ranges).toEqual([[9, 11]]);
    expect(matchSearchToken("bucket", "myBucketName")?.ranges).toEqual([[2, 8]]);
  });

  it("marks accented originals", () => {
    expect(matchSearchToken("ete", "Été 2026")?.ranges).toEqual([[0, 3]]);
  });

  it("tolerates a dropped letter but not a scattered one", () => {
    expect(matchSearchToken("schdul", "Scheduling")).not.toBeNull();
    expect(matchSearchToken("sng", "Secret providers listing")).toBeNull();
  });
});

describe("scoreSearchable", () => {
  it("needs every token to match", () => {
    const store = item("1", "s3-prod-eu", { subtitle: "Store" });
    expect(scoreSearchable(store, ["prod", "eu"])).not.toBeNull();
    expect(scoreSearchable(store, ["prod", "us"])).toBeNull();
  });

  it("weighs the title over the subtitle", () => {
    const byTitle = scoreSearchable(item("1", "Backup"), ["backup"])!;
    const bySubtitle = scoreSearchable(item("2", "Nightly", { subtitle: "Backup" }), [
      "backup",
    ])!;
    expect(byTitle.score).toBeGreaterThan(bySubtitle.score);
  });

  it("finds an item by a keyword", () => {
    const page = item("1", "Policy", { keywords: ["SLA"] });
    expect(scoreSearchable(page, ["sla"])?.titleRanges).toEqual([]);
  });
});

describe("rankSearchables", () => {
  const items = [
    item("restore", "Restore"),
    item("restore-points", "Restore points"),
    item("stores", "Stores"),
  ];

  it("ranks the closest title first", () => {
    expect(rankSearchables(items, "stor").map((hit) => hit.item.id)).toEqual([
      "stores",
      "restore",
      "restore-points",
    ]);
  });

  it("keeps the order without a query, recent items first", () => {
    expect(
      rankSearchables(items, "", { recentIds: ["stores"] }).map((hit) => hit.item.id)
    ).toEqual(["stores", "restore", "restore-points"]);
  });

  it("keeps the given order of equal titles without a query", () => {
    const given = [item("b", "Sign out"), item("a", "Add a user")];
    expect(rankSearchables(given, "").map((hit) => hit.item.id)).toEqual(["b", "a"]);
  });

  it("lets a recent item win a near tie", () => {
    expect(
      rankSearchables(items, "rest", { recentIds: ["restore-points"] })[0].item.id
    ).toBe("restore-points");
  });
});

describe("parseSearchQuery", () => {
  const aliases = { store: "store", member: "member" } as const;

  it("reads an alias ahead of the query", () => {
    expect(parseSearchQuery("store: prod", aliases, "action")).toEqual({
      scope: "store",
      text: "prod",
    });
  });

  it("reads a leading chevron as the actions", () => {
    expect(parseSearchQuery(">theme", aliases, "action")).toEqual({
      scope: "action",
      text: "theme",
    });
  });

  it("leaves an unknown prefix in the text", () => {
    expect(parseSearchQuery("s3://bucket", aliases, "action")).toEqual({
      text: "s3://bucket",
    });
  });
});
