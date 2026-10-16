type BlobDownload = {
  blob: Blob;
  filename: string;
};

function getFilenameFromContentDisposition(header: string) {
  // RFC 5987: filename*=UTF-8''...
  const utf8Match = header.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match) {
    return decodeURIComponent(utf8Match[1]);
  }

  const simpleMatch = header.match(/filename="?([^"]+)"?/i);
  if (simpleMatch) {
    return simpleMatch[1].split(";")[0].trim();
  }
  return null;
}

export async function readBlobDownload(
  res: Response,
  fallbackFilename: string
): Promise<BlobDownload> {
  const header = res.headers.get("Content-Disposition");
  const filename =
    (header ? getFilenameFromContentDisposition(header) : null) ??
    fallbackFilename;

  try {
    return { blob: await res.blob(), filename };
  } catch {
    throw new Error(
      `the API endpoint ${res.url} returned HTTP/${res.status} with an invalid response`
    );
  }
}

export function triggerBlobDownload({ blob, filename }: BlobDownload) {
  const url = URL.createObjectURL(blob);

  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.style.display = "none";

  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();

  URL.revokeObjectURL(url);
}
