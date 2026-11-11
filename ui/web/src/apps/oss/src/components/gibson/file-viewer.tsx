import { SnapshotReader, useKlosetAPI } from "@plakar-ui/common/api/kloset";
import { useEffect, useState } from "react";

import { TowerFile } from "./world";

/** How much of a file the viewer pulls: enough to read, not enough to choke on. */
const PEEK = 64 * 1024;

type Content =
  | { kind: "text"; text: string; truncated: boolean }
  | { kind: "hex"; dump: string; truncated: boolean }
  | { kind: "image"; url: string };

type FileViewerProps = {
  file: TowerFile;
  onClose: () => void;
};

/**
 * The content of a file, opened in-world: text as is, an image as is, and
 * anything else as a hex dump, the way a 1995 sysadmin would read it.
 */
export function FileViewer({ file, onClose }: FileViewerProps) {
  const { api } = useKlosetAPI();
  const [content, setContent] = useState<Content | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setContent(null);
    setError("");

    const load = async (): Promise<Content> => {
      const reader = new SnapshotReader(
        api,
        undefined,
        file.snapshotId,
        0,
        file.path
      );
      const signature = await reader.getSignature();
      if (signature.status !== "ok") {
        throw new Error("the file sits in cold storage");
      }
      const url = reader.getPublicUrl(signature.value);
      if (file.contentType.startsWith("image/")) {
        return { kind: "image", url };
      }
      const bytes = await peek(url, PEEK);
      const truncated = file.size > bytes.length;
      if (looksLikeText(bytes, file.contentType)) {
        return {
          kind: "text",
          text: new TextDecoder().decode(bytes),
          truncated,
        };
      }
      return { kind: "hex", dump: hexdump(bytes), truncated };
    };

    load()
      .then((loaded) => {
        if (!cancelled) {
          setContent(loaded);
        }
      })
      .catch((e: Error) => {
        if (!cancelled) {
          setError(e.message);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [api, file]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (["Escape", "Backspace", "Enter", "KeyX"].includes(event.code)) {
        event.preventDefault();
        onClose();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  return (
    <div className="absolute inset-0 flex items-center justify-center bg-black/40 p-6">
      <div className="gibson-panel flex max-h-full w-full max-w-4xl flex-col text-xs leading-5">
        <div className="flex items-center gap-4 border-b border-cyan-400/40 px-4 py-2">
          <div className="min-w-0 flex-1 truncate font-bold tracking-widest">
            FILE ACCESS // {file.name}
          </div>
          <div className="whitespace-nowrap opacity-70">
            {file.contentType || "unknown"} · {file.sizePretty}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="border border-current px-2 hover:bg-cyan-400/20"
          >
            [ESC] CLOSE
          </button>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4">
          {!content && !error && (
            <div className="gibson-blink">ACCESSING FILE...</div>
          )}
          {error && <div className="text-red-400">ACCESS DENIED: {error}</div>}
          {content?.kind === "image" && (
            <img
              src={content.url}
              alt={file.name}
              className="mx-auto max-h-[70vh]"
            />
          )}
          {(content?.kind === "text" || content?.kind === "hex") && (
            <pre className="break-all whitespace-pre-wrap">
              {content.kind === "text" ? content.text : content.dump}
            </pre>
          )}
          {content && content.kind !== "image" && content.truncated && (
            <div className="mt-2 opacity-70">-- FIRST 64 KB SHOWN --</div>
          )}
        </div>
      </div>
    </div>
  );
}

/** Reads the start of the response and drops the rest. */
async function peek(url: string, limit: number): Promise<Uint8Array> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  const reader = response.body?.getReader();
  if (!reader) {
    return new Uint8Array(await response.arrayBuffer()).slice(0, limit);
  }

  const chunks: Uint8Array[] = [];
  let total = 0;
  while (total < limit) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    chunks.push(value);
    total += value.length;
  }
  await reader.cancel().catch(() => undefined);

  const out = new Uint8Array(Math.min(total, limit));
  let offset = 0;
  for (const chunk of chunks) {
    const n = Math.min(chunk.length, out.length - offset);
    out.set(chunk.subarray(0, n), offset);
    offset += n;
    if (offset >= out.length) {
      break;
    }
  }
  return out;
}

function looksLikeText(bytes: Uint8Array, contentType: string) {
  if (
    contentType.startsWith("text/") ||
    /json|xml|javascript|yaml|toml|x-sh/.test(contentType)
  ) {
    return true;
  }
  const sample = bytes.subarray(0, 4096);
  let control = 0;
  for (const byte of sample) {
    if (byte === 0) {
      return false;
    }
    if (byte < 32 && byte !== 9 && byte !== 10 && byte !== 13) {
      control++;
    }
  }
  return sample.length === 0 || control / sample.length < 0.05;
}

function hexdump(bytes: Uint8Array) {
  const lines: string[] = [];
  for (let offset = 0; offset < bytes.length; offset += 16) {
    const row = bytes.subarray(offset, offset + 16);
    const hex = Array.from(row, (byte) => byte.toString(16).padStart(2, "0"))
      .join(" ")
      .padEnd(47);
    const ascii = Array.from(row, (byte) =>
      byte >= 32 && byte < 127 ? String.fromCharCode(byte) : "."
    ).join("");
    lines.push(`${offset.toString(16).padStart(8, "0")}  ${hex}  |${ascii}|`);
  }
  return lines.join("\n");
}
