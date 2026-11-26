import { TextRange } from "@plakar-ui/ui/components";

export type Searchable = {
  id: string;
  title: string;
  subtitle?: string;
  keywords?: readonly string[];
};

export type RankedSearchable<T extends Searchable> = {
  item: T;
  score: number;
  titleRanges: TextRange[];
};

type Folded = { text: string; offsets: number[] };

type TokenMatch = { score: number; ranges: TextRange[] };

const SCORE = {
  exact: 100,
  prefix: 85,
  wordPrefix: 70,
  acronym: 60,
  substring: 45,
  fuzzy: 25,
} as const;

const FIELD_WEIGHT = { title: 1, keyword: 0.8, subtitle: 0.5 } as const;

const RECENT_BOOST = 30;

export function foldSearchText(text: string): Folded {
  let folded = "";
  const offsets: number[] = [];

  for (let index = 0; index < text.length; index++) {
    const simple = text[index]
      .normalize("NFD")
      .replace(/\p{M}/gu, "")
      .toLowerCase();
    for (const char of simple) {
      folded += char;
      offsets.push(index);
    }
  }

  return { text: folded, offsets };
}

export function tokenizeSearch(query: string): string[] {
  return foldSearchText(query).text.split(/\s+/).filter(Boolean);
}

function isWordStart(original: string, index: number) {
  if (index === 0) return true;
  const previous = original[index - 1];
  const current = original[index];
  if (/[\s\-_./:@·>›]/.test(previous)) return true;
  return /[a-z0-9]/.test(previous) && /[A-Z]/.test(current);
}

function wordStarts(original: string, folded: Folded) {
  const starts: number[] = [];
  folded.offsets.forEach((offset, index) => {
    if (folded.offsets[index - 1] !== offset && isWordStart(original, offset)) {
      starts.push(index);
    }
  });
  return starts;
}

function toOriginal(folded: Folded, ranges: TextRange[]): TextRange[] {
  return ranges.map(([start, end]) => [
    folded.offsets[start],
    folded.offsets[end - 1] + 1,
  ]);
}

export function matchSearchToken(token: string, original: string): TokenMatch | null {
  if (!token || !original) return null;

  const folded = foldSearchText(original);
  const text = folded.text;
  const marked = (ranges: TextRange[], score: number) => ({
    score,
    ranges: toOriginal(folded, ranges),
  });

  if (text === token) return marked([[0, text.length]], SCORE.exact);
  if (text.startsWith(token)) return marked([[0, token.length]], SCORE.prefix);

  const starts = wordStarts(original, folded);
  const wordStart = starts.find((start) => text.startsWith(token, start));
  if (wordStart !== undefined) {
    return marked([[wordStart, wordStart + token.length]], SCORE.wordPrefix);
  }

  if (token.length >= 2 && token.length <= starts.length) {
    const initials = starts.map((start) => text[start]).join("");
    const at = initials.indexOf(token);
    if (at !== -1) {
      return marked(
        starts.slice(at, at + token.length).map((start) => [start, start + 1]),
        SCORE.acronym
      );
    }
  }

  const substring = text.indexOf(token);
  if (substring !== -1) {
    return marked([[substring, substring + token.length]], SCORE.substring);
  }

  return token.length >= 3 ? fuzzy(token, folded, starts) : null;
}

function fuzzy(token: string, folded: Folded, starts: number[]) {
  const text = folded.text;
  const positions: number[] = [];
  let from = 0;

  for (const char of token) {
    const found = text.indexOf(char, from);
    if (found === -1) return null;
    positions.push(found);
    from = found + 1;
  }

  const span = positions[positions.length - 1] - positions[0] + 1;
  if (span > token.length * 2 + 2) return null;

  const onStarts = positions.filter((p) => starts.includes(p)).length;
  const gaps = span - token.length;
  const score = SCORE.fuzzy + onStarts * 3 - gaps * 2;

  const ranges: TextRange[] = [];
  for (const position of positions) {
    const last = ranges[ranges.length - 1];
    if (last && last[1] === position) {
      ranges[ranges.length - 1] = [last[0], position + 1];
    } else {
      ranges.push([position, position + 1]);
    }
  }

  return { score: Math.max(score, 1), ranges: toOriginal(folded, ranges) };
}

export function scoreSearchable<T extends Searchable>(
  item: T,
  tokens: string[]
): RankedSearchable<T> | null {
  if (tokens.length === 0) return { item, score: 0, titleRanges: [] };

  let total = 0;
  const titleRanges: TextRange[] = [];

  for (const token of tokens) {
    const title = matchSearchToken(token, item.title);
    const keyword = Math.max(
      0,
      ...(item.keywords ?? []).map((k) => matchSearchToken(token, k)?.score ?? 0)
    );
    const subtitle = item.subtitle
      ? (matchSearchToken(token, item.subtitle)?.score ?? 0)
      : 0;

    const best = Math.max(
      (title?.score ?? 0) * FIELD_WEIGHT.title,
      keyword * FIELD_WEIGHT.keyword,
      subtitle * FIELD_WEIGHT.subtitle
    );
    if (best === 0) return null;

    if (title) titleRanges.push(...title.ranges);
    total += best;
  }

  const brevity = 1 - Math.min(item.title.length, 80) / 400;
  return { item, score: (total / tokens.length) * brevity, titleRanges };
}

type RankOptions = {
  recentIds?: readonly string[];
  limit?: number;
};

export function rankSearchables<T extends Searchable>(
  items: readonly T[],
  query: string,
  { recentIds = [], limit }: RankOptions = {}
): RankedSearchable<T>[] {
  const tokens = tokenizeSearch(query);
  const recency = (id: string) => {
    const at = recentIds.indexOf(id);
    return at === -1 ? 0 : RECENT_BOOST * (1 - at / recentIds.length);
  };

  const ranked = items
    .map((item, order) => {
      const scored = scoreSearchable(item, tokens);
      return (
        scored && { ...scored, score: scored.score + recency(item.id), order }
      );
    })
    .filter((hit) => hit !== null)
    .sort(
      (a, b) =>
        b.score - a.score ||
        (tokens.length > 0 ? a.item.title.length - b.item.title.length : 0) ||
        a.order - b.order
    )
    .map(({ order: _order, ...hit }) => hit);

  return limit === undefined ? ranked : ranked.slice(0, limit);
}

export type ParsedSearchQuery<S extends string> = { scope?: S; text: string };

export function parseSearchQuery<S extends string>(
  raw: string,
  aliases: Readonly<Record<string, S>>,
  actionScope: S
): ParsedSearchQuery<S> {
  if (raw.startsWith(">")) {
    return { scope: actionScope, text: raw.slice(1).trimStart() };
  }

  const match = /^([a-z-]+):\s*(.*)$/i.exec(raw);
  const scope = match && aliases[match[1].toLowerCase()];
  return scope ? { scope, text: match[2] } : { text: raw };
}
