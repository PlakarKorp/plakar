import { Button, Icon, IconButton, Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { useEffect, useRef, useState } from "react";
import { Dialog, Modal, ModalOverlay } from "react-aria-components";

import {
  EnrichedChildEntryType,
  SnapshotType,
} from "@plakar-ui/common/api/kloset";
import { FileIcon, FilePreview } from "@plakar-ui/common/components";
import { useInView, useLocalStorage } from "@plakar-ui/common/hooks";

import { useEntryURL } from "./use-entry-url";

const SLIDESHOW_INTERVAL = 3500;

type LightboxProps = {
  snapshot: SnapshotType;
  source: number;
  items: EnrichedChildEntryType[];
  index: number;
  onIndexChange: (index: number) => void;
  onClose: () => void;
};

/**
 * Full-screen viewer to page through the photos, videos or documents of a
 * directory, with a filmstrip, a details panel and a slideshow for photos.
 */
export function Lightbox({
  snapshot,
  source,
  items,
  index,
  onIndexChange,
  onClose,
}: LightboxProps) {
  const entry = items[index];
  const [showDetails, setShowDetails] = useLocalStorage(
    "lightboxDetails",
    false
  );
  const [playing, setPlaying] = useState(false);
  const isPhoto = entry?.contentType.startsWith("image/") ?? false;

  const go = (delta: number) => {
    if (items.length === 0) return;
    onIndexChange((index + delta + items.length) % items.length);
  };

  useEffect(() => {
    if (!playing || !isPhoto) return;
    const timer = setInterval(
      () => onIndexChange((index + 1) % items.length),
      SLIDESHOW_INTERVAL
    );
    return () => clearInterval(timer);
  }, [playing, isPhoto, index, items.length, onIndexChange]);

  // Dialog does not take key handlers: listen on the document while open.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Leave the keys to the video and form controls that use them.
      if (
        e.target instanceof HTMLVideoElement ||
        e.target instanceof HTMLInputElement
      ) {
        return;
      }
      const count = items.length;
      if (e.key === "ArrowRight" && count) onIndexChange((index + 1) % count);
      else if (e.key === "ArrowLeft" && count)
        onIndexChange((index - 1 + count) % count);
      else if (e.key === "i") setShowDetails((v) => !v);
      else if (e.key === " " && isPhoto) {
        e.preventDefault();
        setPlaying((v) => !v);
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [index, items.length, isPhoto, onIndexChange, setShowDetails]);

  if (!entry) return null;

  return (
    <ModalOverlay
      isOpen
      isDismissable
      onOpenChange={(open) => !open && onClose()}
      className="fixed inset-0 z-50 bg-zinc-950 data-[entering]:animate-[modal-fade_200ms]"
    >
      <Modal className="h-full w-full">
        <Dialog
          aria-label={`Viewer: ${entry.file_info.name}`}
          className="grid h-full w-full grid-rows-[auto_1fr_auto] text-white outline-none"
        >
          <header className="flex items-center gap-3 px-3 py-2">
            <IconButton
              icon="close"
              aria-label="Close"
              variant="tertiary"
              className="text-white"
              onPress={onClose}
            />
            <div className="min-w-0 flex-1">
              <Typography weight="semibold" className="truncate text-white">
                {entry.file_info.name}
              </Typography>
              <Typography size="xs" className="text-slate-400 tabular-nums">
                {index + 1} / {items.length} · {entry.file_info.size.Pretty}
              </Typography>
            </div>
            {isPhoto && items.length > 1 && (
              <IconButton
                icon={playing ? "pause" : "play"}
                aria-label={playing ? "Stop the slideshow" : "Play a slideshow"}
                variant="tertiary"
                className="text-white"
                onPress={() => setPlaying((v) => !v)}
              />
            )}
            <DownloadButton snapshot={snapshot} source={source} entry={entry} />
            <IconButton
              icon="info-circle"
              aria-label="Details"
              variant="tertiary"
              className={clsx("text-white", showDetails && "bg-white/15")}
              onPress={() => setShowDetails((v) => !v)}
            />
          </header>

          <div
            className={clsx(
              "relative grid min-h-0",
              showDetails ? "md:grid-cols-[1fr_320px]" : "grid-cols-1"
            )}
          >
            <div className="relative grid min-h-0 min-w-0 place-items-center px-2 md:px-16">
              <Stage
                key={entry.path}
                snapshot={snapshot}
                source={source}
                entry={entry}
                onEnded={() => index < items.length - 1 && go(1)}
              />
              {items.length > 1 && (
                <>
                  <NavButton side="left" onPress={() => go(-1)} />
                  <NavButton side="right" onPress={() => go(1)} />
                </>
              )}
            </div>
            {showDetails && <DetailsAside entry={entry} />}
          </div>

          {items.length > 1 && (
            <Filmstrip
              snapshot={snapshot.identifier.Value}
              source={source}
              items={items}
              index={index}
              onSelect={onIndexChange}
            />
          )}
        </Dialog>
      </Modal>
    </ModalOverlay>
  );
}

function NavButton({
  side,
  onPress,
}: {
  side: "left" | "right";
  onPress: () => void;
}) {
  // The button styles position it relatively: a wrapper takes it out of flow.
  return (
    <div
      className={clsx(
        "absolute top-1/2 hidden -translate-y-1/2 md:block",
        side === "left" ? "left-3" : "right-3"
      )}
    >
      <IconButton
        icon={side === "left" ? "chevron-left" : "chevron-right"}
        aria-label={side === "left" ? "Previous" : "Next"}
        variant="tertiary"
        onPress={onPress}
        className="rounded-full bg-white/10 text-white"
      />
    </div>
  );
}

type StageProps = {
  snapshot: SnapshotType;
  source: number;
  entry: EnrichedChildEntryType;
  onEnded: () => void;
};

function Stage({ snapshot, source, entry, onEnded }: StageProps) {
  const contentType = entry.contentType;
  if (contentType.startsWith("image/")) {
    return <PhotoStage snapshot={snapshot} source={source} entry={entry} />;
  }
  if (contentType.startsWith("video/")) {
    return (
      <VideoStage
        snapshot={snapshot}
        source={source}
        entry={entry}
        onEnded={onEnded}
      />
    );
  }
  // Documents reuse the side panel preview: it knows how to render code,
  // text, markdown and PDFs. It sits on a sheet, light in both themes.
  return (
    <div className="bg-primary text-primary flex h-[calc(100vh-11rem)] w-full max-w-5xl flex-col overflow-hidden rounded-lg">
      <FilePreview snapshot={snapshot} source={source} entry={entry} />
    </div>
  );
}

function PhotoStage({ snapshot, source, entry }: Omit<StageProps, "onEnded">) {
  const { url, isColdStorage } = useEntryURL({
    snapshot: snapshot.identifier.Value,
    source,
    path: entry.path,
  });
  const [zoom, setZoom] = useState(false);
  const [failed, setFailed] = useState(false);

  if (isColdStorage || failed) {
    return (
      <StageNotice
        title={
          isColdStorage
            ? "This file is in cold storage"
            : "This browser cannot display this image format"
        }
        entry={entry}
        snapshot={snapshot}
        source={source}
      />
    );
  }
  if (!url) {
    return <Icon type="spinner" size="2xl" className="animate-spin" />;
  }
  return (
    <div className={clsx("max-h-full max-w-full", zoom && "overflow-auto")}>
      <img
        src={url}
        alt={entry.file_info.name}
        onClick={() => setZoom((v) => !v)}
        onError={() => setFailed(true)}
        className={clsx(
          "rounded shadow-2xl motion-safe:animate-[modal-fade_200ms]",
          zoom
            ? "max-w-none cursor-zoom-out"
            : "max-h-[calc(100vh-11rem)] max-w-full cursor-zoom-in object-contain"
        )}
      />
    </div>
  );
}

function VideoStage({ snapshot, source, entry, onEnded }: StageProps) {
  const { url, isColdStorage } = useEntryURL({
    snapshot: snapshot.identifier.Value,
    source,
    path: entry.path,
  });
  if (isColdStorage) {
    return (
      <StageNotice
        title="This file is in cold storage"
        entry={entry}
        snapshot={snapshot}
        source={source}
      />
    );
  }
  if (!url) {
    return <Icon type="spinner" size="2xl" className="animate-spin" />;
  }
  return (
    <video
      src={url}
      controls
      autoPlay
      playsInline
      onEnded={onEnded}
      className="max-h-[calc(100vh-11rem)] max-w-full rounded-md bg-black shadow-2xl"
    />
  );
}

function StageNotice({
  title,
  snapshot,
  source,
  entry,
}: {
  title: string;
  snapshot: SnapshotType;
  source: number;
  entry: EnrichedChildEntryType;
}) {
  const { url } = useEntryURL({
    snapshot: snapshot.identifier.Value,
    source,
    path: entry.path,
    params: { download: true },
  });
  return (
    <div className="flex max-w-sm flex-col items-center gap-3 text-center">
      <FileIcon contentType={entry.contentType} size="3xl" />
      <Typography weight="semibold" className="text-white">
        {title}
      </Typography>
      <Typography size="sm" className="text-slate-400">
        It is safely backed up: download it to open it.
      </Typography>
      {url && (
        <Button href={url} download variant="secondary" leftIcon="download">
          Download
        </Button>
      )}
    </div>
  );
}

function DownloadButton({
  snapshot,
  source,
  entry,
}: {
  snapshot: SnapshotType;
  source: number;
  entry: EnrichedChildEntryType;
}) {
  const { url } = useEntryURL({
    snapshot: snapshot.identifier.Value,
    source,
    path: entry.path,
    params: { download: true },
  });
  return (
    <IconButton
      icon="download"
      aria-label="Download"
      variant="tertiary"
      className="text-white"
      href={url}
      download
      isDisabled={!url}
    />
  );
}

function DetailsAside({ entry }: { entry: EnrichedChildEntryType }) {
  const rows: [string, string][] = [
    ["Name", entry.file_info.name],
    ["Type", entry.contentType || "unknown"],
    [
      "Size",
      `${entry.file_info.size.Pretty} (${entry.file_info.size.Value} bytes)`,
    ],
    ["Modified", new Date(entry.file_info.mod_time.Value).toLocaleString()],
    [
      "Owner",
      `${entry.file_info.username || entry.file_info.uid}:${entry.file_info.groupname || entry.file_info.gid}`,
    ],
    ["Mode", entry.file_info.mode.toString()],
    ["Path", entry.path],
  ];
  if (entry.entropy !== undefined) {
    rows.splice(3, 0, ["Entropy", `${entry.entropy.toFixed(2)} bits/byte`]);
  }
  return (
    <aside className="hidden overflow-y-auto border-l border-white/10 px-5 py-2 md:block">
      <Typography weight="semibold" className="mb-3 text-white">
        Details
      </Typography>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
        {rows.map(([label, value]) => (
          <div key={label} className="contents">
            <Typography as="dt" size="sm" className="text-slate-400">
              {label}
            </Typography>
            <Typography as="dd" size="sm" className="break-all text-white">
              {value}
            </Typography>
          </div>
        ))}
      </dl>
    </aside>
  );
}

function Filmstrip({
  snapshot,
  source,
  items,
  index,
  onSelect,
}: {
  snapshot: string;
  source: number;
  items: EnrichedChildEntryType[];
  index: number;
  onSelect: (index: number) => void;
}) {
  const stripRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const current = stripRef.current?.children[index];
    current?.scrollIntoView({
      block: "nearest",
      inline: "center",
      behavior: "smooth",
    });
  }, [index]);

  return (
    <div
      ref={stripRef}
      className="flex gap-1.5 overflow-x-auto px-3 pt-2 pb-3"
      aria-label="Filmstrip"
    >
      {items.map((item, i) => (
        <FilmstripItem
          key={item.path}
          snapshot={snapshot}
          source={source}
          entry={item}
          isCurrent={i === index}
          onPress={() => onSelect(i)}
        />
      ))}
    </div>
  );
}

function FilmstripItem({
  snapshot,
  source,
  entry,
  isCurrent,
  onPress,
}: {
  snapshot: string;
  source: number;
  entry: EnrichedChildEntryType;
  isCurrent: boolean;
  onPress: () => void;
}) {
  const [ref, inView] = useInView<HTMLButtonElement>("200px");
  const isPhoto = entry.contentType.startsWith("image/");
  const { url } = useEntryURL({
    snapshot,
    source,
    path: entry.path,
    enabled: inView && isPhoto,
  });

  return (
    <button
      ref={ref}
      type="button"
      onClick={onPress}
      aria-label={entry.file_info.name}
      aria-current={isCurrent}
      className={clsx(
        "flex h-12 flex-none cursor-pointer items-center justify-center gap-1 overflow-hidden rounded-md border-2 bg-white/10 text-xs text-slate-300",
        "motion-safe:transition-opacity",
        isPhoto ? "w-16" : "w-32 justify-start px-2",
        isCurrent
          ? "border-accent opacity-100"
          : "border-transparent opacity-60 hover:opacity-90"
      )}
    >
      {isPhoto && url ? (
        <img src={url} alt="" className="h-full w-full object-cover" />
      ) : (
        <>
          <FileIcon contentType={entry.contentType} size="sm" />
          {!isPhoto && <span className="truncate">{entry.file_info.name}</span>}
        </>
      )}
    </button>
  );
}
