import {
  Checkbox,
  Icon,
  IconButton,
  Slider,
  Tag,
  Typography,
} from "@plakar-ui/ui/components";
import clsx from "clsx";
import std_path from "path";
import { useEffect, useRef, useState } from "react";
import { Button } from "react-aria-components";

import { EnrichedChildEntryType } from "@plakar-ui/common/api/kloset";
import { useInView, useLocalStorage } from "@plakar-ui/common/hooks";

import { formatClock, parseTrackName } from "./media-utils";
import { useEntryURL } from "./use-entry-url";

type MusicViewProps = {
  snapshot: string;
  source: number;
  tracks: EnrichedChildEntryType[];
  // The directory being browsed: tracks a search finds below it show the
  // folder they come from.
  basePath: string;
  selected: ReadonlySet<string>;
  onToggleSelected: (entry: EnrichedChildEntryType) => void;
  playingPath: string | null;
  onPlay: (path: string | null) => void;
};

/**
 * Lists the audio files of a directory as a playlist, played by a bar that
 * stays at the bottom of the page.
 */
export function MusicView({
  snapshot,
  source,
  tracks,
  basePath,
  selected,
  onToggleSelected,
  playingPath,
  onPlay,
}: MusicViewProps) {
  return (
    <div className={clsx("flex flex-col gap-3", playingPath && "pb-28")}>
      <div className="flex items-center justify-between gap-3">
        <Typography
          as="h3"
          size="xs"
          weight="semibold"
          className="text-tertiary tracking-wide uppercase"
        >
          Tracks · {tracks.length}
        </Typography>
        <IconButton
          icon="play"
          variant="primary"
          size="sm"
          onPress={() => onPlay(tracks[0].path)}
        >
          Play all
        </IconButton>
      </div>

      <ol className="border-secondary divide-secondary flex flex-col divide-y overflow-hidden rounded-lg border">
        {tracks.map((track, i) => (
          <TrackRow
            key={track.path}
            snapshot={snapshot}
            source={source}
            track={track}
            folder={std_path.relative(basePath, track.parent_path)}
            position={i + 1}
            isCurrent={playingPath === track.path}
            isSelected={selected.has(track.path)}
            onToggleSelected={() => onToggleSelected(track)}
            onPlay={() => onPlay(track.path)}
          />
        ))}
      </ol>
    </div>
  );
}

type MusicPlayerProps = {
  snapshot: string;
  source: number;
  tracks: EnrichedChildEntryType[];
  playingPath: string | null;
  onPlay: (path: string | null) => void;
};

export function MusicPlayer({
  snapshot,
  source,
  tracks,
  playingPath,
  onPlay,
}: MusicPlayerProps) {
  const index = tracks.findIndex((track) => track.path === playingPath);
  if (index === -1) {
    return null;
  }
  return (
    <PlayerBar
      snapshot={snapshot}
      source={source}
      track={tracks[index]}
      position={index + 1}
      count={tracks.length}
      onPrevious={() =>
        onPlay(tracks[(index - 1 + tracks.length) % tracks.length].path)
      }
      onNext={() =>
        onPlay(index < tracks.length - 1 ? tracks[index + 1].path : null)
      }
      onClose={() => onPlay(null)}
    />
  );
}

function Cover({ name, size }: { name: string; size: "sm" | "lg" }) {
  // A gradient seeded by the name, so every track keeps the same cover.
  let hash = 0;
  for (const c of name) hash = (hash * 31 + c.charCodeAt(0)) >>> 0;
  const hue = hash % 360;
  return (
    <span
      aria-hidden
      className={clsx(
        "grid flex-none place-items-center rounded-md text-white/90",
        size === "sm" ? "h-9 w-9" : "h-12 w-12"
      )}
      style={{
        background: `linear-gradient(135deg, hsl(${hue} 70% 55%), hsl(${(hue + 60) % 360} 65% 35%))`,
      }}
    >
      <Icon type="md-audio-file" size={size === "sm" ? "md" : "lg"} />
    </span>
  );
}

type TrackRowProps = {
  snapshot: string;
  source: number;
  track: EnrichedChildEntryType;
  folder: string;
  position: number;
  isCurrent: boolean;
  isSelected: boolean;
  onToggleSelected: () => void;
  onPlay: () => void;
};

function TrackRow({
  snapshot,
  source,
  track,
  folder,
  position,
  isCurrent,
  isSelected,
  onToggleSelected,
  onPlay,
}: TrackRowProps) {
  const { title, tag } = parseTrackName(track.file_info.name);
  return (
    <li
      className={clsx(
        "group flex items-center gap-3 px-3 py-2",
        isCurrent ? "bg-accent-tertiary" : "hover:bg-secondary"
      )}
    >
      <Checkbox
        size="sm"
        aria-label={`Select ${track.file_info.name}`}
        isSelected={isSelected}
        onChange={onToggleSelected}
      />
      <Button
        onPress={onPlay}
        aria-label={`Play ${title}`}
        className="data-focus-visible:outline-focus flex min-w-0 flex-1 cursor-pointer items-center gap-3 rounded text-left data-focus-visible:outline-2"
      >
        <span className="text-tertiary grid w-6 place-items-center tabular-nums">
          {isCurrent ? (
            <Equalizer />
          ) : (
            <>
              <Typography as="span" size="sm" className="group-hover:hidden">
                {position}
              </Typography>
              <span className="text-primary hidden group-hover:inline-flex">
                <Icon type="play" size="sm" />
              </span>
            </>
          )}
        </span>
        <Cover name={track.file_info.name} size="sm" />
        <span className="flex min-w-0 flex-1 flex-col">
          <Typography
            size="sm"
            weight="medium"
            className={clsx("truncate", isCurrent && "text-accent")}
          >
            {title}
          </Typography>
          <Typography size="xs" className="text-tertiary">
            {std_path.extname(track.file_info.name).slice(1).toUpperCase()} ·{" "}
            {track.file_info.size.Pretty}
            {folder && ` · in ${folder}`}
          </Typography>
        </span>
        {tag && (
          <Tag
            variant="default_borderless"
            size="sm"
            className="hidden capitalize md:inline-flex"
          >
            {tag}
          </Tag>
        )}
        <TrackDuration snapshot={snapshot} source={source} track={track} />
      </Button>
    </li>
  );
}

function Equalizer() {
  return (
    <span aria-label="Playing" className="flex h-3.5 items-end gap-0.5">
      {[0, 1, 2].map((i) => (
        <span
          key={i}
          className="bg-accent h-full w-[3px] origin-bottom rounded-sm motion-safe:animate-[equalizer_0.9s_ease-in-out_infinite]"
          style={{ animationDelay: `${-0.3 * i}s` }}
        />
      ))}
    </span>
  );
}

function TrackDuration({
  snapshot,
  source,
  track,
}: {
  snapshot: string;
  source: number;
  track: EnrichedChildEntryType;
}) {
  const [ref, inView] = useInView<HTMLSpanElement>();
  const { url } = useEntryURL({
    snapshot,
    source,
    path: track.path,
    enabled: inView,
  });
  const [duration, setDuration] = useState<number>();

  useEffect(() => {
    if (!url) return;
    // Only the header is fetched to learn the duration.
    const audio = new Audio();
    audio.preload = "metadata";
    const onMetadata = () => setDuration(audio.duration);
    audio.addEventListener("loadedmetadata", onMetadata);
    audio.src = url;
    return () => {
      audio.removeEventListener("loadedmetadata", onMetadata);
      audio.removeAttribute("src");
      audio.load();
    };
  }, [url]);

  return (
    <span ref={ref} className="w-12 text-right">
      <Typography as="span" size="sm" className="text-tertiary tabular-nums">
        {duration !== undefined ? formatClock(duration) : "–:––"}
      </Typography>
    </span>
  );
}

type PlayerBarProps = {
  snapshot: string;
  source: number;
  track: EnrichedChildEntryType;
  position: number;
  count: number;
  onPrevious: () => void;
  onNext: () => void;
  onClose: () => void;
};

function PlayerBar({
  snapshot,
  source,
  track,
  position,
  count,
  onPrevious,
  onNext,
  onClose,
}: PlayerBarProps) {
  const { url } = useEntryURL({ snapshot, source, path: track.path });
  const audioRef = useRef<HTMLAudioElement>(null);
  const [paused, setPaused] = useState(true);
  const [time, setTime] = useState(0);
  const [seekingTo, setSeekingTo] = useState<number | null>(null);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useLocalStorage("playerVolume", 0.8);
  const { title, tag } = parseTrackName(track.file_info.name);

  useEffect(() => {
    if (audioRef.current) audioRef.current.volume = volume;
  }, [volume]);

  const toggle = () => {
    const audio = audioRef.current;
    if (!audio) return;
    if (audio.paused) void audio.play().catch(() => {});
    else audio.pause();
  };

  const previous = () => {
    const audio = audioRef.current;
    // Like every player: a first press restarts the track.
    if (audio && audio.currentTime > 3) audio.currentTime = 0;
    else onPrevious();
  };

  return (
    <div
      role="region"
      aria-label="Player"
      className="border-primary bg-primary/90 flex w-full max-w-[900px] flex-wrap items-center gap-x-6 gap-y-2 rounded-2xl border px-4 py-3 shadow-xl backdrop-blur-md"
    >
      {url && (
        <audio
          ref={audioRef}
          src={url}
          autoPlay
          onPlay={() => setPaused(false)}
          onPause={() => setPaused(true)}
          onTimeUpdate={(e) => setTime(e.currentTarget.currentTime)}
          onLoadedMetadata={(e) => {
            setDuration(e.currentTarget.duration);
            e.currentTarget.volume = volume;
          }}
          onEnded={onNext}
        />
      )}

      <div className="flex min-w-0 flex-[1_1_200px] items-center gap-3">
        <Cover name={track.file_info.name} size="lg" />
        <span className="flex min-w-0 flex-col">
          <Typography weight="semibold" className="truncate">
            {title}
          </Typography>
          <Typography size="xs" className="text-tertiary">
            {tag && <span className="capitalize">{tag} · </span>}
            {position} of {count}
          </Typography>
        </span>
      </div>

      <div className="flex flex-[2_1_300px] flex-col items-center gap-1">
        <div className="flex items-center gap-2">
          <IconButton
            icon="chevron-left"
            aria-label="Previous track"
            variant="tertiary"
            onPress={previous}
          />
          <IconButton
            icon={paused ? "play" : "pause"}
            aria-label={paused ? "Play" : "Pause"}
            variant="primary"
            onPress={toggle}
            isDisabled={!url}
          />
          <IconButton
            icon="chevron-right"
            aria-label="Next track"
            variant="tertiary"
            onPress={onNext}
          />
        </div>
        <div className="flex w-full items-center gap-2">
          <Typography
            as="span"
            size="xs"
            className="text-tertiary w-10 text-right tabular-nums"
          >
            {formatClock(seekingTo ?? time)}
          </Typography>
          <Slider
            aria-label="Seek"
            minValue={0}
            maxValue={duration}
            step={0.1}
            value={seekingTo ?? time}
            onChange={setSeekingTo}
            onChangeEnd={(value) => {
              const audio = audioRef.current;
              if (audio) audio.currentTime = value;
              setSeekingTo(null);
            }}
            className="flex-1"
          />
          <Typography
            as="span"
            size="xs"
            className="text-tertiary w-10 tabular-nums"
          >
            {formatClock(duration)}
          </Typography>
        </div>
      </div>

      <div className="hidden flex-[1_1_160px] items-center justify-end gap-2 md:flex">
        <Icon type="volume-high" size="sm" className="text-tertiary" />
        <Slider
          aria-label="Volume"
          minValue={0}
          maxValue={1}
          step={0.05}
          value={volume}
          onChange={setVolume}
          className="w-24"
        />
        <IconButton
          icon="close"
          aria-label="Close the player"
          variant="tertiary"
          onPress={onClose}
        />
      </div>
    </div>
  );
}
