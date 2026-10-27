import { IconType } from "@plakar-ui/ui/components";

import { ContentType } from "@plakar-ui/common/api/enrich";
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
  kind?: ViewerKind;
};

const DOCUMENT_TYPES = ["text", "json", "pdf"];

export function isDocumentContentType(contentType: string) {
  return DOCUMENT_TYPES.includes(ContentType(contentType).resolved);
}

export type ViewerKind = "photo" | "video" | "audio" | "document";

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
    kind: "photo",
  },
  {
    id: "videos",
    name: "Videos",
    icon: "md-video-file",
    mimes: ["video"],
    matches: (contentType) => contentType.startsWith("video/"),
    kind: "video",
  },
  {
    id: "music",
    name: "Music",
    icon: "md-audio-file",
    mimes: ["audio"],
    matches: (contentType) => contentType.startsWith("audio/"),
    kind: "audio",
  },
  {
    id: "documents",
    name: "Documents",
    icon: "md-file-copy",
    mimes: ["text", "application"],
    matches: isDocumentContentType,
    kind: "document",
  },
];

export function viewerKind(contentType: string): ViewerKind | null {
  return (
    BROWSE_VIEWS.find((view) => view.kind && view.matches(contentType))?.kind ??
    null
  );
}

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

/**
 * Lists the views a directory surely has nothing for. Documents only count as
 * empty without any text or application/ file, where PDFs and JSON live.
 */
export function emptyBrowseViews(entry: VFSEntryType): Set<BrowseViewId> {
  const directory = entry.summary?.directory;
  const counts = browseViewCounts(entry);
  const mayHoldDocuments =
    (directory?.MIME_text ?? 0) + (directory?.MIME_application ?? 0) > 0;
  return new Set(
    BROWSE_VIEW_IDS.filter((id) =>
      id === "all"
        ? false
        : id === "documents"
          ? !mayHoldDocuments
          : counts[id] === 0
    )
  );
}

export function searchViewCounts(
  contentTypes: Record<string, number>
): Record<BrowseViewId, number> {
  const matches = Object.entries(contentTypes);
  const count = (id: BrowseViewId) =>
    matches.reduce(
      (sum, [contentType, n]) =>
        getBrowseView(id).matches(contentType) ? sum + n : sum,
      0
    );
  return {
    all: count("all"),
    photos: count("photos"),
    videos: count("videos"),
    music: count("music"),
    documents: count("documents"),
  };
}

export function emptySearchViews(
  counts: Record<BrowseViewId, number>
): Set<BrowseViewId> {
  return new Set(
    BROWSE_VIEW_IDS.filter((id) => id !== "all" && counts[id] === 0)
  );
}

// The share of a directory's files one kind must reach to be suggested, and the
// number of files below which nothing is.
const SUGGEST_SHARE = 0.6;
const SUGGEST_MIN_FILES = 3;

/**
 * Suggests the view matching most of a directory's files when one kind clearly
 * dominates, "all" otherwise.
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
