import {
  SnapshotPathError,
  SnapshotType,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";
import std_path from "path";

import { SnapshotIntegrity } from "~/api/kloset/schema";

/** How many towers a level shows at most; the HUD says how many are left out. */
export const MAX_TOWERS = 400;

/** How many snapshots the repository level verifies: every one would read the whole store. */
export const MAX_VERIFIED_SNAPSHOTS = 20;

/** One line on the state of a verification, for the HUD. */
export function integrityCaption(integrity: SnapshotIntegrity | undefined) {
  if (!integrity) {
    return "INTEGRITY: UNKNOWN";
  }
  switch (integrity.status) {
    case "running":
      return `INTEGRITY: SCANNING (${integrity.checked} FILES READ)`;
    case "failed":
      return `INTEGRITY: SCAN FAILED, ${integrity.message ?? ""}`;
    default:
      return integrity.corrupted.length > 0
        ? `INTEGRITY: ${integrity.corrupted.length} CORRUPTED`
        : "INTEGRITY: CLEAN";
  }
}

/** Distance between two tower centers. */
const SPACING = 10;

export type TowerKind = "snapshot" | "directory" | "file" | "symlink" | "other";

export type TowerDestination = {
  snapshotId: string;
  snapshotLabel: string;
  path: string;
};

export type Tower = {
  id: string;
  kind: TowerKind;
  /** Name drawn above the tower. */
  label: string;
  /** Rows of the HUD panel shown while the tower is targeted. */
  details: [string, string][];
  size: number;
  hasErrors: boolean;
  x: number;
  z: number;
  width: number;
  depth: number;
  height: number;
  /** Where diving into the tower leads; a file has none. */
  destination?: TowerDestination;
  /** The file behind the tower, when diving in opens its content. */
  file?: TowerFile;
};

export type TowerFile = {
  snapshotId: string;
  path: string;
  name: string;
  contentType: string;
  size: number;
  sizePretty: string;
};

export type World = {
  /** Caption of the level in the HUD. */
  title: string;
  towers: Tower[];
  /** How many items the level holds, shown or not. */
  total: number;
  /** Half the side of the laid out city. */
  extent: number;
};

/** A tower grows with the logarithm of its size: a kilobyte is a hut, a terabyte a skyscraper. */
function towerHeight(size: number) {
  return 2 + 6 * Math.log10(1 + size / 4096);
}

function snapshotSource(snapshot: SnapshotType) {
  const source = snapshot.sources[0];
  if (!source) {
    return "";
  }
  return `${source.importer.type}:${source.importer.directory.toHumanString()}`;
}

/** The label of a snapshot level: its short id and where it came from. */
export function snapshotLabel(snapshot: SnapshotType) {
  return `${snapshot.identifier.toHumanString()} ${snapshotSource(snapshot)}`;
}

/**
 * Builds the repository level. A snapshot is red when its backup recorded
 * errors or when its verification found a corrupted file.
 */
export function worldFromSnapshots(
  snapshots: SnapshotType[],
  total: number,
  integrity: Map<string, SnapshotIntegrity>
): World {
  const towers = snapshots.slice(0, MAX_TOWERS).map((snapshot): Tower => {
    const id = snapshot.identifier.toHumanString();
    const size = snapshot.total.size.Value;
    const verification = integrity.get(snapshot.identifier.Value);
    const corrupted = verification?.corrupted.length ?? 0;
    return {
      id: snapshot.identifier.Value,
      kind: "snapshot",
      label: id,
      details: [
        ["ID", id],
        ["WHEN", snapshot.timestamp.toHumanString() ?? ""],
        ["SOURCE", snapshotSource(snapshot)],
        ["HOST", snapshot.context.Hostname],
        ["SIZE", snapshot.total.size.Pretty],
        ["FILES", String(snapshot.total.files)],
        ["DIRS", String(snapshot.total.directories)],
        ["ERRORS", String(snapshot.total.errors)],
        [
          "INTEGRITY",
          integrityCaption(verification).replace("INTEGRITY: ", ""),
        ],
      ],
      size,
      hasErrors: snapshot.total.errors > 0 || corrupted > 0,
      x: 0,
      z: 0,
      width: 6,
      depth: 6,
      height: towerHeight(size) + 4,
      destination: {
        snapshotId: snapshot.identifier.Value,
        snapshotLabel: snapshotLabel(snapshot),
        path: "/",
      },
    };
  });
  return layout("REPOSITORY", towers, total);
}

function entryKind(entry: VFSEntryType): TowerKind {
  const mode = entry.file_info.mode;
  if (mode.isDir()) {
    return "directory";
  }
  if (mode.isSymlink()) {
    return "symlink";
  }
  if (mode.isRegular()) {
    return "file";
  }
  return "other";
}

/**
 * Builds a directory level. The errors are the ones recorded beneath the
 * directory: a file that failed is red, and so is every directory above it,
 * which the summaries already count. A failed file missing from the listing
 * still gets a red tower, so the hole shows. The verification adds the files
 * whose data no longer reads back, red too along with their ancestors.
 */
export function worldFromEntries(
  entries: VFSEntryType[],
  errors: SnapshotPathError[],
  total: number,
  snapshotId: string,
  snapshotLabel: string,
  path: string,
  integrity: SnapshotIntegrity | undefined
): World {
  const failures = new Map(errors.map((error) => [error.name, error.error]));
  const corrupted = new Map(
    (integrity?.corrupted ?? []).map((failure) => [failure.path, failure.error])
  );
  const corruptedBelow = (dir: string) => {
    const prefix = dir.endsWith("/") ? dir : `${dir}/`;
    let count = 0;
    for (const file of corrupted.keys()) {
      if (file.startsWith(prefix)) {
        count++;
      }
    }
    return count;
  };
  const shown = entries.filter((entry) => entry.file_info.name !== "..");
  // Directories first: they are the tall towers, so they make the skyline.
  shown.sort((a, b) => {
    const dirA = a.file_info.mode.isDir() ? 0 : 1;
    const dirB = b.file_info.mode.isDir() ? 0 : 1;
    return dirA - dirB || a.file_info.name.localeCompare(b.file_info.name);
  });

  const towers = shown.slice(0, MAX_TOWERS).map((entry): Tower => {
    const kind = entryKind(entry);
    const summary = entry.summary?.Total;
    const size =
      kind === "directory"
        ? (summary?.size.Value ?? 0)
        : entry.file_info.size.Value;
    const pretty =
      kind === "directory"
        ? (summary?.size.Pretty ?? entry.file_info.size.Pretty)
        : entry.file_info.size.Pretty;

    const details: [string, string][] = [
      ["NAME", entry.file_info.name],
      ["TYPE", kind.toUpperCase()],
      ["SIZE", pretty],
      ["MODIFIED", entry.file_info.mod_time.toHumanString() ?? ""],
      ["MODE", entry.file_info.mode.toString()],
    ];
    if (kind === "directory") {
      details.push([
        "HOLDS",
        `${summary?.files ?? 0} FILES, ${summary?.directories ?? 0} DIRS`,
      ]);
    } else if (kind === "symlink") {
      details.push(["TARGET", entry.symlink_target ?? ""]);
    } else if (entry.contentType) {
      details.push(["CONTENT", entry.contentType]);
    }
    const failure = failures.get(entry.path);
    if (failure) {
      details.push(["ERROR", failure]);
    }
    const isDir = kind === "directory";
    const damage = isDir ? corruptedBelow(entry.path) : 0;
    const corruption = corrupted.get(entry.path);
    if (corruption) {
      details.push(["INTEGRITY", `CORRUPTED, ${corruption}`]);
    } else if (damage > 0) {
      details.push(["INTEGRITY", `${damage} CORRUPTED BELOW`]);
    }

    return {
      id: entry.path,
      kind,
      label: entry.file_info.name,
      details,
      size,
      hasErrors:
        (summary?.errors ?? 0) > 0 ||
        failure !== undefined ||
        corruption !== undefined ||
        damage > 0,
      x: 0,
      z: 0,
      width: isDir ? 5 : 3,
      depth: isDir ? 5 : 3,
      height: towerHeight(size) + (isDir ? 2 : 0),
      destination: isDir
        ? { snapshotId, snapshotLabel, path: entry.path }
        : undefined,
      file:
        kind === "file"
          ? {
              snapshotId,
              path: entry.path,
              name: entry.file_info.name,
              contentType: entry.contentType,
              size,
              sizePretty: pretty,
            }
          : undefined,
    };
  });

  // The files the backup could not read sit nowhere in the listing: a red
  // stump marks each one.
  const listed = new Set(shown.map((entry) => entry.path));
  let ghosts = 0;
  for (const [name, error] of failures) {
    if (std_path.dirname(name) !== path || listed.has(name)) {
      continue;
    }
    ghosts++;
    towers.push({
      id: name,
      kind: "file",
      label: std_path.basename(name),
      details: [
        ["NAME", std_path.basename(name)],
        ["STATUS", "CORRUPTED"],
        ["ERROR", error],
      ],
      size: 0,
      hasErrors: true,
      x: 0,
      z: 0,
      width: 3,
      depth: 3,
      height: 3,
    });
  }

  return layout(`${snapshotLabel} :: ${path}`, towers, total + ghosts);
}

/** Lays the towers out on a square grid centered on the origin. */
function layout(title: string, towers: Tower[], total: number): World {
  const cols = Math.max(1, Math.ceil(Math.sqrt(towers.length)));
  const rows = Math.max(1, Math.ceil(towers.length / cols));
  towers.forEach((tower, i) => {
    const col = i % cols;
    const row = Math.floor(i / cols);
    tower.x = (col - (cols - 1) / 2) * SPACING;
    tower.z = (row - (rows - 1) / 2) * SPACING;
  });
  return {
    title,
    towers,
    total,
    extent: (Math.max(cols, rows) * SPACING) / 2,
  };
}
