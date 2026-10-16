import { APIResponseBadRequest } from "@plakar-ui/common/api/errors";
import {
  downloadSnapshotVFSFn,
  downloadSnapshotVFSSchema,
  generateSnapshotSignedURLFn,
  GetRepositoryImporterTypesSchema,
  GetRepositoryInfoSchema,
  getRepositoryLocatePathnameFn,
  GetRepositoryLocatePathnameSchema,
  GetRepositorySnapshotsSchema,
  getSnapshotFn,
  getSnapshotPathChildrenFn,
  getSnapshotPathChildrenSchema,
  getSnapshotPathChunksFn,
  getSnapshotPathChunksSchema,
  getSnapshotPathErrorsFn,
  getSnapshotPathErrorsSchema,
  getSnapshotPathFn,
  getSnapshotPathSchema,
  getSnapshotPathSearchFn,
  getSnapshotPathSearchSchema,
  getSnapshotSchema,
  getSnapshotsListFn,
  ReaderSignUrlSchema,
} from "@plakar-ui/common/api/kloset";
import { getPaginationParams } from "@plakar-ui/common/api/pagination";

import { CallApi, PLAKAR_URL } from "~/api/http";

export function getKlosetURL() {
  return `${PLAKAR_URL}/api`;
}

export async function getRepositoryInfo() {
  const url = "/api/repository/info";
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }
  return GetRepositoryInfoSchema.parse(data);
}

export async function getRepositoryImporterTypes() {
  const url = "/api/repository/importer-types";
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }
  return GetRepositoryImporterTypesSchema.parse(data);
}

export async function getRepositoryLocatePathname(
  ...[
    { importerType, importerOrigin, importerDirectory, path, pagination },
  ]: Parameters<getRepositoryLocatePathnameFn>
) {
  const qs = new URLSearchParams({
    sort: "-Timestamp",
    ...getPaginationParams(pagination),
    importerType,
    importerOrigin,
    importerDirectory,
    resource: path,
  });
  const url = `/api/repository/locate-pathname?${qs}`;
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }
  return GetRepositoryLocatePathnameSchema.parse(data);
}

export async function getSnapshotsList(
  ...[{ importer, pagination, since }]: Parameters<getSnapshotsListFn>
) {
  const qs = new URLSearchParams({
    sort: "-Timestamp",
    ...getPaginationParams(pagination),
    importer: importer ?? "",
    since: since ? since.toISOString() : "",
  });
  const url = `/api/repository/snapshots?${qs}`;
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }
  return GetRepositorySnapshotsSchema.parse(data);
}

export async function getSnapshot(
  ...[{ snapshot }]: Parameters<getSnapshotFn>
) {
  const url = `/api/snapshot/${snapshot}`;
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  try {
    return getSnapshotSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function generateSnapshotSignedURL(
  ...[{ snapshot, path }]: Parameters<generateSnapshotSignedURLFn>
) {
  const url = `/api/snapshot/reader-sign-url/${snapshot}:${encodeURIComponent(path)}`;
  let res: Response;

  try {
    res = await CallApi(url, { method: "POST" });
  } catch (e) {
    if (e instanceof APIResponseBadRequest) {
      return { status: "glacier" as const };
    }
    throw e;
  }

  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  try {
    const obj = ReaderSignUrlSchema.parse(data);
    return { status: "ok" as const, value: obj.item.signature };
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function getSnapshotPath(
  ...[{ snapshot, path }]: Parameters<getSnapshotPathFn>
) {
  const url = `/api/snapshot/vfs/${snapshot}:${encodeURIComponent(path)}`;
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  try {
    return getSnapshotPathSchema
      .transform((data) => ({
        ...data,
        SnapshotId: snapshot,
      }))
      .parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function getSnapshotPathChildren(
  ...[{ snapshot, path, pagination }]: Parameters<getSnapshotPathChildrenFn>
) {
  const qs = new URLSearchParams({
    ...getPaginationParams(pagination),
  });
  const url = `/api/snapshot/vfs/children/${snapshot}:${encodeURIComponent(path)}?${qs}`;
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  try {
    return getSnapshotPathChildrenSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function getSnapshotPathChunks(
  ...[{ snapshot, path, pagination }]: Parameters<getSnapshotPathChunksFn>
) {
  const qs = new URLSearchParams({
    ...getPaginationParams(pagination),
  });
  const url = `/api/snapshot/vfs/chunks/${snapshot}:${encodeURIComponent(path)}?${qs}`;
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  try {
    return getSnapshotPathChunksSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function downloadSnapshotVFS(
  ...[{ snapshot, paths }]: Parameters<downloadSnapshotVFSFn>
) {
  const url = `/api/snapshot/vfs/downloader/${snapshot}`;

  const response = await CallApi(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      items: paths.map((path) => ({ pathname: path })),
    }),
  });

  let data;

  try {
    data = await response.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${response.status} with an invalid JSON payload`
    );
  }

  try {
    return {
      ...downloadSnapshotVFSSchema.parse(data),
      url: `${PLAKAR_URL}/api/snapshot/vfs/downloader-sign-url/${data.id}?format=tarball`,
    };
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function getSnapshotPathErrors(
  ...[{ snapshot, path, pagination }]: Parameters<getSnapshotPathErrorsFn>
) {
  const qs = new URLSearchParams(getPaginationParams(pagination));
  const url = `/api/snapshot/vfs/errors/${snapshot}:${encodeURIComponent(path)}?${qs}`;
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  try {
    return getSnapshotPathErrorsSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}

export async function getSnapshotPathSearch(
  ...[
    { snapshot, path, pagination, recursive, mimes, filename },
  ]: Parameters<getSnapshotPathSearchFn>
) {
  const qs = new URLSearchParams({
    ...getPaginationParams(pagination),
    recursive: recursive.toString(),
  });

  mimes?.forEach((mime) => {
    qs.append("mime", mime);
  });

  if (filename) {
    qs.append("pattern", filename);
  }

  const url = `/api/snapshot/vfs/search/${snapshot}:${encodeURIComponent(path)}?${qs}`;
  const res = await CallApi(url);
  let data;

  try {
    data = await res.json();
  } catch {
    throw new Error(
      `the API endpoint ${url} returned HTTP/${res.status} with an invalid JSON payload`
    );
  }

  try {
    return getSnapshotPathSearchSchema.parse(data);
  } catch (e) {
    throw new Error(`Failed to parse ${url} response: ${e}`, { cause: e });
  }
}
