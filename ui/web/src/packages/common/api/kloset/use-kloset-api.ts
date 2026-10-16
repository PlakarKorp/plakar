import { createContext, useContext } from "react";

import { KlosetAPI } from "./api";

function oops(name: string) {
  return function () {
    throw new Error(`${name} function not provided`);
  };
}

export const KlosetAPIContext = createContext<{
  repository: string | undefined;
  api: KlosetAPI;
}>({
  repository: undefined,
  api: {
    getKlosetURL: oops("getKlosetURL"),
    getRepositoryInfo: oops("getRepositoryInfo"),
    getRepositoryImporterTypes: oops("getRepositoryImporterTypes"),
    getRepositoryLocatePathname: oops("getRepositoryLocatePathname"),
    getSnapshotsList: oops("getSnapshotsList"),
    getSnapshot: oops("getSnapshot"),
    getSnapshotPath: oops("getSnapshotPath"),
    getSnapshotPathChildren: oops("getSnapshotPathChildren"),
    getSnapshotPathSearch: oops("getSnapshotPathSearch"),
    getSnapshotPathErrors: oops("getSnapshotPathErrors"),
    downloadSnapshotVFS: oops("downloadSnapshotVFS"),
    getSnapshotPathChunks: oops("getSnapshotPathChunks"),
    generateSnapshotSignedURL: oops("generateSnapshotSignedURL"),
  },
});

export function useKlosetAPI() {
  return useContext(KlosetAPIContext);
}

export function useKlosetAPIWithRepository() {
  const { api, repository } = useContext(KlosetAPIContext);
  if (!repository) {
    throw new Error("Repository not provided in KlosetAPIContext");
  }
  return { api, repository };
}
