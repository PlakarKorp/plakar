import { ReactNode } from "react";

export type TextRange = readonly [start: number, end: number];

type TextHighlightProps = {
  text: string;
  search?: string;
  ranges?: readonly TextRange[];
};

export function TextHighlight({ text, search, ranges }: TextHighlightProps) {
  const spans = ranges ?? searchRanges(text, search);

  if (!text || spans.length === 0) {
    return <>{text}</>;
  }

  const parts: ReactNode[] = [];
  let cursor = 0;

  for (const [start, end] of [...spans].sort((a, b) => a[0] - b[0])) {
    if (start < cursor || end <= start) continue;
    parts.push(text.substring(cursor, start));
    parts.push(<mark key={start}>{text.substring(start, end)}</mark>);
    cursor = end;
  }
  parts.push(text.substring(cursor));

  return <>{parts}</>;
}

function searchRanges(text: string, search?: string): TextRange[] {
  if (!search || !text) return [];
  const start = text.toLowerCase().indexOf(search.toLowerCase());
  return start === -1 ? [] : [[start, start + search.length]];
}
