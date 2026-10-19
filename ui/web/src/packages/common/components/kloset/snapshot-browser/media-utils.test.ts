import { describe, expect, it } from "vitest";

import { formatClock, parseTrackName } from "./media-utils";

describe("media utils", () => {
  it("formats durations like a player", () => {
    expect(formatClock(0)).toBe("0:00");
    expect(formatClock(65.9)).toBe("1:05");
    expect(formatClock(3725)).toBe("1:02:05");
    expect(formatClock(NaN)).toBe("–:––");
  });

  it("splits a trailing tag off track names", () => {
    expect(parseTrackName("song1 (hiphop).mp3")).toEqual({
      title: "song1",
      tag: "hiphop",
    });
    expect(parseTrackName("Ocean noise.wav")).toEqual({
      title: "Ocean noise",
      tag: null,
    });
  });
});
