import { IconType } from "@plakar-ui/ui/components";

import { VFSEntryType } from "@plakar-ui/common/api/kloset";

export const BROWSE_VIEW_IDS = [
  "all",
  "photos",
  "videos",
  "music",
  "documents",
] as const;

export type BrowseViewId = (typeof BROWSE_VIEW_IDS)[number];

type BrowseView = {
  id: BrowseViewId;
  name: string;
  icon: IconType;
  // The content types to search for, as the search endpoint matches them. The
  // endpoint only compares the part before the slash when it does not recurse,
  // so a view also filters what comes back with matches().
  mimes?: string[];
  matches: (contentType: string) => boolean;
};

const DOCUMENT_TYPES = [
  "application/pdf",
  "application/json",
  "application/xml",
  "application/javascript",
  "application/x-yaml",
  "application/yaml",
  "application/toml",
];

/**
 * Says whether a content type is something people read: text, PDFs, and the
 * structured text formats that the content sniffer files under application/.
 */
export function isDocumentContentType(contentType: string) {
  const base = contentType.split(";")[0].trim();
  return base.startsWith("text/") || DOCUMENT_TYPES.includes(base);
}

export const BROWSE_VIEWS: BrowseView[] = [
  {
    id: "all",
    name: "All files",
    icon: "folder",
    matches: () => true,
  },
  {
    id: "photos",
    name: "Photos",
    icon: "md-image",
    mimes: ["image"],
    matches: (contentType) => contentType.startsWith("image/"),
  },
  {
    id: "videos",
    name: "Videos",
    icon: "md-video-file",
    mimes: ["video"],
    matches: (contentType) => contentType.startsWith("video/"),
  },
  {
    id: "music",
    name: "Music",
    icon: "md-audio-file",
    mimes: ["audio"],
    matches: (contentType) => contentType.startsWith("audio/"),
  },
  {
    id: "documents",
    name: "Documents",
    icon: "md-file-copy",
    mimes: ["text", "application"],
    matches: isDocumentContentType,
  },
];

export function getBrowseView(id: BrowseViewId) {
  return BROWSE_VIEWS.find((view) => view.id === id) ?? BROWSE_VIEWS[0];
}

/**
 * Counts the files a directory holds directly, per view, from its summary.
 * Documents are counted from text files only: the summary does not tell PDFs
 * apart from the rest of application/.
 */
export function browseViewCounts(
  entry: VFSEntryType
): Record<BrowseViewId, number> {
  const directory = entry.summary?.directory;
  return {
    all: directory?.files ?? 0,
    photos: directory?.MIME_image ?? 0,
    videos: directory?.MIME_video ?? 0,
    music: directory?.MIME_audio ?? 0,
    documents: directory?.MIME_text ?? 0,
  };
}

// The share of a directory's files one kind must reach to become its default
// view, and the number of files below which "all" always wins.
const SUGGEST_SHARE = 0.6;
const SUGGEST_MIN_FILES = 3;

/**
 * Picks the view a directory opens in: the one matching most of its files when
 * one kind clearly dominates, "all" otherwise.
 */
export function suggestBrowseView(entry: VFSEntryType): BrowseViewId {
  const counts = browseViewCounts(entry);
  if (counts.all < SUGGEST_MIN_FILES) {
    return "all";
  }
  const candidates: BrowseViewId[] = ["photos", "videos", "music"];
  const best = candidates.reduce((a, b) => (counts[b] > counts[a] ? b : a));
  return counts[best] / counts.all >= SUGGEST_SHARE ? best : "all";
}
