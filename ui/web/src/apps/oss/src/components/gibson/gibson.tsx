import {
  GetSnapshotQueryOptions,
  RepositorySnapshotsQueryOptions,
  SnapshotPathError,
  SnapshotType,
  useKlosetAPI,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";
import { useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import std_path from "path";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { FileViewer } from "./file-viewer";
import "./gibson.css";
import { GibsonScene, Telemetry } from "./scene";
import {
  integrityCaption,
  MAX_TOWERS,
  MAX_VERIFIED_SNAPSHOTS,
  snapshotLabel,
  Tower,
  TowerFile,
  World,
  worldFromEntries,
  worldFromSnapshots,
} from "./world";

import { snapshotIntegrityQueryOptions } from "~/api/kloset/queries";
import { SnapshotIntegrity } from "~/api/kloset/schema";

/** Where the Gibson opens: the repository, or a directory of a snapshot. */
export type GibsonLocation = {
  snapshot?: string;
  path?: string;
};

type Level =
  | { kind: "repository" }
  | {
      kind: "directory";
      snapshotId: string;
      snapshotLabel: string;
      path: string;
    };

/** What a level is made of, before the verification colors it. */
type Loaded =
  | { kind: "repository"; snapshots: SnapshotType[]; total: number }
  | {
      kind: "directory";
      snapshotId: string;
      snapshotLabel: string;
      path: string;
      entries: VFSEntryType[];
      errors: SnapshotPathError[];
      total: number;
    };

const PAGE = { page: 0, perPage: MAX_TOWERS };

function levelFromLocation(location: GibsonLocation): Level {
  if (!location.snapshot) {
    return { kind: "repository" };
  }
  return {
    kind: "directory",
    snapshotId: location.snapshot,
    snapshotLabel: location.snapshot.slice(0, 8),
    path: location.path || "/",
  };
}

function sameLevel(a: Level, b: Level) {
  if (a.kind === "repository" || b.kind === "repository") {
    return a.kind === b.kind;
  }
  return a.snapshotId === b.snapshotId && a.path === b.path;
}

export function Gibson({ location }: { location: GibsonLocation }) {
  const { api } = useKlosetAPI();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const containerRef = useRef<HTMLDivElement>(null);
  const sceneRef = useRef<GibsonScene | null>(null);
  const levelRef = useRef<Level>(levelFromLocation(location));
  // The last world placed in the scene, so a rebuilt scene gets it back and
  // a recolor is told apart from a new level.
  const placedRef = useRef<{ loaded: Loaded; world: World } | null>(null);

  const [level, setLevel] = useState<Level>(levelRef.current);
  const [loaded, setLoaded] = useState<Loaded | null>(null);
  const [target, setTarget] = useState<Tower | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">(
    "loading"
  );
  const [error, setError] = useState("");
  const [openFile, setOpenFile] = useState<TowerFile | null>(null);
  const [vrSupported, setVrSupported] = useState(false);
  const [telemetry, setTelemetry] = useState<Telemetry>({
    x: 0,
    y: 0,
    z: 0,
    heading: 0,
    locked: false,
    vr: false,
  });

  useEffect(() => {
    levelRef.current = level;
  }, [level]);

  useEffect(() => {
    GibsonScene.vrSupported().then(setVrSupported);
  }, []);

  // A file on screen takes the keyboard away from the flight.
  useEffect(() => {
    sceneRef.current?.setFrozen(openFile !== null);
  }, [openFile]);

  // Back to the flat UI, at the place the pilot is flying over.
  const exit = useCallback(() => {
    const current = levelRef.current;
    if (current.kind === "directory") {
      navigate({
        to: "/snapshots/$snapshotId/$source/browse",
        params: { snapshotId: current.snapshotId, source: 0 },
        search: { path: current.path },
      });
    } else {
      navigate({ to: "/snapshots" });
    }
  }, [navigate]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }
    const scene = new GibsonScene(container, {
      onTarget: setTarget,
      onEnter: (tower) => {
        if (tower.destination) {
          setLevel({ kind: "directory", ...tower.destination });
        } else if (tower.file) {
          setOpenFile(tower.file);
        }
      },
      onUp: () =>
        setLevel((prev) => {
          if (prev.kind === "repository") {
            return prev;
          }
          if (prev.path === "/") {
            return { kind: "repository" };
          }
          return { ...prev, path: std_path.dirname(prev.path) };
        }),
      onExit: exit,
      onTelemetry: setTelemetry,
    });
    sceneRef.current = scene;
    if (placedRef.current) {
      scene.setWorld(placedRef.current.world);
    }
    return () => {
      scene.dispose();
      sceneRef.current = null;
    };
  }, [exit]);

  // The URL follows the flight, so a reload or a shared link lands at the same place.
  useEffect(() => {
    navigate({
      to: "/gibson",
      search:
        level.kind === "repository"
          ? {}
          : { snapshot: level.snapshotId, path: level.path },
      replace: true,
    });
  }, [level, navigate]);

  // And the flight follows the URL, so the browser's back button works.
  useEffect(() => {
    const wanted = levelFromLocation(location);
    if (!sameLevel(wanted, levelRef.current)) {
      setLevel(wanted);
    }
  }, [location]);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");

    const load = async (): Promise<Loaded> => {
      if (level.kind === "repository") {
        const snapshots = await queryClient.fetchQuery({
          ...RepositorySnapshotsQueryOptions(api, {
            repository: undefined,
            pagination: PAGE,
          }),
          staleTime: 30 * 1000,
        });
        return {
          kind: "repository",
          snapshots: snapshots.items,
          total: snapshots.total,
        };
      }

      const { snapshotId, path } = level;
      const [snapshot, children, errors] = await Promise.all([
        queryClient.fetchQuery(
          GetSnapshotQueryOptions(api, {
            repository: undefined,
            snapshot: snapshotId,
          })
        ),
        queryClient.fetchQuery({
          queryKey: ["gibson", "children", snapshotId, path],
          queryFn: () =>
            api.getSnapshotPathChildren({
              repository: undefined,
              snapshot: snapshotId,
              source: 0,
              path,
              pagination: PAGE,
            }),
          staleTime: Infinity,
        }),
        queryClient.fetchQuery({
          queryKey: ["gibson", "errors", snapshotId, path],
          queryFn: () =>
            api.getSnapshotPathErrors({
              repository: undefined,
              snapshot: snapshotId,
              source: 0,
              path,
              pagination: PAGE,
            }),
          staleTime: Infinity,
        }),
      ]);
      return {
        kind: "directory",
        snapshotId,
        snapshotLabel: snapshotLabel(snapshot.item),
        path,
        entries: children.items,
        errors: errors.items,
        total: children.total,
      };
    };

    load()
      .then((result) => {
        if (cancelled) {
          return;
        }
        setLoaded(result);
        setStatus("ready");
      })
      .catch((e: Error) => {
        if (cancelled) {
          return;
        }
        setError(e.message);
        setStatus("error");
      });

    return () => {
      cancelled = true;
    };
  }, [level, api, queryClient]);

  // The verification of the snapshot flown over, and of the most recent ones
  // at the repository level. Each starts on first ask and is polled to its end.
  const currentSnapshotId =
    level.kind === "directory" ? level.snapshotId : undefined;
  const currentIntegrity = useQuery({
    ...snapshotIntegrityQueryOptions(currentSnapshotId ?? ""),
    enabled: currentSnapshotId !== undefined,
  });
  const verifiedIds = useMemo(
    () =>
      loaded?.kind === "repository"
        ? loaded.snapshots
            .slice(0, MAX_VERIFIED_SNAPSHOTS)
            .map((snapshot) => snapshot.identifier.Value)
        : [],
    [loaded]
  );
  const integrities = useQueries({
    queries: verifiedIds.map((id) => snapshotIntegrityQueryOptions(id)),
    combine: (results) => {
      const byId = new Map<string, SnapshotIntegrity>();
      results.forEach((result, i) => {
        const id = verifiedIds[i];
        if (result.data && id) {
          byId.set(id, result.data.item);
        }
      });
      return byId;
    },
  });

  const world = useMemo(() => {
    if (!loaded) {
      return null;
    }
    if (loaded.kind === "repository") {
      return worldFromSnapshots(loaded.snapshots, loaded.total, integrities);
    }
    return worldFromEntries(
      loaded.entries,
      loaded.errors,
      loaded.total,
      loaded.snapshotId,
      loaded.snapshotLabel,
      loaded.path,
      currentIntegrity.data?.item
    );
  }, [loaded, integrities, currentIntegrity.data]);

  // A new level is placed from scratch; new colors on the same level are
  // painted onto the towers in place, so the flight is not interrupted.
  useEffect(() => {
    if (!world || !loaded) {
      return;
    }
    const scene = sceneRef.current;
    if (placedRef.current?.loaded === loaded) {
      scene?.restyle(world);
    } else {
      scene?.setWorld(world);
    }
    placedRef.current = { loaded, world };
  }, [world, loaded]);

  const hidden = world ? world.total - world.towers.length : 0;
  const caption =
    level.kind === "repository"
      ? "REPOSITORY"
      : `SNAPSHOT ${level.snapshotId.slice(0, 8)} :: ${level.path}`;
  const integrityLine =
    level.kind === "directory"
      ? integrityCaption(currentIntegrity.data?.item)
      : null;

  // The headset cannot read the DOM, so it gets the same HUD painted in-world.
  useEffect(() => {
    sceneRef.current?.setHud([
      `THE GIBSON // ${caption}`,
      world ? `${world.towers.length} OBJECTS` : "ACCESSING...",
      integrityLine ?? "",
      ...(target
        ? [
            `TARGET ${target.label}`,
            ...target.details.map(([key, value]) => `${key}: ${value}`),
          ]
        : ["NO TARGET"]),
    ]);
  }, [caption, world, target, integrityLine]);

  const enterVR = () => {
    sceneRef.current?.enterVR().catch((e: Error) => {
      setError(e.message);
      setStatus("error");
    });
  };

  return (
    <div className="gibson relative h-full w-full overflow-hidden bg-black select-none">
      <div ref={containerRef} className="absolute inset-0" />
      <div className="gibson-scanlines pointer-events-none absolute inset-0" />
      <div className="gibson-vignette pointer-events-none absolute inset-0" />

      {telemetry.locked && (
        <div className="gibson-crosshair pointer-events-none absolute top-1/2 left-1/2 h-6 w-6 -translate-x-1/2 -translate-y-1/2" />
      )}

      <div className="absolute top-4 left-4 flex flex-col gap-1 text-xs leading-5 sm:text-sm">
        <div className="text-xl font-bold tracking-[0.3em]">THE GIBSON</div>
        <div>PLAKAR // {caption}</div>
        {world && (
          <div>
            {world.towers.length} OBJECTS
            {hidden > 0 && ` (+${hidden} BEYOND RANGE)`}
          </div>
        )}
        {integrityLine && (
          <div
            className={
              currentIntegrity.data?.item.corrupted.length
                ? "text-red-400"
                : undefined
            }
          >
            {integrityLine}
          </div>
        )}
        {status === "loading" && (
          <div className="gibson-blink">ACCESSING...</div>
        )}
        {status === "error" && (
          <div className="text-red-400">ACCESS DENIED: {error}</div>
        )}
        <div className="mt-2 flex gap-2">
          <button
            type="button"
            onClick={exit}
            className="border border-current px-2 py-0.5 hover:bg-cyan-400/20"
          >
            [X] EXIT GIBSON
          </button>
          {vrSupported && !telemetry.vr && (
            <button
              type="button"
              onClick={enterVR}
              className="border border-current px-2 py-0.5 hover:bg-cyan-400/20"
            >
              ENTER VR
            </button>
          )}
        </div>
      </div>

      {target && (
        <div className="gibson-panel pointer-events-none absolute top-4 right-4 w-80 p-3 text-xs leading-5">
          <div className="mb-1 font-bold tracking-widest">TARGET LOCK</div>
          <table>
            <tbody>
              {target.details.map(([key, value]) => (
                <tr key={key} className="align-top">
                  <td className="pr-3 whitespace-nowrap opacity-70">{key}</td>
                  <td className="break-all">{value}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="mt-2 opacity-70">
            {target.destination
              ? "[ENTER] DIVE IN"
              : target.file
                ? "[ENTER] OPEN FILE"
                : "[ENTER] PING"}
          </div>
        </div>
      )}

      <div className="pointer-events-none absolute right-56 bottom-4 left-4 text-xs opacity-80">
        CLICK A TOWER TO DIVE · WASD / ARROWS FLY · SPACE / C UP / DOWN · SHIFT
        BOOST · DRAG TO LOOK · L LOCK MOUSE · BACKSPACE SURFACE · X EXIT
      </div>

      <div className="pointer-events-none absolute right-4 bottom-4 text-right text-xs tabular-nums">
        POS {telemetry.x.toFixed(0)} {telemetry.y.toFixed(0)}{" "}
        {telemetry.z.toFixed(0)} · HDG {telemetry.heading.toFixed(0)}°
      </div>

      {openFile && (
        <FileViewer file={openFile} onClose={() => setOpenFile(null)} />
      )}
    </div>
  );
}
