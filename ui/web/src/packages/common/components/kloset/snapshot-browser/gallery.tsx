import { Checkbox, Icon, Typography } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import clsx from "clsx";
import std_path from "path";
import { ReactNode, useState } from "react";
import { Button } from "react-aria-components";

import {
  EnrichedChildEntryType,
  SnapshotFileHeadQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { FileIcon } from "@plakar-ui/common/components";
import { useInView } from "@plakar-ui/common/hooks";

import { BrowseViewId, isDocumentContentType } from "./browse-views";
import { formatClock } from "./media-utils";
import { useEntryURL } from "./use-entry-url";

export type GalleryDensity = "small" | "medium" | "large";

const TILE_WIDTH: Record<GalleryDensity, string> = {
  small: "140px",
  medium: "200px",
  large: "300px",
};

type GalleryProps = {
  snapshot: string;
  source: number;
  view: BrowseViewId;
  entries: EnrichedChildEntryType[];
  // The directory being browsed: files found below it by a search show the
  // folder they come from.
  basePath: string;
  density: GalleryDensity;
  selected: ReadonlySet<string>;
  onToggleSelected: (entry: EnrichedChildEntryType) => void;
  onOpen: (entry: EnrichedChildEntryType) => void;
  onOpenDirectory: (path: string) => void;
};

/**
 * Shows the entries of a directory as tiles: folders first, then files with a
 * preview fitting their kind.
 */
export function Gallery({
  snapshot,
  source,
  view,
  entries,
  basePath,
  density,
  selected,
  onToggleSelected,
  onOpen,
  onOpenDirectory,
}: GalleryProps) {
  const directories = entries.filter(
    (entry) => entry.file_info.mode.isDir() && entry.file_info.name !== ".."
  );
  const files = entries.filter((entry) => !entry.file_info.mode.isDir());

  return (
    <div className="flex flex-col gap-6">
      {directories.length > 0 && (
        <GallerySection title="Folders" count={directories.length}>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-2">
            {directories.map((entry) => (
              <FolderTile
                key={entry.path}
                entry={entry}
                onPress={() => onOpenDirectory(entry.path)}
              />
            ))}
          </div>
        </GallerySection>
      )}

      {files.length > 0 && (
        <GallerySection title="Files" count={files.length}>
          <div
            className="grid gap-3"
            style={{
              gridTemplateColumns: `repeat(auto-fill, minmax(min(${TILE_WIDTH[density]}, 100%), 1fr))`,
            }}
          >
            {files.map((entry) => (
              <FileTile
                key={entry.path}
                snapshot={snapshot}
                source={source}
                view={view}
                entry={entry}
                folder={std_path.relative(basePath, entry.parent_path)}
                isSelected={selected.has(entry.path)}
                onToggleSelected={() => onToggleSelected(entry)}
                onPress={() => onOpen(entry)}
              />
            ))}
          </div>
        </GallerySection>
      )}
    </div>
  );
}

function GallerySection({
  title,
  count,
  children,
}: {
  title: string;
  count: number;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-2">
      <Typography
        as="h3"
        size="xs"
        weight="semibold"
        className="text-tertiary tracking-wide uppercase"
      >
        {title} · {count}
      </Typography>
      {children}
    </section>
  );
}

function FolderTile({
  entry,
  onPress,
}: {
  entry: EnrichedChildEntryType;
  onPress: () => void;
}) {
  return (
    <Button
      onPress={onPress}
      className={clsx(
        "border-secondary bg-secondary flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 text-left",
        "data-hovered:bg-tertiary data-focus-visible:outline-focus data-focus-visible:outline-2",
        "duration-150 motion-safe:transition-colors"
      )}
    >
      <Icon type="md-folder" size="xl" className="text-accent" />
      <span className="flex min-w-0 flex-col">
        <Typography size="sm" weight="medium" className="truncate">
          {entry.file_info.name}
        </Typography>
        <Typography size="xs" className="text-tertiary">
          {entry.summary?.directory.children ?? 0} items ·{" "}
          {entry.summary?.Total.size.Pretty}
        </Typography>
      </span>
    </Button>
  );
}

type FileTileProps = {
  snapshot: string;
  source: number;
  view: BrowseViewId;
  entry: EnrichedChildEntryType;
  folder: string;
  isSelected: boolean;
  onToggleSelected: () => void;
  onPress: () => void;
};

function FileTile({
  snapshot,
  source,
  view,
  entry,
  folder,
  isSelected,
  onToggleSelected,
  onPress,
}: FileTileProps) {
  const [ref, inView] = useInView<HTMLDivElement>();
  const contentType = entry.contentType;
  const kind = contentType.startsWith("image/")
    ? "photo"
    : contentType.startsWith("video/")
      ? "video"
      : isDocumentContentType(contentType)
        ? "document"
        : "other";

  // In the photos view the picture is the whole tile; elsewhere tiles share a
  // shape so mixed folders line up.
  const aspect =
    view === "photos"
      ? "aspect-square"
      : view === "videos"
        ? "aspect-video"
        : view === "documents"
          ? "aspect-[3/3.4]"
          : "aspect-[4/3]";

  return (
    <div
      ref={ref}
      className={clsx(
        "group bg-secondary relative flex flex-col overflow-hidden rounded-xl border",
        "duration-150 hover:-translate-y-0.5 motion-safe:transition-[border-color,box-shadow,transform]",
        isSelected
          ? "border-accent outline-accent-secondary outline-2"
          : "border-secondary hover:border-primary"
      )}
    >
      <div
        className={clsx(
          "absolute top-2 left-2 z-10 rounded-md bg-black/40 p-1 backdrop-blur-sm",
          isSelected ? "opacity-100" : "opacity-0 group-hover:opacity-100",
          "focus-within:opacity-100 motion-safe:transition-opacity"
        )}
      >
        <Checkbox
          size="sm"
          aria-label={`Select ${entry.file_info.name}`}
          isSelected={isSelected}
          onChange={onToggleSelected}
        />
      </div>

      <Button
        onPress={onPress}
        aria-label={`Open ${entry.file_info.name}`}
        className="data-focus-visible:outline-focus flex cursor-pointer flex-col text-left data-focus-visible:outline-2"
      >
        <div
          className={clsx(
            "bg-tertiary relative grid w-full place-items-center overflow-hidden",
            aspect
          )}
        >
          {kind === "photo" ? (
            <PhotoThumbnail
              snapshot={snapshot}
              source={source}
              entry={entry}
              enabled={inView}
            />
          ) : kind === "video" ? (
            <VideoThumbnail
              snapshot={snapshot}
              source={source}
              entry={entry}
              enabled={inView}
            />
          ) : kind === "document" ? (
            <DocumentThumbnail
              snapshot={snapshot}
              source={source}
              entry={entry}
              enabled={inView}
            />
          ) : (
            <GenericThumbnail entry={entry} />
          )}
        </div>
        <div className="border-secondary flex min-w-0 items-center gap-1.5 border-t px-2.5 py-2">
          <FileIcon contentType={contentType} size="sm" />
          <Typography size="xs" weight="medium" className="flex-1 truncate">
            {entry.file_info.name}
          </Typography>
          <Typography size="xs" className="text-tertiary whitespace-nowrap">
            {entry.file_info.size.Pretty}
          </Typography>
        </div>
        {folder && (
          <Typography
            size="xs"
            className="text-tertiary -mt-1.5 truncate px-2.5 pb-2"
            title={folder}
          >
            in {folder}
          </Typography>
        )}
      </Button>
    </div>
  );
}

type ThumbnailProps = {
  snapshot: string;
  source: number;
  entry: EnrichedChildEntryType;
  enabled: boolean;
};

function PhotoThumbnail({ snapshot, source, entry, enabled }: ThumbnailProps) {
  const { url, isColdStorage } = useEntryURL({
    snapshot,
    source,
    path: entry.path,
    enabled,
  });
  const [state, setState] = useState<"loading" | "loaded" | "unsupported">(
    "loading"
  );

  if (isColdStorage) {
    return <ThumbnailNotice icon="snowflake" label="Cold storage" />;
  }
  if (state === "unsupported") {
    return (
      <ThumbnailNotice
        icon="md-image"
        label={`${extension(entry)} · no browser preview`}
      />
    );
  }
  return (
    <>
      {state === "loading" && (
        <Icon type="md-image" size="2xl" className="text-quaternary" />
      )}
      {url && (
        <img
          src={url}
          alt={entry.file_info.name}
          loading="lazy"
          decoding="async"
          onLoad={() => setState("loaded")}
          onError={() => setState("unsupported")}
          className={clsx(
            "absolute inset-0 h-full w-full object-cover motion-safe:transition-[opacity,transform] motion-safe:duration-300",
            "group-hover:scale-[1.03]",
            state === "loaded" ? "opacity-100" : "opacity-0"
          )}
        />
      )}
    </>
  );
}

function VideoThumbnail({ snapshot, source, entry, enabled }: ThumbnailProps) {
  const { url, isColdStorage } = useEntryURL({
    snapshot,
    source,
    path: entry.path,
    enabled,
  });
  const [duration, setDuration] = useState<number>();

  if (isColdStorage) {
    return <ThumbnailNotice icon="snowflake" label="Cold storage" />;
  }
  return (
    <>
      <Icon type="md-video-file" size="2xl" className="text-quaternary" />
      {url && (
        // Muted and paused on its first frames: hovering plays a silent preview.
        <video
          src={`${url}#t=0.8`}
          muted
          playsInline
          loop
          preload="metadata"
          onLoadedMetadata={(e) => setDuration(e.currentTarget.duration)}
          onMouseEnter={(e) => void e.currentTarget.play().catch(() => {})}
          onMouseLeave={(e) => {
            e.currentTarget.pause();
            e.currentTarget.currentTime = 0.8;
          }}
          className="absolute inset-0 h-full w-full bg-black object-cover"
        />
      )}
      {duration !== undefined && (
        <span className="absolute right-2 bottom-2 flex items-center gap-1 rounded-md bg-black/60 px-1.5 py-0.5 text-white">
          <Icon type="play" size="xs" />
          <Typography
            as="span"
            size="xs"
            weight="medium"
            className="tabular-nums"
          >
            {formatClock(duration)}
          </Typography>
        </span>
      )}
    </>
  );
}

function DocumentThumbnail({
  snapshot,
  source,
  entry,
  enabled,
}: ThumbnailProps) {
  const { api, repository } = useKlosetAPI();
  const isPDF = entry.contentType.startsWith("application/pdf");
  const { data } = useQuery({
    ...SnapshotFileHeadQueryOptions(api, {
      repository,
      snapshot,
      source,
      path: entry.path,
      bytes: 1500,
    }),
    enabled: enabled && !isPDF,
  });

  return (
    // A sheet of paper, white in both themes, with the first lines of the file.
    <div className="absolute inset-x-3 top-3 bottom-0 overflow-hidden rounded-t-md bg-white px-3 pt-3 shadow-md">
      {isPDF ? (
        <>
          <span className="rounded bg-red-600 px-1.5 py-0.5 text-[10px] font-bold tracking-wider text-white">
            PDF
          </span>
          <PaperLines />
        </>
      ) : data ? (
        <pre className="font-mono text-[7.5px] leading-[1.35] break-all whitespace-pre-wrap text-slate-700">
          {data}
        </pre>
      ) : (
        <PaperLines />
      )}
      <span className="absolute right-2 bottom-2 rounded bg-black/60 px-1.5 py-0.5 text-[10px] font-semibold text-white">
        {extension(entry)}
      </span>
    </div>
  );
}

function PaperLines() {
  const widths = [
    "w-11/12",
    "w-9/12",
    "w-10/12",
    "w-7/12",
    "w-11/12",
    "w-8/12",
  ];
  return (
    <div className="mt-3 flex flex-col gap-2">
      {widths.map((width, i) => (
        <span key={i} className={clsx("h-1.5 rounded bg-slate-200", width)} />
      ))}
    </div>
  );
}

function GenericThumbnail({ entry }: { entry: EnrichedChildEntryType }) {
  return <ThumbnailNotice icon="md-file-copy" label={extension(entry)} />;
}

function ThumbnailNotice({
  icon,
  label,
}: {
  icon: "md-image" | "md-file-copy" | "snowflake";
  label: string;
}) {
  return (
    <div className="text-tertiary flex flex-col items-center gap-1.5 p-2 text-center">
      <Icon type={icon} size="2xl" />
      <Typography size="xs" weight="semibold" className="tracking-wide">
        {label}
      </Typography>
    </div>
  );
}

function extension(entry: EnrichedChildEntryType) {
  return (
    std_path.extname(entry.file_info.name).slice(1).toUpperCase() || "FILE"
  );
}
