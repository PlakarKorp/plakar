const TEXT_APPLICATION_TYPES = [
  "application/xml",
  "application/javascript",
  "application/x-yaml",
  "application/yaml",
  "application/toml",
];

export function ContentType(value: string) {
  if (value.startsWith("text/")) {
    return { resolved: "text", raw: value } as const;
  } else if (value.includes("json")) {
    return { resolved: "json", raw: value } as const;
  } else if (value.includes("image")) {
    return { resolved: "image", raw: value } as const;
  } else if (value.includes("application/pdf")) {
    return { resolved: "pdf", raw: value } as const;
  } else if (value.includes("video")) {
    return { resolved: "video", raw: value } as const;
  } else if (value.includes("audio")) {
    return { resolved: "audio", raw: value } as const;
  } else if (TEXT_APPLICATION_TYPES.includes(value.split(";")[0].trim())) {
    return { resolved: "text", raw: value } as const;
  }
  return { resolved: "unknown", raw: value } as const;
}
