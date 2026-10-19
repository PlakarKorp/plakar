import {
  Alert,
  Button,
  Icon,
  Id,
  Select,
  Spinner,
  Typography,
} from "@plakar-ui/ui/components";
import clsx from "clsx";
import { ReactNode, useState } from "react";
import { Button as RAButton, Key } from "react-aria-components";

import { ContentType, Size } from "@plakar-ui/common/api/enrich";
import {
  ReadFileParams,
  SnapshotType,
  useKlosetAPI,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";
import { FileIcon } from "@plakar-ui/common/components";
import {
  useLocalStorage,
  usePreview,
  useSpinner,
} from "@plakar-ui/common/hooks";

import { MountFileButton } from "./mount-file-button";
import { useEntryURL } from "./use-entry-url";

type ViewerVariant = "panel" | "stage";

const WARNING_SIZE = Size(1024 * 1024 * 50);
const MAX_SIZE = Size(1024 * 1024 * 100);

type ViewerProps = {
  variant: ViewerVariant;
  name: string;
  contentType: string;
  size?: ReturnType<typeof Size>;
  getUrl: (params?: ReadFileParams) => string;
  downloadUrl?: string;
  entropy?: number;
  contentMAC?: string;
  onExpand?: () => void;
  onEnded?: () => void;
};

export function Viewer({
  variant,
  name,
  contentType,
  size,
  getUrl,
  downloadUrl,
  entropy,
  contentMAC,
  onExpand,
  onEnded,
}: ViewerProps) {
  const type = ContentType(contentType);
  const isStreamed = type.resolved === "video" || type.resolved === "audio";
  const [isLargeConfirmed, setLargeConfirmed] = useState(false);
  const [isUnreadable, setUnreadable] = useState(false);
  const onError = () => setUnreadable(true);

  const notice = (title: string, children?: ReactNode, action?: ReactNode) => (
    <ViewerNotice
      variant={variant}
      title={title}
      downloadUrl={downloadUrl}
      action={action}
    >
      {children}
    </ViewerNotice>
  );

  if (!isStreamed && size && size.Value >= MAX_SIZE.Value) {
    return notice(
      "This file is too large to preview",
      `It is ${size.Pretty}, over the ${MAX_SIZE.Pretty} preview limit.`
    );
  }
  if (
    !isStreamed &&
    size &&
    size.Value >= WARNING_SIZE.Value &&
    !isLargeConfirmed
  ) {
    return notice(
      "This file is large",
      `It is ${size.Pretty} and may take a while to load.`,
      <Button variant="secondary" onPress={() => setLargeConfirmed(true)}>
        Preview anyway
      </Button>
    );
  }
  if (isUnreadable) {
    return notice("This browser can't display this file");
  }

  switch (type.resolved) {
    case "image":
      return (
        <ImageViewer
          variant={variant}
          url={getUrl()}
          name={name}
          onExpand={onExpand}
          onError={onError}
        />
      );
    case "video":
      return (
        <video
          src={getUrl()}
          controls
          autoPlay={variant === "stage"}
          playsInline
          onEnded={onEnded}
          onError={onError}
          className={
            variant === "stage"
              ? "max-h-[calc(100vh-11rem)] max-w-full rounded-md bg-black shadow-2xl"
              : "w-full"
          }
        />
      );
    case "audio":
      return (
        <div
          className={clsx(
            "flex w-full flex-col items-center gap-4 text-center",
            variant === "stage" ? "max-w-md" : "p-4"
          )}
        >
          {variant === "stage" && (
            <>
              <FileIcon contentType={contentType} size="3xl" />
              <Typography weight="semibold" className="text-white">
                {name}
              </Typography>
            </>
          )}
          <audio
            src={getUrl()}
            controls
            autoPlay={variant === "stage"}
            onEnded={onEnded}
            onError={onError}
            className="w-full"
          />
        </div>
      );
    case "pdf":
      return (
        <DocumentSheet variant={variant}>
          <embed
            src={getUrl()}
            type="application/pdf"
            className="h-full w-full"
          />
        </DocumentSheet>
      );
    case "text":
    case "json":
      return (
        <DocumentSheet variant={variant}>
          <TextViewer
            getUrl={getUrl}
            contentType={contentType}
            entropy={entropy}
            contentMAC={contentMAC}
          />
        </DocumentSheet>
      );
    case "unknown":
      return notice("No preview for this type of file", type.raw);
  }
}

type ViewerNoticeProps = {
  variant: ViewerVariant;
  title: string;
  tone?: "warning" | "error";
  children?: ReactNode;
  downloadUrl?: string;
  action?: ReactNode;
};

export function ViewerNotice({
  variant,
  title,
  tone = "warning",
  children,
  downloadUrl,
  action,
}: ViewerNoticeProps) {
  return (
    <div className={variant === "stage" ? "w-full max-w-md" : "p-4"}>
      <Alert
        variant={tone}
        title={title}
        actions={
          (action || downloadUrl) && (
            <>
              {action}
              {downloadUrl && (
                <Button
                  href={downloadUrl}
                  download
                  variant="secondary"
                  leftIcon="download"
                >
                  Download
                </Button>
              )}
            </>
          )
        }
      >
        {children}
      </Alert>
    </div>
  );
}

type EntryViewerProps = {
  variant: ViewerVariant;
  snapshot: SnapshotType;
  source: number;
  entry: VFSEntryType;
  onMount?: (mountId: string) => void;
  onExpand?: () => void;
  onEnded?: () => void;
};

export function EntryViewer({
  variant,
  snapshot,
  source,
  entry,
  onMount,
  onExpand,
  onEnded,
}: EntryViewerProps) {
  const { preview } = usePreview();
  const { api } = useKlosetAPI();
  const isRegular = entry.file_info.mode.isRegular();
  const { getUrl, isColdStorage, error } = useEntryURL({
    snapshot: snapshot.identifier.Value,
    source,
    path: entry.path,
    enabled: isRegular,
  });

  if (preview && api.mount && /\.(qcow2|vmdk|raw)$/.test(entry.path)) {
    return (
      <div className="p-4">
        <Alert
          variant="default"
          title="Mount preview"
          actions={
            <MountFileButton
              snapshot={snapshot}
              source={source}
              vfsEntry={entry}
              onMount={onMount}
            />
          }
        >
          This file appears to be a disk image. You can mount it to browse its
          contents.
        </Alert>
      </div>
    );
  }
  if (!isRegular) {
    return (
      <ViewerNotice
        variant={variant}
        title="Cannot preview non-regular files"
      />
    );
  }
  if (error) {
    return (
      <ViewerNotice
        variant={variant}
        tone="error"
        title="Unable to preview file"
      >
        {error.message}
      </ViewerNotice>
    );
  }
  if (isColdStorage) {
    return (
      <ViewerNotice variant={variant} title="This file is in cold storage">
        It is safely backed up, but it can't be previewed from cold storage.
      </ViewerNotice>
    );
  }
  if (!getUrl) {
    return variant === "stage" ? (
      <Icon type="spinner" size="2xl" className="animate-spin" />
    ) : (
      <Spinner />
    );
  }
  return (
    <Viewer
      variant={variant}
      name={entry.file_info.name}
      contentType={entry.contentType}
      size={entry.file_info.size}
      getUrl={getUrl}
      downloadUrl={getUrl({ download: true })}
      entropy={entry.object?.entropy ?? entry.entropy}
      contentMAC={entry.object?.contentMAC}
      onExpand={onExpand}
      onEnded={onEnded}
    />
  );
}

function ImageViewer({
  variant,
  url,
  name,
  onExpand,
  onError,
}: {
  variant: ViewerVariant;
  url: string;
  name: string;
  onExpand?: () => void;
  onError: () => void;
}) {
  const [isZoomed, setZoomed] = useState(false);

  if (variant === "panel") {
    const image = (
      <img src={url} alt={name} onError={onError} className="w-full" />
    );
    return onExpand ? (
      <RAButton
        aria-label="Open in the viewer"
        className="cursor-zoom-in"
        onPress={onExpand}
      >
        {image}
      </RAButton>
    ) : (
      image
    );
  }

  return (
    <div className={clsx("max-h-full max-w-full", isZoomed && "overflow-auto")}>
      <img
        src={url}
        alt={name}
        onClick={() => setZoomed((v) => !v)}
        onError={onError}
        className={clsx(
          "rounded shadow-2xl motion-safe:animate-[modal-fade_200ms]",
          isZoomed
            ? "max-w-none cursor-zoom-out"
            : "max-h-[calc(100vh-11rem)] max-w-full cursor-zoom-in object-contain"
        )}
      />
    </div>
  );
}

function DocumentSheet({
  variant,
  children,
}: {
  variant: ViewerVariant;
  children: ReactNode;
}) {
  return (
    <div
      className={clsx(
        "flex min-h-0 flex-col",
        variant === "stage"
          ? "bg-primary text-primary h-[calc(100vh-11rem)] w-full max-w-5xl overflow-hidden rounded-lg"
          : "h-full"
      )}
    >
      {children}
    </div>
  );
}

const RENDER_OPTIONS = [
  { id: "highlight", name: "Syntax highlighting", render: "code" },
  { id: "plain", name: "Plain text preview", render: "text_styled" },
  { id: "rendered", name: "Rendered", render: "auto" },
] as const;

function TextViewer({
  getUrl,
  contentType,
  entropy,
  contentMAC,
}: {
  getUrl: (params?: ReadFileParams) => string;
  contentType: string;
  entropy?: number;
  contentMAC?: string;
}) {
  const [renderId, setRenderId] = useLocalStorage<string>(
    "filePreviewType",
    "highlight"
  );
  const option =
    RENDER_OPTIONS.find((o) => o.id === renderId) ?? RENDER_OPTIONS[0];
  const url = getUrl({ render: option.render });
  const [loadedUrl, setLoadedUrl] = useState<string>();
  const showSpinner = useSpinner(loadedUrl === url);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {showSpinner && <Spinner className="bg-[#282a36]" />}
      <iframe
        key={url}
        title="File preview"
        src={url}
        onLoad={() => setLoadedUrl(url)}
        sandbox=""
        className={clsx(
          "h-full min-h-0 w-full flex-1",
          showSpinner && "hidden"
        )}
        style={{ colorScheme: "dark", backgroundColor: "black" }}
      />
      <div className="border-terminal bg-terminal flex w-full flex-wrap items-center justify-between border-t px-3 py-1">
        <Typography
          size="xs"
          className="text-terminal flex flex-1 flex-wrap items-baseline justify-start gap-x-6"
        >
          <span>{contentType}</span>
          {entropy && (
            <span className="hidden md:block">
              Entropy:{" "}
              <Typography as="span" size="sm" weight="medium">
                {entropy.toFixed(2)}
              </Typography>
            </span>
          )}
          {contentMAC && (
            <Id value={contentMAC} size="xs" className="hidden md:block" />
          )}
        </Typography>
        <Select
          value={option.id}
          onChange={(key: Key | Key[] | null) => {
            if (key !== null && !Array.isArray(key)) setRenderId(String(key));
          }}
          variant="terminal"
          className="w-46"
        >
          <Select.ListBox>
            {RENDER_OPTIONS.map(({ id, name }) => (
              <Select.Item key={id} id={id} name={name} />
            ))}
          </Select.ListBox>
        </Select>
      </div>
    </div>
  );
}
