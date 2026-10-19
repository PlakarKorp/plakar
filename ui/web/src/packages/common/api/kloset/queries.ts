import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";

import { MIME_FILTERS } from "../../components";
import { APIResponseNotFoundError } from "../errors";
import { PaginationParams } from "../pagination";
import { KlosetAPI } from "./api";
import { klosetKeys } from "./keys";
import {
  browseMountFn,
  enrichGetSnapshotPathChildren,
  getMountFn,
  getRepositoryImporterTypesFn,
  getRepositoryInfoFn,
  getSnapshotFn,
  getSnapshotPathChunksFn,
  getSnapshotPathErrorsFn,
  getSnapshotPathFn,
  getSnapshotsListFn,
  ReadFileParams,
  RepositoryLocatePathname,
  SnapshotType,
  VFSEntryType,
} from "./schema";
import { ContentType } from "../enrich/content_type";

export const RepositoryInfoQueryOptions = (
  api: KlosetAPI,
  ...props: Parameters<getRepositoryInfoFn>
) =>
  queryOptions({
    queryKey: klosetKeys.repositoryInfo(props[0]),
    queryFn: () => api.getRepositoryInfo(...props),
  });

export const RepositoryImporterTypesQueryOptions = (
  api: KlosetAPI,
  ...props: Parameters<getRepositoryImporterTypesFn>
) =>
  queryOptions({
    queryKey: klosetKeys.repositoryImporterTypes(props[0]),
    queryFn: () => api.getRepositoryImporterTypes(...props),
  });

export const RepositorySnapshotsQueryOptions = (
  api: KlosetAPI,
  ...props: Parameters<getSnapshotsListFn>
) =>
  queryOptions({
    queryKey: klosetKeys.snapshotsList(props[0]),
    queryFn: () => api.getSnapshotsList(...props),
  });

export const GetSnapshotQueryOptions = (
  api: KlosetAPI,
  ...props: Parameters<getSnapshotFn>
) =>
  queryOptions({
    queryKey: klosetKeys.snapshotGet(props[0]),
    queryFn: () => api.getSnapshot(...props),

    // Snapshots are immutable, so we don't need to refetch them.
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    refetchOnReconnect: false,
    staleTime: Infinity,
  });

/**
 * Adapts a name filter to how the search endpoint applies it. A recursive
 * search with content types goes through kloset's content-type index, which
 * matches names as globs only, not as the substrings the plain search finds:
 * a bare word becomes *word* there so both searches find the same files.
 */
export function searchNameFilter(
  filename: string | undefined,
  { indexed }: { indexed: boolean }
) {
  if (!filename || !indexed || /[*?[\]]/.test(filename)) {
    return filename;
  }
  return `*${filename}*`;
}

export const SnapshotVFSQueryOptions = (
  api: KlosetAPI,
  {
    repository,
    snapshot,
    source,
    vfsEntry,
    page,
    perPage,
    search,
  }: {
    repository?: string;
    snapshot: string;
    source: number;
    vfsEntry: VFSEntryType;
    search?: {
      contentTypes?: (typeof MIME_FILTERS)[number]["contentTypes"];
      filename?: string;
      // Defaults to true: the filters search the whole tree below the path.
      recursive?: boolean;
    };
  } & PaginationParams
) =>
  queryOptions({
    queryKey: klosetKeys.snapshotVfs({
      repository,
      snapshot,
      source,
      path: vfsEntry.path,
      page,
      perPage,
      search,
    }),
    queryFn: async () => {
      const resp = await (search === undefined
        ? api.getSnapshotPathChildren({
            repository,
            snapshot,
            source,
            path: vfsEntry.path,
            pagination: { page, perPage },
          })
        : api.getSnapshotPathSearch({
            repository,
            snapshot,
            source,
            path: vfsEntry.path,
            pagination: { page, perPage },
            recursive: search.recursive ?? true,
            mimes: search.contentTypes,
            filename: searchNameFilter(search.filename, {
              indexed:
                (search.recursive ?? true) &&
                (search.contentTypes?.length ?? 0) > 0,
            }),
          }));

      return {
        ...resp,
        isSearching: search !== undefined,
        items: enrichGetSnapshotPathChildren(vfsEntry, resp.items),
      };
    },
    // Snapshots content is immutable, so we don't need to refetch it.
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    refetchOnReconnect: false,
    staleTime: Infinity,
  });

export const SnapshotVFSErrorsQueryOptions = (
  api: KlosetAPI,
  ...props: Parameters<getSnapshotPathErrorsFn>
) =>
  queryOptions({
    queryKey: klosetKeys.snapshotErrors(props[0]),
    queryFn: () => api.getSnapshotPathErrors(...props),
  });

export const GetSnapshotPathQueryOptions = (
  api: KlosetAPI,
  ...props: Parameters<getSnapshotPathFn>
) =>
  queryOptions({
    queryKey: klosetKeys.snapshotPath(props[0]),
    queryFn: () => api.getSnapshotPath(...props),

    // Snapshots are immutable, so we don't need to refetch them.
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    refetchOnReconnect: false,
    staleTime: Infinity,
  });

export class SnapshotReader {
  klosetAPI: KlosetAPI;
  repository?: string;
  snapshotId: string;
  source: number;
  path: string;

  constructor(
    klosetAPI: KlosetAPI,
    repository: string | undefined,
    snapshotId: string,
    source: number,
    path: string
  ) {
    this.klosetAPI = klosetAPI;
    this.repository = repository;
    this.snapshotId = snapshotId;
    this.source = source;
    this.path = path;
  }

  /**
   * Returns the type of the file (text, json, image, unknown) based on its content type.
   */
  async getType() {
    const respGetSnapshotPath = await this.klosetAPI.getSnapshotPath({
      repository: this.repository,
      snapshot: this.snapshotId,
      source: this.source,
      path: this.path,
    });

    if (!respGetSnapshotPath.item.file_info.mode.isRegular()) {
      throw new Error("attempted to read a non-file entry");
    }

    const contentType = respGetSnapshotPath.item.contentType;
    return ContentType(contentType);
  }

  /**
   * Calls the API endpoint /api/snapshot/reader-sign-url/:snapshot_id:path to get a signed URL.
   * This URL can be used to read the file without providing the Authorization header.
   */
  async getSignature() {
    return this.klosetAPI.generateSnapshotSignedURL({
      repository: this.repository,
      snapshot: this.snapshotId,
      source: this.source,
      path: this.path,
    });
  }

  /**
   * Construct the public URL to read the file.
   */
  getPublicUrl(signature: string, params?: ReadFileParams) {
    const qs = new URLSearchParams({ signature });

    if (params?.render) {
      qs.append("render", params.render);
    }
    if (params?.download) {
      qs.append("download", "true");
    }
    const publicUrl = this.klosetAPI.getKlosetURL(this.repository);
    return `${publicUrl}/snapshot/reader/${this.snapshotId}:${encodeURIComponent(this.path)}?${qs}`;
  }
}

/**
 * Signs the URL to read one file. One signature serves every way of reading
 * it, so the viewer, its download button and a gallery tile share the query.
 */
export const SnapshotReaderURLQueryOptions = (
  api: KlosetAPI,
  {
    repository,
    snapshot,
    source,
    path,
  }: {
    repository?: string;
    snapshot: string;
    source: number;
    path: string;
  }
) =>
  queryOptions({
    queryKey: klosetKeys.snapshotReaderURL({
      repository,
      snapshot,
      source,
      path,
    }),
    queryFn: async () => {
      const reader = new SnapshotReader(
        api,
        repository,
        snapshot,
        source,
        path
      );
      const signature = await reader.getSignature();
      if (signature.status !== "ok") {
        return { status: signature.status, getUrl: null };
      }
      return {
        status: signature.status,
        getUrl: (params?: ReadFileParams) =>
          reader.getPublicUrl(signature.value, params),
      };
    },
    // Signatures last two hours: refresh them well before that.
    staleTime: 60 * 60 * 1000,
    gcTime: 60 * 60 * 1000,
    refetchOnWindowFocus: false,
  });

/**
 * Reads the first bytes of a file as text, for the previews that only need
 * its opening lines.
 */
export const SnapshotFileHeadQueryOptions = (
  api: KlosetAPI,
  {
    repository,
    snapshot,
    source,
    path,
    bytes,
  }: {
    repository?: string;
    snapshot: string;
    source: number;
    path: string;
    bytes: number;
  }
) =>
  queryOptions({
    queryKey: klosetKeys.snapshotFileHead({
      repository,
      snapshot,
      source,
      path,
      bytes,
    }),
    queryFn: async ({ signal }) => {
      const reader = new SnapshotReader(
        api,
        repository,
        snapshot,
        source,
        path
      );
      const signature = await reader.getSignature();
      if (signature.status !== "ok") {
        return null;
      }
      const res = await fetch(
        reader.getPublicUrl(signature.value, { render: "text" }),
        { headers: { Range: `bytes=0-${bytes - 1}` }, signal }
      );
      if (!res.ok) {
        throw new Error(`reading ${path} failed with HTTP/${res.status}`);
      }
      return (await res.text()).slice(0, bytes);
    },
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    refetchOnReconnect: false,
    staleTime: Infinity,
  });

/**
 * Lists a whole directory of a snapshot, up to `max` entries, for views that
 * need every entry at once such as a comparison. Resolves to null when the
 * path does not exist in the snapshot or is not a directory.
 */
export const SnapshotDirectoryQueryOptions = (
  api: KlosetAPI,
  {
    repository,
    snapshot,
    source,
    path,
    max,
  }: {
    repository?: string;
    snapshot: string;
    source: number;
    path: string;
    max: number;
  }
) =>
  queryOptions({
    queryKey: klosetKeys.snapshotDirectory({
      repository,
      snapshot,
      source,
      path,
      max,
    }),
    queryFn: async () => {
      let entry: VFSEntryType;
      try {
        entry = (
          await api.getSnapshotPath({ repository, snapshot, source, path })
        ).item;
      } catch (err) {
        if (err instanceof APIResponseNotFoundError) return null;
        throw err;
      }
      if (!entry.file_info.mode.isDir()) return null;

      const total = entry.summary?.directory.children ?? 0;
      // The listing starts with ".." below the root: ask for one more.
      const resp = await api.getSnapshotPathChildren({
        repository,
        snapshot,
        source,
        path,
        pagination: { page: 0, perPage: Math.min(total, max) + 1 },
      });
      return {
        entry,
        children: enrichGetSnapshotPathChildren(
          entry,
          resp.items.filter((child) => child.file_info.name !== "..")
        ),
        isTruncated: total > max,
      };
    },
    // Snapshots are immutable.
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    refetchOnReconnect: false,
    staleTime: Infinity,
  });

export const GetSnapshotPathChunksInfiniteQueryOptions = (
  api: KlosetAPI,
  props: Omit<Parameters<getSnapshotPathChunksFn>[0], "pagination">
) =>
  infiniteQueryOptions({
    queryKey: klosetKeys.snapshotChunks(props),
    initialPageParam: { page: 0, perPage: 10 },

    queryFn: async ({ pageParam }) => {
      return api.getSnapshotPathChunks({
        ...props,
        pagination: { page: pageParam.page, perPage: pageParam.perPage },
      });
    },

    getNextPageParam: (lastPage, _allPages, lastPageParams) => {
      if (!lastPage) {
        return undefined;
      }
      if (
        (lastPageParams.page + 1) * lastPageParams.perPage >=
        lastPage.total
      ) {
        return undefined;
      }
      return { page: lastPageParams.page + 1, perPage: lastPageParams.perPage };
    },
  });

export const RepositoryLocatePathnameQueryOptions = (
  api: KlosetAPI,
  snapshot: SnapshotType,
  source: number,
  vfsEntry: VFSEntryType,
  props: {
    repository: string | undefined;
    pagination: PaginationParams;
  }
) => {
  return {
    queryKey: klosetKeys.locatePathname({
      snapshot,
      source,
      vfsEntry,
      ...props,
    }),
    initialPageParam: props.pagination,

    queryFn: async ({
      pageParam,
    }: {
      pageParam: { page: number; perPage: number };
    }) => {
      const resp = await api.getRepositoryLocatePathname({
        repository: props.repository,
        importerType: snapshot.sources[source].importer.type,
        importerOrigin: snapshot.sources[source].importer.origin,
        importerDirectory: snapshot.sources[source].importer.directory.Value,
        path: vfsEntry.path,
        pagination: { page: pageParam.page, perPage: pageParam.perPage },
      });

      let lastEntry: RepositoryLocatePathname | undefined = undefined;

      if (pageParam.page === 0 && resp.total > resp.items.length) {
        const respLastEntry = await api.getRepositoryLocatePathname({
          repository: props.repository,
          importerType: snapshot.sources[source].importer.type,
          importerOrigin: snapshot.sources[source].importer.origin,
          importerDirectory: snapshot.sources[source].importer.directory.Value,
          path: vfsEntry.path,
          pagination: { page: resp.total - 1, perPage: 1 },
        });

        lastEntry = respLastEntry.items[0];
      }

      return {
        page: pageParam.page,
        perPage: pageParam.perPage,
        response: resp,
        lastEntry,
      };
    },

    getNextPageParam: (lastPage: {
      page: number;
      perPage: number;
      response: { total: number };
    }) => {
      if ((lastPage.page + 1) * lastPage.perPage >= lastPage.response.total) {
        return null;
      }
      return { page: lastPage.page + 1, perPage: lastPage.perPage };
    },
  };
};

export const GetMountQueryOptions = (
  api: KlosetAPI,
  ...props: Parameters<getMountFn>
) => {
  return queryOptions({
    queryKey: klosetKeys.mount(...props),
    queryFn: () => {
      if (!api.mount) {
        throw new Error("Mount API is not available");
      }
      return api.mount.get(...props);
    },
  });
};

export const BrowseMountQueryOptions = (
  api: KlosetAPI,
  ...props: Parameters<browseMountFn>
) => {
  return queryOptions({
    queryKey: klosetKeys.mountBrowsePath(...props),
    queryFn: () => {
      if (!api.mount) {
        throw new Error("Mount API is not available");
      }
      return api.mount.browse(...props);
    },
  });
};

export const GetMountFileDownloadQueryOptions = (
  api: KlosetAPI,
  mountId: string,
  path: string
) => {
  return queryOptions({
    queryKey: klosetKeys.mountFileDownload({ mountId, path }),
    queryFn: async () => {
      if (!api.mount) {
        throw new Error("Mount API is not available");
      }

      const signatureResponse = await api.mount.getFileDownloadSignature({
        mountId,
        path,
      });
      const signature = signatureResponse.signature;

      const fileInfo = await api.mount.getFileInfo({
        mountId,
        path,
        signature,
      });
      return fileInfo;
    },
  });
};
