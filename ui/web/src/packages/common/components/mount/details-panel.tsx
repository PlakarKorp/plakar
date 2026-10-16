import { IconButton, Spinner, Typography } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import std_path from "path";
import { useCallback } from "react";

import { ContentType } from "@plakar-ui/common/api/enrich";
import { FileIcon } from "@plakar-ui/common/components";

import {
  BrowseMountQueryOptions,
  GetMountFileDownloadQueryOptions,
  KlosetAPI,
} from "../../api/kloset";
import {
  FilePreviewAudio,
  FilePreviewError,
  FilePreviewImage,
  FilePreviewPDF,
  FilePreviewText,
  FilePreviewVideo,
} from "../kloset/snapshot-browser/file-preview";

type MountDetailsPanelProps = {
  api: KlosetAPI;
  mountId: string;
  path: string;
};

export function MountDetailsPanel({
  api,
  mountId,
  path,
}: MountDetailsPanelProps) {
  const parentPath = std_path.dirname(path);
  const fileName = std_path.basename(path);

  const browseQuery = useQuery(
    BrowseMountQueryOptions(api, { mountId, path: parentPath })
  );
  const entry = browseQuery.data?.find((e) => e.name === fileName);
  const isNonRegular = entry !== undefined && !entry.mode.isRegular();

  const { data, isPending, error } = useQuery({
    ...GetMountFileDownloadQueryOptions(api, mountId, path),
    enabled: !isNonRegular,
  });

  const handleDownload = useCallback(async () => {
    if (!data) return;
    const response = await fetch(data.getPublicURL());
    const blob = await response.blob();
    const blobUrl = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = blobUrl;
    a.download = std_path.basename(path);
    a.click();
    URL.revokeObjectURL(blobUrl);
  }, [data, path]);

  const type = data ? ContentType(data.content_type) : null;

  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center justify-between px-4">
        <Typography as="h2" className="text-primary flex items-center gap-2">
          <FileIcon contentType={data?.content_type} />
          {fileName}
        </Typography>
        <IconButton
          aria-label="Download file"
          tooltip="Download file"
          icon="download"
          variant="tertiary"
          size="sm"
          isDisabled={!data || isNonRegular}
          onPress={handleDownload}
        />
      </div>

      {isPending && !isNonRegular ? (
        <Spinner />
      ) : isNonRegular ? (
        <FilePreviewError type="warning">
          Cannot preview non-regular files.
        </FilePreviewError>
      ) : error ? (
        <FilePreviewError type="error">{error.message}</FilePreviewError>
      ) : type?.resolved === "text" || type?.resolved === "json" ? (
        <FilePreviewText
          getUrl={(render) => data!.getPublicURL({ render })}
          contentType={data!.content_type}
        />
      ) : type?.resolved === "image" ? (
        <FilePreviewImage url={data!.getPublicURL()} />
      ) : type?.resolved === "pdf" ? (
        <FilePreviewPDF url={data!.getPublicURL()} />
      ) : type?.resolved === "video" ? (
        <FilePreviewVideo url={data!.getPublicURL()} />
      ) : type?.resolved === "audio" ? (
        <FilePreviewAudio url={data!.getPublicURL()} />
      ) : (
        <FilePreviewError type="warning">
          Unknown file type: {type?.raw}
        </FilePreviewError>
      )}
    </div>
  );
}
