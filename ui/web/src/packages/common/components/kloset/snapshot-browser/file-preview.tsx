import {
  Alert,
  Button,
  Id,
  Select,
  Spinner,
  Typography,
} from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import clsx from "clsx";
import { ComponentProps, ReactNode, useState, useEffect } from "react";
import { Key } from "react-aria-components";

import { Size } from "@plakar-ui/common/api/enrich";
import {
  ReadFileParams,
  SnapshotReader,
  SnapshotReaderSignUrlQueryOptions,
  SnapshotType,
  useKlosetAPI,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";
import { useSpinner } from "@plakar-ui/common/hooks";
import { useLocalStorage, usePreview } from "@plakar-ui/common/hooks";

import { MountFileButton } from "./mount-file-button";

type FilePreviewProps = {
  snapshot: SnapshotType;
  source: number;
  entry: VFSEntryType;
  onMount?: (mountId: string) => void;
};

export function FilePreview({
  snapshot,
  source,
  entry,
  onMount,
}: FilePreviewProps) {
  const { preview } = usePreview();
  const { api } = useKlosetAPI();

  if (
    preview &&
    api.mount &&
    (entry.path.endsWith(".qcow2") ||
      entry.path.endsWith(".vmdk") ||
      entry.path.endsWith(".raw"))
  ) {
    return (
      <MountFileAlert
        snapshot={snapshot}
        source={source}
        vfsEntry={entry}
        onMount={onMount}
      />
    );
  }

  if (!entry.file_info.mode.isRegular()) {
    return (
      <FilePreviewError type="warning">
        Cannot preview non-regular files.
      </FilePreviewError>
    );
  }
  return (
    <FilePreviewRegular snapshot={snapshot} source={source} entry={entry} />
  );
}

function FilePreviewRegular({ snapshot, source, entry }: FilePreviewProps) {
  const warningSize = Size(1024 * 1024 * 50); // 50MB
  const maxSize = Size(1024 * 1024 * 100); // 100MB
  const [warning, setWarning] = useState(
    entry.file_info.size.Value >= warningSize.Value
  );
  const { api, repository } = useKlosetAPI();
  const { data, error, isPending } = useQuery(
    SnapshotReaderSignUrlQueryOptions(api, {
      repository,
      snapshot: snapshot.identifier.Value,
      source,
      path: entry.path,
    })
  );

  if (isPending) {
    return <Spinner />;
  }

  if (error) {
    return <FilePreviewError type="error">{error.message}</FilePreviewError>;
  }

  const signature = data.signature;

  if (signature.status !== "ok") {
    return (
      <FilePreviewError type="warning">
        The file is stored in a cold storage and cannot be previewed.
      </FilePreviewError>
    );
  }

  const reader = new SnapshotReader(
    api,
    repository,
    snapshot.identifier.Value,
    source,
    entry.path
  );

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      {entry.file_info.size.Value >= maxSize.Value ? (
        // File too big to be displayed in the UI
        <FilePreviewError type="warning">
          The file is {entry.file_info.size.Pretty}, exceeding the{" "}
          {maxSize.Pretty} display size limit.
        </FilePreviewError>
      ) : warning ? (
        // File is big, but we can still display it. Require user confirmation before displaying
        <FilePreviewError
          type="warning"
          actions={[
            <Button onPress={() => setWarning(false)} variant="secondary">
              Display the file anyway
            </Button>,
          ]}
        >
          <div className="flex flex-col gap-4">
            <p>
              The file is {entry.file_info.size.Pretty}, which may take a while
              to load.
            </p>
          </div>
        </FilePreviewError>
      ) : data.type.resolved === "text" || data.type.resolved === "json" ? (
        // Text or JSON
        <FilePreviewText
          getUrl={(render) => reader.getPublicUrl(signature.value, { render })}
          entropy={entry.object?.entropy ?? entry.entropy}
          contentType={entry.contentType}
          contentMAC={entry.object?.contentMAC}
        />
      ) : data.type.resolved === "image" ? (
        <FilePreviewImage url={reader.getPublicUrl(signature.value)} />
      ) : data.type.resolved === "pdf" ? (
        <FilePreviewPDF url={reader.getPublicUrl(signature.value)} />
      ) : data.type.resolved === "video" ? (
        <FilePreviewVideo url={reader.getPublicUrl(signature.value)} />
      ) : data.type.resolved === "audio" ? (
        <FilePreviewAudio url={reader.getPublicUrl(signature.value)} />
      ) : (
        // Unknown
        <FilePreviewError type="warning">
          Unknown file type: {data.type.raw}
        </FilePreviewError>
      )}
    </div>
  );
}

type FilePreviewErrorProps = ComponentProps<typeof Alert> & {
  type: "error" | "warning";
};

export function FilePreviewError({ type, ...props }: FilePreviewErrorProps) {
  return (
    <div className="p-4">
      <Alert
        variant={type === "error" ? "error" : "warning"}
        title="Unable to preview file"
        {...props}
      />
    </div>
  );
}

type FilePreviewIframeProps = {
  url: string;
  footer?: ReactNode;
};

export function FilePreviewIframe({ url, footer }: FilePreviewIframeProps) {
  const [loaded, setLoaded] = useState(false);
  const showSpinner = useSpinner(loaded);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {showSpinner && <Spinner className="bg-[#282a36]" />}
      <iframe
        title="File preview"
        src={url}
        onLoad={() => setLoaded(true)}
        sandbox=""
        className={clsx(
          "h-full min-h-0 w-full flex-1",
          showSpinner && "hidden"
        )}
        style={{
          colorScheme: "dark",
          backgroundColor: "black",
        }}
      />
      {footer}
    </div>
  );
}

type FilePreviewTextProps = {
  getUrl: (render: ReadFileParams["render"]) => string;
  entropy?: number;
  contentType?: string;
  contentMAC?: string;
};

export function FilePreviewText({
  getUrl,
  entropy,
  contentType,
  contentMAC,
}: FilePreviewTextProps) {
  const previewOptions = [
    { id: "highlight", name: "Syntax highlighting" },
    { id: "plain", name: "Plain text preview" },
    { id: "rendered", name: "Rendered" },
  ];

  const [previewType, setPreviewType] = useLocalStorage<string>(
    "filePreviewType",
    "highlight"
  );

  const render =
    previewType === "highlight"
      ? "code"
      : previewType === "plain"
        ? "text_styled"
        : "auto";

  const url = getUrl(render);

  const handleChange = (key: Key | Key[] | null) => {
    if (key !== previewType) {
      setPreviewType(String(key));
    }
  };

  return (
    <FilePreviewIframe
      key={url}
      url={url}
      footer={
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
              <Id value={contentMAC} size="xs" className="hidden md:block"></Id>
            )}
          </Typography>
          <Select
            value={previewType}
            defaultValue={previewOptions[0].id}
            onChange={handleChange}
            variant="terminal"
            className="w-46"
          >
            <Select.ListBox>
              {previewOptions.map((option) => (
                <Select.Item key={option.id} {...option} />
              ))}
            </Select.ListBox>
          </Select>
        </div>
      }
    />
  );
}

type FilePreviewImageProps = {
  url: string;
};

export function FilePreviewImage({ url }: FilePreviewImageProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && isFullscreen) {
        setIsFullscreen(false);
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isFullscreen]);

  return (
    <>
      <div className="max-h-full max-w-full">
        <button
          className="cursor-zoom-in"
          onClick={() => setIsFullscreen(true)}
        >
          <img src={url} alt="Preview" className="w-full" />
        </button>
      </div>

      {isFullscreen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/90">
          <div className="relative h-full w-full">
            <img
              src={url}
              alt="Fullscreen preview"
              className="h-full w-full cursor-zoom-out object-contain"
              onClick={() => setIsFullscreen(false)}
            />
          </div>
        </div>
      )}
    </>
  );
}

type FilePreviewPDFProps = {
  url: string;
};

export function FilePreviewPDF({ url }: FilePreviewPDFProps) {
  return (
    <embed src={url} type="application/pdf" width="100%" height="100%"></embed>
  );
}

type FilePreviewVideoProps = {
  url: string;
};

export function FilePreviewVideo({ url }: FilePreviewVideoProps) {
  return (
    <video controls width="100%">
      <source src={url} />
    </video>
  );
}

type FilePreviewAudioProps = {
  url: string;
};

export function FilePreviewAudio({ url }: FilePreviewAudioProps) {
  return (
    <audio controls>
      <source src={url} />
    </audio>
  );
}

type MountFileAlertProps = {
  snapshot: SnapshotType;
  source: number;
  vfsEntry: VFSEntryType;
  onMount?: (mountId: string) => void;
};

function MountFileAlert({
  snapshot,
  source,
  vfsEntry,
  onMount,
}: MountFileAlertProps) {
  return (
    <div className="p-4">
      <Alert
        variant="default"
        title="Mount preview"
        actions={
          <MountFileButton
            snapshot={snapshot}
            source={source}
            vfsEntry={vfsEntry}
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
