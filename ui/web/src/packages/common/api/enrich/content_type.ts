export function ContentType(value: string) {
  if (value.includes("text")) {
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
  }
  return { resolved: "unknown", raw: value } as const;
}
