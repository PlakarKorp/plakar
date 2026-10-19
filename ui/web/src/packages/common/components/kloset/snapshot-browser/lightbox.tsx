import { IconButton, Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { useEffect, useRef, useState } from "react";
import { Button, Dialog, Modal, ModalOverlay } from "react-aria-components";

import {
  EnrichedChildEntryType,
  SnapshotType,
} from "@plakar-ui/common/api/kloset";
import { FileIcon } from "@plakar-ui/common/components";
import { useInView, useLocalStorage } from "@plakar-ui/common/hooks";

import { viewerKind } from "./browse-views";
import { useEntryURL } from "./use-entry-url";
import { EntryViewer } from "./viewer";

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
 * Full-screen viewer to page through the photos, videos, songs or documents
 * of a directory, with a filmstrip, a details panel and a slideshow for photos.
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
  const isPhoto = entry ? viewerKind(entry.contentType) === "photo" : false;
  const overlayRef = useRef<HTMLDivElement>(null);
  const [hasEntered, setHasEntered] = useState(false);

  useEffect(() => {
    const overlay = overlayRef.current;
    if (!overlay) return;
    const settle = (event?: AnimationEvent) => {
      if (!event || event.target === overlay) setHasEntered(true);
    };
    overlay.addEventListener("animationend", settle);
    const frame = requestAnimationFrame(() => {
      if (overlay.getAnimations().length === 0) settle();
    });
    return () => {
      overlay.removeEventListener("animationend", settle);
      cancelAnimationFrame(frame);
    };
  }, []);

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
      ref={overlayRef}
      isOpen
      isDismissable
      onOpenChange={(open) => !open && onClose()}
      className="dark bg-secondary text-primary fixed inset-0 z-50 data-[entering]:animate-[modal-fade_200ms]"
    >
      <Modal className="h-full w-full">
        <Dialog
          aria-label={`Viewer: ${entry.file_info.name}`}
          className="grid h-full w-full grid-rows-[auto_1fr_auto] outline-none"
        >
          <header className="flex items-center gap-3 px-3 py-2">
            <IconButton
              icon="close"
              aria-label="Close"
              variant="tertiary"
              onPress={onClose}
            />
            <div className="min-w-0 flex-1">
              <Typography weight="semibold" className="truncate">
                {entry.file_info.name}
              </Typography>
              <Typography size="xs" className="text-tertiary tabular-nums">
                {index + 1} / {items.length} · {entry.file_info.size.Pretty}
              </Typography>
            </div>
            {isPhoto && items.length > 1 && (
              <IconButton
                icon={playing ? "pause" : "play"}
                aria-label={playing ? "Stop the slideshow" : "Play a slideshow"}
                variant="tertiary"
                onPress={() => setPlaying((v) => !v)}
              />
            )}
            <DownloadButton snapshot={snapshot} source={source} entry={entry} />
            <IconButton
              icon="info-circle"
              aria-label="Details"
              variant="tertiary"
              className={clsx(showDetails && "bg-primary-hover")}
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
              {hasEntered && (
                <EntryViewer
                  key={entry.path}
                  variant="stage"
                  snapshot={snapshot}
                  source={source}
                  entry={entry}
                  onEnded={() => index < items.length - 1 && go(1)}
                />
              )}
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
        className="bg-primary rounded-full"
      />
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
  const { getUrl } = useEntryURL({
    snapshot: snapshot.identifier.Value,
    source,
    path: entry.path,
  });
  return (
    <IconButton
      icon="download"
      aria-label="Download"
      variant="tertiary"
      href={getUrl?.({ download: true })}
      download
      isDisabled={!getUrl}
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
    <aside className="border-secondary hidden overflow-y-auto border-l px-5 py-2 md:block">
      <Typography weight="semibold" className="mb-3">
        Details
      </Typography>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
        {rows.map(([label, value]) => (
          <div key={label} className="contents">
            <Typography as="dt" size="sm" className="text-tertiary">
              {label}
            </Typography>
            <Typography as="dd" size="sm" className="break-all">
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
  const isPhoto = viewerKind(entry.contentType) === "photo";
  const { url } = useEntryURL({
    snapshot,
    source,
    path: entry.path,
    enabled: inView && isPhoto,
  });

  return (
    <Button
      ref={ref}
      onPress={onPress}
      aria-label={entry.file_info.name}
      aria-current={isCurrent}
      className={clsx(
        "bg-primary text-tertiary flex h-12 flex-none cursor-pointer items-center justify-center gap-1 overflow-hidden rounded-md border-2 text-xs",
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
          {!isPhoto && (
            <Typography as="span" size="xs" className="truncate">
              {entry.file_info.name}
            </Typography>
          )}
        </>
      )}
    </Button>
  );
}
