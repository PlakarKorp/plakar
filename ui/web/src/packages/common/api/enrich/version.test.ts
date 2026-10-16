import { expect, test } from "vitest";

import { Version } from "./version";

test("parses simple version strings correctly", () => {
  expect(Version("1.2.3")).toStrictEqual({
    Value: "1.2.3",
    Pretty: "1.2.3",
    Channel: "1.2.3",
  });

  expect(Version("v1.2.3")).toStrictEqual({
    Value: "v1.2.3",
    Pretty: "v1.2.3",
    Channel: "v1.2.3",
  });
});

test("parses unexpected version", () => {
  expect(Version("devel1234567890")).toStrictEqual({
    Value: "devel1234567890",
    Pretty: "devel1234567890",
    Channel: "devel1234567890",
  });
});

test("parses dev version strings correctly", () => {
  expect(Version("1.2.3-rc.1.abc1234567890")).toStrictEqual({
    Value: "1.2.3-rc.1.abc1234567890",
    Pretty: "1.2.3-rc.1.abc1234",
    Channel: "1.2.3-rc.1",
  });

  expect(Version("1.2.3-rc1.abc1234567890")).toStrictEqual({
    Value: "1.2.3-rc1.abc1234567890",
    Pretty: "1.2.3-rc1.abc1234",
    Channel: "1.2.3-rc1",
  });

  expect(Version("v1.2.3-rc.1.abc1234567890")).toStrictEqual({
    Value: "v1.2.3-rc.1.abc1234567890",
    Pretty: "v1.2.3-rc.1.abc1234",
    Channel: "v1.2.3-rc.1",
  });

  expect(Version("v1.2.3-rc.1.abc1234567890")).toStrictEqual({
    Value: "v1.2.3-rc.1.abc1234567890",
    Pretty: "v1.2.3-rc.1.abc1234",
    Channel: "v1.2.3-rc.1",
  });

  expect(Version("v1.2.3-rc")).toStrictEqual({
    Value: "v1.2.3-rc",
    Pretty: "v1.2.3-rc",
    Channel: "v1.2.3-rc",
  });

  expect(Version("v1.2.3-rc.1")).toStrictEqual({
    Value: "v1.2.3-rc.1",
    Pretty: "v1.2.3-rc.1",
    Channel: "v1.2.3-rc.1",
  });
});
