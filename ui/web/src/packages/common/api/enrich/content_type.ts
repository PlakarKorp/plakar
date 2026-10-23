const TEXT_APPLICATION_TYPES = [
  "application/xml",
  "application/javascript",
  "application/x-yaml",
  "application/yaml",
  "application/toml",
];

export function ContentType(value: string) {
  // Drop the parameters ("; charset=utf-8") and match on the media type
  // alone: a substring match sent application/vnd.oasis.opendocument.text to
  // the text viewer and application/x-iso9660-image to the image viewer.
  const mime = value.split(";")[0].trim();
  if (mime.startsWith("text/")) {
    return { resolved: "text", raw: value } as const;
  } else if (mime === "application/json" || mime.endsWith("+json")) {
    return { resolved: "json", raw: value } as const;
  } else if (mime.startsWith("image/")) {
    return { resolved: "image", raw: value } as const;
  } else if (mime === "application/pdf") {
    return { resolved: "pdf", raw: value } as const;
  } else if (mime.startsWith("video/")) {
    return { resolved: "video", raw: value } as const;
  } else if (mime.startsWith("audio/")) {
    return { resolved: "audio", raw: value } as const;
  } else if (TEXT_APPLICATION_TYPES.includes(mime)) {
    return { resolved: "text", raw: value } as const;
  }
  return { resolved: "unknown", raw: value } as const;
}
