import { expect, test } from "vitest";

import { readBlobDownload } from "./download";

function respondWith(contentDisposition?: string) {
  return new Response("payload", {
    headers: contentDisposition
      ? { "Content-Disposition": contentDisposition }
      : {},
  });
}

test("keeps the quoted filename the server chose", async () => {
  const { filename } = await readBlobDownload(
    respondWith('attachment; filename="audit-log.csv"'),
    "fallback.csv"
  );

  expect(filename).toBe("audit-log.csv");
});

test("keeps a filename left unquoted", async () => {
  const { filename } = await readBlobDownload(
    respondWith("attachment; filename=audit-log.csv"),
    "fallback.csv"
  );

  expect(filename).toBe("audit-log.csv");
});

test("prefers the RFC 5987 encoding when both are offered", async () => {
  const { filename } = await readBlobDownload(
    respondWith(
      "attachment; filename=\"requisition.csv\"; filename*=UTF-8''r%C3%A9quisition.csv"
    ),
    "fallback.csv"
  );

  expect(filename).toBe("réquisition.csv");
});

test("falls back to the name the caller asked for", async () => {
  expect((await readBlobDownload(respondWith(), "fallback.csv")).filename).toBe(
    "fallback.csv"
  );
  expect(
    (await readBlobDownload(respondWith("attachment"), "fallback.csv")).filename
  ).toBe("fallback.csv");
});

test("hands back the body alongside the name", async () => {
  const { blob } = await readBlobDownload(respondWith(), "fallback.csv");

  await expect(blob.text()).resolves.toBe("payload");
});
