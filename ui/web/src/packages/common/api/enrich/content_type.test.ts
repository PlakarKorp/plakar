import { describe, expect, it } from "vitest";

import { ContentType } from "./content_type";

describe("ContentType", () => {
  it("resolves by media type, ignoring parameters", () => {
    expect(ContentType("text/plain; charset=utf-8").resolved).toBe("text");
    expect(ContentType("application/json").resolved).toBe("json");
    expect(ContentType("application/ld+json").resolved).toBe("json");
    expect(ContentType("image/png").resolved).toBe("image");
    expect(ContentType("application/pdf").resolved).toBe("pdf");
    expect(ContentType("video/mp4").resolved).toBe("video");
    expect(ContentType("audio/mpeg").resolved).toBe("audio");
    expect(ContentType("application/xml").resolved).toBe("text");
  });

  it("does not match on a substring of the type", () => {
    expect(
      ContentType("application/vnd.oasis.opendocument.text").resolved
    ).toBe("unknown");
    expect(ContentType("application/x-iso9660-image").resolved).toBe("unknown");
    expect(ContentType("application/x-videomail").resolved).toBe("unknown");
    expect(ContentType("application/jsonl-ish").resolved).toBe("unknown");
    expect(ContentType("application/octet-stream").resolved).toBe("unknown");
  });
});
