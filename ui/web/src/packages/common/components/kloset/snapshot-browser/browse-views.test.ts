import { describe, expect, it } from "vitest";

import { VFSEntrySchema } from "@plakar-ui/common/api/kloset";

import {
  browseViewCounts,
  emptyBrowseViews,
  getBrowseView,
  isDocumentContentType,
  suggestBrowseView,
} from "./browse-views";

const emptyCounts = {
  directories: 0,
  files: 0,
  symlinks: 0,
  devices: 0,
  pipes: 0,
  sockets: 0,
  children: 0,
  setuid: 0,
  setgid: 0,
  sticky: 0,
  objects: 0,
  chunks: 0,
  min_size: 0,
  max_size: 0,
  size: 0,
  min_mod_time: 0,
  max_mod_time: 0,
  min_entropy: 0,
  max_entropy: 0,
  hi_entropy: 0,
  lo_entropy: 0,
  MIME_audio: 0,
  MIME_video: 0,
  MIME_image: 0,
  MIME_text: 0,
  MIME_application: 0,
  MIME_other: 0,
  errors: 0,
};

function directory(counts: Partial<typeof emptyCounts>) {
  return VFSEntrySchema.parse({
    version: 65536,
    parent_path: "/",
    file_info: {
      name: "photos",
      size: 0,
      mode: 2147484141,
      mod_time: "2026-10-14T00:11:37Z",
      dev: 0,
      ino: 0,
      uid: 0,
      gid: 0,
      nlink: 1,
      username: "",
      groupname: "",
    },
    summary: {
      directory: {
        ...emptyCounts,
        ...counts,
        avg_size: 0,
        sum_entropy: 0,
        avg_entropy: 0,
      },
      below: emptyCounts,
    },
    mac: "00",
  });
}

describe("browse views", () => {
  it("disables documents only without any text or application file", () => {
    expect(emptyBrowseViews(directory({ files: 6, MIME_audio: 6 }))).toEqual(
      new Set(["photos", "videos", "documents"])
    );
    expect(
      emptyBrowseViews(directory({ files: 2, MIME_application: 2 }))
    ).toEqual(new Set(["photos", "videos", "music"]));
  });

  it("counts the files a directory holds directly", () => {
    const counts = browseViewCounts(
      directory({ files: 10, MIME_image: 6, MIME_audio: 1, MIME_text: 3 })
    );
    expect(counts).toEqual({
      all: 10,
      photos: 6,
      videos: 0,
      music: 1,
      documents: 3,
    });
  });

  it("suggests the view of a dominant kind of media", () => {
    expect(suggestBrowseView(directory({ files: 10, MIME_image: 6 }))).toBe(
      "photos"
    );
    expect(suggestBrowseView(directory({ files: 20, MIME_audio: 20 }))).toBe(
      "music"
    );
  });

  it("suggests a view in directories with subfolders too", () => {
    expect(
      suggestBrowseView(directory({ files: 4, MIME_audio: 4, directories: 4 }))
    ).toBe("music");
  });

  it("keeps all files when no kind dominates", () => {
    expect(
      suggestBrowseView(directory({ files: 10, MIME_image: 5, MIME_video: 5 }))
    ).toBe("all");
  });

  it("keeps all files for tiny directories", () => {
    expect(suggestBrowseView(directory({ files: 2, MIME_image: 2 }))).toBe(
      "all"
    );
  });

  it("never suggests documents, which would take over every source tree", () => {
    expect(suggestBrowseView(directory({ files: 10, MIME_text: 10 }))).toBe(
      "all"
    );
  });

  it("recognizes documents beyond text/", () => {
    expect(isDocumentContentType("text/plain; charset=utf-8")).toBe(true);
    expect(isDocumentContentType("application/pdf")).toBe(true);
    expect(isDocumentContentType("application/json")).toBe(true);
    expect(isDocumentContentType("application/zip")).toBe(false);
    expect(isDocumentContentType("image/png")).toBe(false);
  });

  it("filters search results the endpoint over-matches", () => {
    const documents = getBrowseView("documents");
    expect(documents.matches("application/octet-stream")).toBe(false);
    expect(documents.matches("text/csv")).toBe(true);
  });
});
