import std_path from "path";

/**
 * Formats a duration in seconds as m:ss, or h:mm:ss past an hour.
 */
export function formatClock(seconds: number) {
  if (!Number.isFinite(seconds)) {
    return "–:––";
  }
  const total = Math.floor(seconds);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = String(total % 60).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${s}` : `${m}:${s}`;
}

/**
 * Splits "song1 (hiphop).mp3" into a title and the tag in parentheses, the way
 * many collections name their files.
 */
export function parseTrackName(name: string) {
  const stem = name.slice(0, name.length - std_path.extname(name).length);
  const match = stem.match(/^(.*?)\s*\(([^)]+)\)\s*$/);
  return match
    ? { title: match[1], tag: match[2] }
    : { title: stem, tag: null };
}
