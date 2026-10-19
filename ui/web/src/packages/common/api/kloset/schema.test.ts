import { describe, expect, it } from "vitest";

import { VFSEntrySchema } from "./schema";

const fileInfo = {
  name: "a.jpg",
  size: 1024,
  mode: 420,
  mod_time: "2026-10-14T00:11:37Z",
  dev: 0,
  ino: 0,
  uid: 0,
  gid: 0,
  nlink: 1,
  username: "",
  groupname: "",
};

describe("VFSEntrySchema", () => {
  it("reads the content type listings carry at the top level", () => {
    const entry = VFSEntrySchema.parse({
      version: 65536,
      parent_path: "/photos",
      file_info: fileInfo,
      content_type: "image/jpeg",
      entropy: 7.9,
      chunks: 1,
      mac: "00",
    });
    expect(entry.contentType).toBe("image/jpeg");
    expect(entry.entropy).toBe(7.9);
  });

  it("prefers the content type of the resolved object", () => {
    const entry = VFSEntrySchema.parse({
      version: 65536,
      parent_path: "/photos",
      file_info: fileInfo,
      object: {
        version: 65536,
        contentMAC: "00",
        content_type: "image/png",
        entropy: 7.9,
        flags: 0,
      },
      content_type: "image/jpeg",
      mac: "00",
    });
    expect(entry.contentType).toBe("image/png");
  });

  it("leaves the content type empty when nothing tells it", () => {
    const entry = VFSEntrySchema.parse({
      version: 65536,
      parent_path: "/",
      file_info: fileInfo,
      mac: "00",
    });
    expect(entry.contentType).toBe("");
  });
});
