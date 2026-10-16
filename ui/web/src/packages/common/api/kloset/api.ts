import {
  downloadSnapshotVFSFn,
  generateSnapshotSignedURLFn,
  getRepositoryImporterTypesFn,
  getRepositoryInfoFn,
  getRepositoryLocatePathnameFn,
  getSnapshotFn,
  getSnapshotPathChunksFn,
  getSnapshotPathChildrenFn,
  getSnapshotPathErrorsFn,
  getSnapshotPathFn,
  getSnapshotPathSearchFn,
  getSnapshotsListFn,
  createMountFn,
  browseMountFn,
  deleteMountFn,
  getMountFn,
  getMountFileInfoFn,
  getMountFileDownloadSignatureURLFn,
} from "./schema";

export type KlosetAPI = {
  getKlosetURL: (repository: string | undefined) => string;
  getRepositoryInfo: getRepositoryInfoFn;
  getRepositoryImporterTypes: getRepositoryImporterTypesFn;
  getRepositoryLocatePathname: getRepositoryLocatePathnameFn;
  getSnapshotsList: getSnapshotsListFn;
  getSnapshot: getSnapshotFn;
  getSnapshotPath: getSnapshotPathFn;
  getSnapshotPathChildren: getSnapshotPathChildrenFn;
  getSnapshotPathSearch: getSnapshotPathSearchFn;
  getSnapshotPathErrors: getSnapshotPathErrorsFn;
  downloadSnapshotVFS: downloadSnapshotVFSFn;
  getSnapshotPathChunks: getSnapshotPathChunksFn;
  generateSnapshotSignedURL: generateSnapshotSignedURLFn;

  mount?: {
    get: getMountFn;
    create: createMountFn;
    delete: deleteMountFn;
    browse: browseMountFn;
    getFileDownloadSignature: getMountFileDownloadSignatureURLFn;
    getFileInfo: getMountFileInfoFn;
  };
};
