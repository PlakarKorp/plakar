import type { Address, Attachment } from "postal-mime";

export type EmailAddress = { name: string; address: string };

export type EmailAttachment = {
  filename: string;
  mimeType: string;
  content: Uint8Array<ArrayBuffer>;
};

export type ParsedEmail = {
  subject: string;
  from?: EmailAddress;
  to: EmailAddress[];
  cc: EmailAddress[];
  date?: Date;
  messageId?: string;
  html?: string;
  hasRemoteContent: boolean;
  text?: string;
  attachments: EmailAttachment[];
  // The raw message, decoded only when the Source view asks for it.
  bytes: ArrayBuffer;
};

export type EmailFlags = { read: boolean; flagged: boolean };

function toAddress(a: Address): EmailAddress {
  return a.group ? a.group[0] : { name: a.name, address: a.address };
}

function flatten(addresses: Address[] = []): EmailAddress[] {
  return addresses.flatMap((a) => (a.group ? a.group : [toAddress(a)]));
}

function toBase64(bytes: Uint8Array) {
  let binary = "";
  for (let i = 0; i < bytes.length; i += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  }
  return btoa(binary);
}

function contentId(part: Attachment) {
  return part.contentId?.replace(/^<|>$/g, "").toLowerCase();
}

// cid: URLs are percent-encoded (RFC 2392), Content-IDs are not.
function decodeCid(id: string) {
  try {
    return decodeURIComponent(id).toLowerCase();
  } catch {
    return id.toLowerCase();
  }
}

// Only embedded resources (src, background, url()) are inlined: a link to a
// part stays a link, and the part stays in the attachment list.
const EMBEDDED_CID =
  /(\b(?:src|background)\s*=\s*["']?\s*|url\(\s*["']?\s*)cid:([^"'\s)>]+)/gi;

// Replaces each embedded cid: reference with the part it names, as a data:
// URL, and tells which parts were used that way.
function inlineParts(html: string, parts: Attachment[]) {
  const byId = new Map(parts.map((part) => [contentId(part), part]));
  const urls = new Map<Attachment, string>();
  const inlined = html.replace(
    EMBEDDED_CID,
    (ref, prefix: string, id: string) => {
      const part = byId.get(decodeCid(id));
      if (!part) {
        return ref;
      }
      const content = new Uint8Array(part.content as ArrayBuffer);
      const url =
        urls.get(part) ?? `data:${part.mimeType};base64,${toBase64(content)}`;
      urls.set(part, url);
      return prefix + url;
    }
  );
  return { html: inlined, used: new Set(urls.keys()) };
}

const REMOTE_CONTENT = new RegExp(
  [
    String.raw`\b(?:src|srcset|background|poster)\s*=\s*["']?\s*(?:https?:)?//`,
    String.raw`url\(\s*["']?\s*(?:https?:)?//`,
    String.raw`@import\s+["']\s*(?:https?:)?//`,
    String.raw`<(?:link|image|use)\b[^>]*\bhref\s*=\s*["']?\s*(?:https?:)?//`,
  ].join("|"),
  "i"
);

export async function parseEmail(bytes: ArrayBuffer): Promise<ParsedEmail> {
  // Loaded on the first email opened, not with the explorer.
  const { default: PostalMime } = await import("postal-mime");
  const email = await PostalMime.parse(bytes);

  const { html, used } = email.html?.trim()
    ? inlineParts(email.html, email.attachments)
    : { html: undefined, used: new Set<Attachment>() };
  const date = email.date ? new Date(email.date) : undefined;
  return {
    subject: email.subject ?? "",
    from: email.from && toAddress(email.from),
    to: flatten(email.to),
    cc: flatten(email.cc),
    date: date && !isNaN(date.getTime()) ? date : undefined,
    messageId: email.messageId,
    html,
    hasRemoteContent: !!html && REMOTE_CONTENT.test(html),
    text: email.text?.trim() ? email.text : undefined,
    attachments: email.attachments
      .filter((part) => !used.has(part))
      .map((part) => ({
        filename: part.filename ?? "attachment",
        mimeType: part.mimeType,
        content: new Uint8Array(part.content as ArrayBuffer),
      })),
    bytes,
  };
}

// The Microsoft 365 integration names a message "<16 hex>.eml" and appends
// its flags only when one is set: "01bca9cde77e7388,RF.eml" is read and
// flagged. Other .eml names carry no flags.
const O365_MESSAGE = /^[0-9a-f]{16}(?:,([A-Z]*))?\.eml$/i;

export function emailFlags(fileName: string): EmailFlags | null {
  const match = fileName.match(O365_MESSAGE);
  if (!match) {
    return null;
  }
  const flags = (match[1] ?? "").toUpperCase();
  return { read: flags.includes("R"), flagged: flags.includes("F") };
}

// The frame runs no script, so the CSP is what keeps remote images, fonts and
// styles (tracking pixels) from loading until the user asks for them.
export function bodyDocument(
  html: string,
  { remoteContent }: { remoteContent: boolean }
) {
  const remote = remoteContent ? " https: http:" : "";
  const csp = `default-src 'none'; img-src data:${remote}; style-src 'unsafe-inline'${remote}; font-src data:${remote};`;
  return `<!doctype html><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="${csp}"><meta name="referrer" content="no-referrer"><base target="_blank"><style>body{margin:16px;font-family:system-ui,sans-serif;overflow-wrap:anywhere}img{max-width:100%;height:auto}</style>${html}`;
}
