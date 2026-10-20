import { Alert, Icon, Spinner, Typography } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import clsx from "clsx";
import { useMemo } from "react";

import {
  SnapshotFileHeadQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { diffHunks, diffLines } from "@plakar-ui/common/utils";

// Diffs run in the browser: past these, the side-by-side cost stops paying.
const MAX_BYTES = 1024 * 1024;
const MAX_LINES = 5000;
const MAX_CHANGES = 2000;

export type DiffSide = {
  snapshot: string;
  path: string;
  size: number;
  label: string;
};

type TextDiffProps = {
  source: number;
  before: DiffSide;
  after: DiffSide;
};

/**
 * Shows the line diff between two versions of a text file, each read from
 * its restore point, with unchanged runs folded away.
 */
export function TextDiff({ source, before, after }: TextDiffProps) {
  const { api, repository } = useKlosetAPI();
  const tooBig = before.size > MAX_BYTES || after.size > MAX_BYTES;
  const read = (side: DiffSide) =>
    SnapshotFileHeadQueryOptions(api, {
      repository,
      snapshot: side.snapshot,
      source,
      path: side.path,
      bytes: MAX_BYTES,
    });
  const a = useQuery({ ...read(before), enabled: !tooBig });
  const b = useQuery({ ...read(after), enabled: !tooBig });

  const result = useMemo(() => {
    if (a.data == null || b.data == null) return null;
    // A NUL byte in the first kilobytes means the file is not text.
    if ((a.data.slice(0, 8000) + b.data.slice(0, 8000)).includes("\u0000")) {
      return { kind: "binary" as const };
    }
    const linesA = a.data.split("\n");
    const linesB = b.data.split("\n");
    if (linesA.length > MAX_LINES || linesB.length > MAX_LINES) {
      return { kind: "too-long" as const };
    }
    const lines = diffLines(linesA, linesB, MAX_CHANGES);
    if (!lines) {
      return { kind: "too-many-changes" as const };
    }
    return {
      kind: "diff" as const,
      added: lines.filter((l) => l.op === "insert").length,
      removed: lines.filter((l) => l.op === "delete").length,
      hunks: diffHunks(lines),
    };
  }, [a.data, b.data]);

  if (tooBig) {
    return (
      <Alert variant="default" title="Too large to diff here">
        Diffs are computed in the browser and limited to 1 MiB per version.
      </Alert>
    );
  }
  if (a.isError || b.isError) {
    return (
      <Alert variant="error" title="Unable to read the file">
        {(a.error ?? b.error)?.message}
      </Alert>
    );
  }
  if (a.data === null || b.data === null) {
    return (
      <Alert variant="warning" title="Unable to read the file">
        One of the versions is in cold storage.
      </Alert>
    );
  }
  if (!result) {
    return <Spinner />;
  }
  if (result.kind === "binary") {
    return (
      <Alert variant="default" title="Binary content">
        These versions differ, but they are not text files.
      </Alert>
    );
  }
  if (result.kind === "too-long") {
    return (
      <Alert variant="default" title="Too many lines to diff here">
        Diffs are limited to {MAX_LINES} lines per version.
      </Alert>
    );
  }
  if (result.kind === "too-many-changes") {
    return (
      <Alert variant="default" title="Too many changes to diff here">
        Diffs are limited to {MAX_CHANGES} added and removed lines.
      </Alert>
    );
  }
  if (result.added === 0 && result.removed === 0) {
    return (
      <Alert variant="success" title="Identical content">
        The bytes are the same: only metadata, such as the modification time,
        differs.
      </Alert>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-3">
        <Typography size="sm" weight="semibold" className="text-success">
          +{result.added}
        </Typography>
        <Typography size="sm" weight="semibold" className="text-error">
          −{result.removed}
        </Typography>
        <Typography size="sm" className="text-tertiary">
          {before.label} → {after.label}
        </Typography>
      </div>
      <div className="border-secondary bg-secondary overflow-auto rounded-lg border font-mono text-xs leading-relaxed">
        <table className="w-full border-collapse">
          <tbody>
            {result.hunks.map((line, i) =>
              line.op === "skip" ? (
                <tr key={i} className="bg-tertiary text-tertiary">
                  <td colSpan={4} className="px-3 py-1 font-sans">
                    <span className="flex items-center gap-2">
                      <Icon type="ellipsis" size="xs" />
                      {line.count} unchanged line{line.count > 1 ? "s" : ""}
                    </span>
                  </td>
                </tr>
              ) : (
                <tr
                  key={i}
                  className={clsx(
                    line.op === "insert" && "bg-success-tertiary/60",
                    line.op === "delete" && "bg-error-tertiary/60"
                  )}
                >
                  <td className="text-quaternary border-secondary w-12 border-r px-2 text-right select-none">
                    {line.op !== "insert" ? line.a : ""}
                  </td>
                  <td className="text-quaternary border-secondary w-12 border-r px-2 text-right select-none">
                    {line.op !== "delete" ? line.b : ""}
                  </td>
                  <td
                    className={clsx(
                      "w-5 text-center select-none",
                      line.op === "insert" && "text-success",
                      line.op === "delete" && "text-error"
                    )}
                  >
                    {line.op === "insert"
                      ? "+"
                      : line.op === "delete"
                        ? "−"
                        : ""}
                  </td>
                  <td className="text-primary pr-3 whitespace-pre">
                    {line.text}
                  </td>
                </tr>
              )
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
