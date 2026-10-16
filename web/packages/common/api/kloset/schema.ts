import std_path from "path";
import z from "zod";

import { Checksum, Date, Mode, Path, Size } from "../enrich";
import { PaginationParams } from "../pagination";

export const ReadFileParamsSchema = z.object({
  render: z.enum(["code", "text", "text_styled", "auto"]).optional(),
  download: z.boolean().optional(),
});

export type ReadFileParams = z.infer<typeof ReadFileParamsSchema>;

// ─── Shared internal schemas ─────────────────────────────────────────────────

const VersionSchema = z.number().transform((version) => {
  const major = (version & 0xff0000) >> 16;
  const minor = (version & 0x00ff00) >> 8;
  const patch = version & 0x0000ff;
  return { major, minor, patch, str: `${major}.${minor}.${patch}` };
});

const ClassificationSchema = z.object({
  analyzer: z.string(),
  classes: z.array(z.string()),
});

const SummaryBelowSchema = z.object({
  directories: z.number(),
  files: z.number(),
  symlinks: z.number(),
  devices: z.number(),
  pipes: z.number(),
  sockets: z.number(),

  children: z.number(),

  setuid: z.number(),
  setgid: z.number(),
  sticky: z.number(),

  objects: z.number(),
  chunks: z.number(),

  min_size: z.number(),
  max_size: z.number(),
  size: z.number().transform(Size),

  min_mod_time: z.number(),
  max_mod_time: z.number(),

  min_entropy: z.number(),
  max_entropy: z.number(),
  hi_entropy: z.number(),
  lo_entropy: z.number(),

  MIME_audio: z.number(),
  MIME_video: z.number(),
  MIME_image: z.number(),
  MIME_text: z.number(),
  MIME_application: z.number(),
  MIME_other: z.number(),

  errors: z.number(),
});

const SummaryDirectorySchema = SummaryBelowSchema.extend({
  avg_size: z.number(),
  sum_entropy: z.number(),
  avg_entropy: z.number(),
});

const SummarySchema = z
  .object({
    directory: SummaryDirectorySchema,
    below: SummaryBelowSchema,
  })
  .transform((data) => ({
    ...data,
    Total: {
      directories: data.directory.directories + data.below.directories,
      files: data.directory.files + data.below.files,
      symlinks: data.directory.symlinks + data.below.symlinks,
      devices: data.directory.devices + data.below.devices,
      pipes: data.directory.pipes + data.below.pipes,
      sockets: data.directory.sockets + data.below.sockets,

      children: data.directory.children + data.below.children,

      setuid: data.directory.setuid + data.below.setuid,
      setgid: data.directory.setgid + data.below.setgid,
      sticky: data.directory.sticky + data.below.sticky,

      objects: data.directory.objects + data.below.objects,
      chunks: data.directory.chunks + data.below.chunks,

      size: Size(data.directory.size.Value + data.below.size.Value),

      hi_entropy: data.directory.hi_entropy + data.below.hi_entropy,
      low_entropy: data.directory.lo_entropy + data.below.lo_entropy,

      MIME_audio: data.directory.MIME_audio + data.below.MIME_audio,
      MIME_video: data.directory.MIME_video + data.below.MIME_video,
      MIME_image: data.directory.MIME_image + data.below.MIME_image,
      MIME_text: data.directory.MIME_text + data.below.MIME_text,
      MIME_application:
        data.directory.MIME_application + data.below.MIME_application,
      MIME_other: data.directory.MIME_other + data.below.MIME_other,

      errors: data.directory.errors + data.below.errors,
    },
  }));

// ─── Repository info ──────────────────────────────────────────────────────────

const RepositoryInfoConfigurationSchema = z.object({
  timestamp: z.string().transform(Date),
  repository_id: z.string().transform(Checksum),
  hashing: z
    .object({
      algorithm: z.string(),
      bits: z.number(),
    })
    .nullable(),
  chunking: z
    .object({
      algorithm: z.string(),
    })
    .nullable(),
  compression: z
    .object({
      algorithm: z.string(),
    })
    .nullable(),
  encryption: z
    .object({
      data_algorithm: z.string(),
    })
    .nullable(),
});

const RepositoryInfoSnapshotsSchema = z.object({
  total: z.number(),
  storage_size: z
    .number()
    .transform((value) => (value < 0 ? null : Size(value))),
  logical_size: z
    .number()
    .transform((value) => (value < 0 ? null : Size(value))),
  efficiency: z.number().transform((value) => (value < 0 ? null : value)),
  snapshots_per_day: z.array(z.number()),
});

export const RepositoryInfoSchema = z.object({
  location: z.string(),
  browsable: z.boolean(),
  snapshots: RepositoryInfoSnapshotsSchema,
  configuration: RepositoryInfoConfigurationSchema,
  os: z.string(),
  arch: z.string(),
});

export type RepositoryInfo = z.infer<typeof RepositoryInfoSchema>;

export const GetRepositoryInfoSchema = z.object({
  item: RepositoryInfoSchema,
});

export type getRepositoryInfoFn = (props: {
  repository: string | undefined;
}) => Promise<z.infer<typeof GetRepositoryInfoSchema>>;

// ─── Repository importer types ────────────────────────────────────────────────

export const ImporterTypeSchema = z.object({
  name: z.string(),
});

export const GetRepositoryImporterTypesSchema = z.object({
  total: z.number(),
  items: z.array(ImporterTypeSchema),
});

export type getRepositoryImporterTypesFn = (props: {
  repository: string | undefined;
}) => Promise<z.infer<typeof GetRepositoryImporterTypesSchema>>;

// ─── Snapshot ─────────────────────────────────────────────────────────────────

const IdentitySchema = z.object({
  identifier: z.string(),
  public_key: z.string().nullable(),
});

const ContextSchema = z.object({
  key: z.string(),
  value: z.string(),
});

const ImporterSchema = z.object({
  type: z.string(),
  origin: z.string(),
  directory: z.string(),
});

const IndexSchema = z.object({
  name: z.string(),
  type: z.string(),
  value: z.string(),
});

const RootSchema = z.object({
  root: z.string(),
  xattrs: z.string(),
  errors: z.string(),
});

const SourceSchema = z.object({
  importer: ImporterSchema,
  root: RootSchema,
  indexes: z.array(IndexSchema),
  summary: SummarySchema,
});

const DurationSchema = z.number().transform((duration) => ({
  ns: duration,
  ms: Math.floor(duration / 1e6),
  seconds: Math.floor(duration / 1e9),
  minutes: Math.floor(duration / (60 * 1e9)),
  hours: Math.floor(duration / (60 * 60 * 1e9)),
}));

export const SnapshotSchema = z
  .object({
    version: VersionSchema,
    identifier: z.string().transform(Checksum),
    timestamp: z.string().transform(Date),
    duration: DurationSchema,
    identity: IdentitySchema,
    name: z.string(),
    category: z.string(),
    environment: z.string(),
    perimeter: z.string(),
    classifications: z.array(ClassificationSchema),
    tags: z.array(z.string()),
    legal_hold: z.boolean().optional().default(false),
    context: z.array(ContextSchema).transform((entries) => {
      const objects = entries.reduce(
        (acc, item) => {
          acc[item.key] = item.value;
          return acc;
        },
        {} as Record<string, string | undefined>
      );
      return {
        Objects: objects,
        Architecture: objects["Architecture"] ?? "",
        Client: objects["Client"] ?? "",
        CommandLine: objects["CommandLine"] ?? "",
        Hostname: objects["Hostname"] ?? "",
        MachineID: objects["MachineID"] ?? "",
        MacProcs: objects["MacProcs"] ?? "",
        NumCPU: objects["NumCPU"] ?? "",
        OperatingSystem: objects["OperatingSystem"] ?? "",
        ProcessID: objects["ProcessID"] ?? "",
        Username: objects["Username"] ?? "",
      };
    }),
    sources: z.array(SourceSchema),
  })
  .transform((data) => ({
    ...data,
    total: {
      size: Size(
        data.sources.reduce(
          (acc, source) => acc + source.summary.Total.size.Value,
          0
        )
      ),
      files: data.sources.reduce(
        (acc, source) => acc + source.summary.Total.files,
        0
      ),
      directories: data.sources.reduce(
        (acc, source) => acc + source.summary.Total.directories,
        0
      ),
      errors: data.sources.reduce(
        (acc, source) => acc + source.summary.Total.errors,
        0
      ),
    },
    sources: data.sources.map((source) => ({
      ...source,
      importer: {
        ...source.importer,
        directory: Path(source.importer.directory, data.context.Username),
      },
    })),
  }));

export type SnapshotType = z.infer<typeof SnapshotSchema>;

export const getSnapshotSchema = z.object({
  item: SnapshotSchema,
});

export type getSnapshotFn = (props: {
  repository: string | undefined;
  snapshot: string;
}) => Promise<z.infer<typeof getSnapshotSchema>>;

// ─── Snapshots list ───────────────────────────────────────────────────────────

export const GetRepositorySnapshotsSchema = z.object({
  total: z.number(),
  items: z.array(SnapshotSchema),
});

export type getSnapshotsListFn = (
  props: {
    repository: string | undefined;
    importer?: string;
    pagination: PaginationParams;
    since?: Date;
  },
  init?: RequestInit
) => Promise<z.infer<typeof GetRepositorySnapshotsSchema>>;

// ─── VFS ──────────────────────────────────────────────────────────────────────

const FileInfoSchema = z.object({
  name: z.string(),
  size: z.number().transform(Size),
  mode: z.number().transform(Mode),
  mod_time: z.string().transform(Date),
  dev: z.number(),
  ino: z.number(),
  uid: z.number(),
  gid: z.number(),
  nlink: z.number(),
  username: z.string(),
  groupname: z.string(),
});

const CustomMetadataSchema = z.object({
  key: z.string(),
  value: z.string(),
});

const ObjectSchema = z.object({
  version: VersionSchema,
  contentMAC: z.string(),
  content_type: z.string(),
  entropy: z.number(),
  flags: z.number(),
});

export const VFSEntrySchema = z
  .object({
    version: z.number(),
    parent_path: z.string(),
    file_info: FileInfoSchema,

    summary: SummarySchema.optional(),

    symlink_target: z.string().optional(),
    object: ObjectSchema.optional(),

    alternate_data_streams: z.array(z.string()).optional(),
    security_descriptor: z.string().optional(),
    file_attributes: z.number().optional(),

    extended_attributes: z.array(z.string()).optional(),

    classifications: z.array(ClassificationSchema).optional(),
    custom_metadata: z.array(CustomMetadataSchema).optional(),
    tags: z.array(z.string()).optional(),
    mac: z.string(),
  })
  .transform((data) => ({
    ...data,
    path: std_path.normalize(
      data.parent_path +
        (data.parent_path.endsWith("/") ? "" : "/") +
        data.file_info.name
    ),
  }));

export type VFSEntryType = z.infer<typeof VFSEntrySchema>;

export const getSnapshotPathChildrenSchema = z.object({
  total: z.number(),
  items: z.array(VFSEntrySchema),
});

export type getSnapshotPathChildrenFn = (props: {
  repository: string | undefined;
  snapshot: string;
  source: number;
  path: string;
  pagination: PaginationParams;
}) => Promise<z.infer<typeof getSnapshotPathChildrenSchema>>;

export const getSnapshotPathSearchSchema = z.object({
  has_next: z.boolean(),
  items: z.array(VFSEntrySchema),
});

export type getSnapshotPathSearchFn = (props: {
  repository: string | undefined;
  snapshot: string;
  source: number;
  path: string;
  pagination: PaginationParams;
  recursive: boolean;
  mimes?: readonly string[];
  filename?: string;
}) => Promise<z.infer<typeof getSnapshotPathSearchSchema>>;

export function enrichGetSnapshotPathChildren(
  snapshot: VFSEntryType,
  entries: VFSEntryType[]
) {
  return entries.map((child) => {
    const size = child.file_info.mode.isDir()
      ? (child.summary?.Total.size.Value ?? 0)
      : child.file_info.size.Value;

    return {
      ...child,
      weight: {
        directory: snapshot.summary?.Total.size.Value
          ? (size * 100) / snapshot.summary.Total.size.Value
          : 0,
      },
    };
  });
}

export type EnrichedChildEntryType = ReturnType<
  typeof enrichGetSnapshotPathChildren
>[number];

// ─── VFS errors ───────────────────────────────────────────────────────────────

const SnapshotPathErrorSchema = z.object({
  name: z.string(),
  error: z.string(),
});

export type SnapshotPathError = z.infer<typeof SnapshotPathErrorSchema>;

export const getSnapshotPathErrorsSchema = z.object({
  total: z.number(),
  items: z.array(SnapshotPathErrorSchema),
});

export type getSnapshotPathErrorsFn = (props: {
  repository: string | undefined;
  snapshot: string;
  source: number;
  path: string;
  pagination: PaginationParams;
}) => Promise<z.infer<typeof getSnapshotPathErrorsSchema>>;

// ─── VFS chunks ───────────────────────────────────────────────────────────────

const ChunkSchema = z.object({
  version: VersionSchema,
  contentMAC: z.string(),
  length: z.number(),
  entropy: z.number(),
  flags: z.number(),
});

export const getSnapshotPathChunksSchema = z.object({
  total: z.number(),
  items: z.array(ChunkSchema),
});

export type getSnapshotPathChunksFn = (props: {
  repository: string | undefined;
  snapshot: string;
  path: string;
  pagination: PaginationParams;
}) => Promise<z.infer<typeof getSnapshotPathChunksSchema>>;

// ─── Snapshot path ────────────────────────────────────────────────────────────

export const getSnapshotPathSchema = z.object({
  item: VFSEntrySchema,
});

export type getSnapshotPathFn = (props: {
  repository: string | undefined;
  snapshot: string;
  source: number;
  path: string;
}) => Promise<z.infer<typeof getSnapshotPathSchema>>;

// ─── Snapshot reader sign URL ─────────────────────────────────────────────────

export const ReaderSignUrlSchema = z.object({
  item: z.object({
    signature: z.string(),
  }),
});

export type generateSnapshotSignedURLFn = (props: {
  repository: string | undefined;
  snapshot: string;
  source: number;
  path: string;
}) => Promise<
  | {
      status: "ok";
      value: string;
    }
  | {
      status: "glacier";
    }
>;

// ─── Locate pathname ──────────────────────────────────────────────────────────

export const RepositoryLocatePathnameSchema = z.object({
  snapshot: SnapshotSchema,
  vfs_entry: VFSEntrySchema,
});

export type RepositoryLocatePathname = z.infer<
  typeof RepositoryLocatePathnameSchema
>;

export const GetRepositoryLocatePathnameSchema = z.object({
  total: z.number(),
  items: z.array(RepositoryLocatePathnameSchema),
});

export type getRepositoryLocatePathnameFn = (props: {
  repository: string | undefined;
  importerType: string;
  importerOrigin: string;
  importerDirectory: string;
  path: string;
  pagination: PaginationParams;
}) => Promise<z.infer<typeof GetRepositoryLocatePathnameSchema>>;

// ─── Downloader ───────────────────────────────────────────────────────────────

export const downloadSnapshotVFSSchema = z.object({
  id: z.string(),
});

export type downloadSnapshotVFSFn = (props: {
  repository: string | undefined;
  snapshot: string;
  paths: string[];
}) => Promise<
  z.infer<typeof downloadSnapshotVFSSchema> & {
    url: string;
  }
>;

// ─── Mount ───────────────────────────────────────────────────────────────
export const MountSchema = z.object({
  id: z.string(),
  type: z.string(),
  state: z.enum(["pending", "up", "down"]),
  store_id: z.string(),
  snapshot_id: z.string(),
  // Currently, the API doesn't return the source number of the snapshot
  // containing snapshot_path, so we default to 0.
  source: z.number().optional().default(0),
  snapshot_path: z.string(),
});

export type getMountFn = (props: {
  mountId: string;
}) => Promise<z.infer<typeof MountSchema>>;

export const CreateMountSchema = z.object({
  mount_id: z.string(),
});

export type createMountFn = (props: {
  repository: string | undefined;
  snapshot: string;
  source: number;
  path: string;
}) => Promise<z.infer<typeof CreateMountSchema>>;

export type deleteMountFn = (props: {
  repository: string | undefined;
  mountId: string;
}) => Promise<void>;

export const MountFileEntrySchema = z.object({
  name: z.string(),
  size: z.number().transform(Size),
  mode: z.number().transform(Mode),
  mod_time: z.string().transform(Date),
});

export type MountFileEntry = z.infer<typeof MountFileEntrySchema>;

export const BrowseMountResponseSchema = z.array(MountFileEntrySchema);

export type browseMountFn = (props: {
  mountId: string;
  path: string;
}) => Promise<z.infer<typeof BrowseMountResponseSchema>>;

export const getMountFileDownloadSignatureSchema = z.object({
  signature: z.string(),
});

export type getMountFileDownloadSignatureURLFn = (props: {
  mountId: string;
  path: string;
}) => Promise<z.infer<typeof getMountFileDownloadSignatureSchema>>;

export const getMountFileInfoSchema = z.object({
  content_type: z.string(),
  content_length: z.string(),
  getPublicURL: z.function({
    input: [ReadFileParamsSchema.optional()],
    output: z.string(),
  }),
});

export type getMountFileInfoFn = (props: {
  mountId: string;
  path: string;
  signature: string;
}) => Promise<z.infer<typeof getMountFileInfoSchema>>;
