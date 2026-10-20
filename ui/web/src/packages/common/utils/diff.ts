export type DiffLine =
  | { op: "equal"; text: string; a: number; b: number }
  | { op: "insert"; text: string; b: number }
  | { op: "delete"; text: string; a: number };

/**
 * Diffs two lists of lines with Myers' algorithm, returning every line with
 * its 1-based line number in the old (a) and new (b) version, or null past
 * `maxEdits` inserted and deleted lines.
 */
export function diffLines(
  a: string[],
  b: string[],
  maxEdits = 2000
): DiffLine[] | null {
  const n = a.length;
  const m = b.length;
  const max = n + m;
  // Diagonals -max-1 to max+1, with diagonal k at index o + k.
  const o = max + 1;
  const v = new Int32Array(2 * max + 3);
  // Step d only reads diagonals -d-1 to d+1 of the step before: keeping just
  // those makes the trace grow with the edits, not with the file sizes.
  const trace: Int32Array[] = [];

  search: for (let d = 0; d <= max; d++) {
    if (d > maxEdits) return null;
    trace.push(v.slice(o - d - 1, o + d + 2));
    for (let k = -d; k <= d; k += 2) {
      let x =
        k === -d || (k !== d && v[o + k - 1] < v[o + k + 1])
          ? v[o + k + 1]
          : v[o + k - 1] + 1;
      let y = x - k;
      while (x < n && y < m && a[x] === b[y]) {
        x++;
        y++;
      }
      v[o + k] = x;
      if (x >= n && y >= m) break search;
    }
  }

  // Walk the trace back from the end to recover the edit script.
  const out: DiffLine[] = [];
  let x = n;
  let y = m;
  for (let d = trace.length - 1; d >= 0; d--) {
    const vd = trace[d];
    const at = (diagonal: number) => vd[diagonal + d + 1];
    const k = x - y;
    const prevK =
      k === -d || (k !== d && at(k - 1) < at(k + 1)) ? k + 1 : k - 1;
    const prevX = at(prevK);
    const prevY = prevX - prevK;
    while (x > prevX && y > prevY) {
      out.push({ op: "equal", text: a[x - 1], a: x, b: y });
      x--;
      y--;
    }
    if (d > 0) {
      if (x === prevX) out.push({ op: "insert", text: b[y - 1], b: y });
      else out.push({ op: "delete", text: a[x - 1], a: x });
    }
    x = prevX;
    y = prevY;
  }
  return out.reverse();
}

export type DiffHunkLine = DiffLine | { op: "skip"; count: number };

/**
 * Keeps the changed lines of a diff with `context` lines around them, folding
 * every other run of unchanged lines into a skip.
 */
export function diffHunks(lines: DiffLine[], context = 3): DiffHunkLine[] {
  const keep = lines.map(() => false);
  lines.forEach((line, i) => {
    if (line.op === "equal") return;
    for (
      let j = Math.max(0, i - context);
      j <= Math.min(lines.length - 1, i + context);
      j++
    ) {
      keep[j] = true;
    }
  });

  const out: DiffHunkLine[] = [];
  let skipped = 0;
  lines.forEach((line, i) => {
    if (keep[i]) {
      if (skipped) out.push({ op: "skip", count: skipped });
      skipped = 0;
      out.push(line);
    } else {
      skipped++;
    }
  });
  if (skipped) out.push({ op: "skip", count: skipped });
  return out;
}
