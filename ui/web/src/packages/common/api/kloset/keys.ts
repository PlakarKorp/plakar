import type { PaginationParams } from "../pagination";
import type { SnapshotType, VFSEntryType } from "./schema";

type SnapshotVFSKeyParams = {
  repository?: string;
  snapshot: string;
  source: number;
  path: string;
  page: number;
  perPage: number;
  search?: {
    contentTypes?: string[];
    filename?: string;
  };
};

type LocatePathnameKeyParams = {
  snapshot: SnapshotType;
  source: number;
  vfsEntry: VFSEntryType;
  repository: string | undefined;
  pagination: PaginationParams;
};

export const klosetKeys = {
  all: () => ["kloset"] as const,

  snapshotsLists: () => [...klosetKeys.all(), "snapshots.list"] as const,

  snapshotsList: (params: {
    repository: string | undefined;
    importer?: string;
    pagination: PaginationParams;
    since?: Date;
  }) => [...klosetKeys.snapshotsLists(), params] as const,

  snapshotGet: (params: { repository: string | undefined; snapshot: string }) =>
    [...klosetKeys.all(), "snapshots.get", params] as const,

  snapshotVfs: (params: SnapshotVFSKeyParams) =>
    [...klosetKeys.all(), "snapshots.getVFS", params] as const,

  snapshotErrors: (params: {
    repository: string | undefined;
    snapshot: string;
    source: number;
    path: string;
    pagination: PaginationParams;
  }) => [...klosetKeys.all(), "snapshot.errors", params] as const,

  snapshotPath: (params: {
    repository: string | undefined;
    snapshot: string;
    source: number;
    path: string;
  }) => [...klosetKeys.all(), "snapshot.path", params] as const,

  snapshotReaderSignUrl: (params: {
    repository?: string;
    snapshot: string;
    source: number;
    path: string;
  }) => [...klosetKeys.all(), "snapshot.reader-sign-url", params] as const,

  snapshotChunks: (params: {
    repository: string | undefined;
    snapshot: string;
    path: string;
  }) => [...klosetKeys.all(), "snapshot.get-chunks", params] as const,

  repositoryInfo: (params: { repository: string | undefined }) =>
    [...klosetKeys.all(), "repository.info", params] as const,

  repositoryImporterTypes: (params: { repository: string | undefined }) =>
    [...klosetKeys.all(), "repository.importer-types", params] as const,

  locatePathname: (params: LocatePathnameKeyParams) =>
    [...klosetKeys.all(), "repository.locate-pathname", params] as const,

  mounts: () => [...klosetKeys.all(), "mounts"] as const,
  mount: (params: { mountId: string }) =>
    [...klosetKeys.mounts(), params.mountId] as const,

  mountBrowse: (params: { mountId: string }) =>
    [...klosetKeys.mount(params), "browse"] as const,

  mountBrowsePath: (params: { mountId: string; path: string }) =>
    [
      ...klosetKeys.mountBrowse({ mountId: params.mountId }),
      params.path,
    ] as const,

  mountFileDownload: (params: { mountId: string; path: string }) =>
    [...klosetKeys.mount(params), "download", params.path] as const,
};
