import { z } from "zod";

export const CreatedRanges = ["24h", "7d", "30d", "custom"] as const;
type CreatedRange = (typeof CreatedRanges)[number];

const HOUR_MS = 60 * 60 * 1000;

const PRESET_MS: Record<Exclude<CreatedRange, "custom">, number> = {
  "24h": 24 * HOUR_MS,
  "7d": 7 * 24 * HOUR_MS,
  "30d": 30 * 24 * HOUR_MS,
};

/**
 * Search parameters of a creation date filter, to extend a route's search
 * schema with.
 */
export const createdRangeSearchSchema = z.object({
  created: z.enum(CreatedRanges).optional().catch(undefined),
  since: z.iso.datetime({ offset: true }).optional().catch(undefined),
  before: z.iso.datetime({ offset: true }).optional().catch(undefined),
});

export type CreatedRangeSearch = z.infer<typeof createdRangeSearchSchema>;

/**
 * Resolves the creation bounds a search asks for. A preset is resolved now,
 * to the minute, so the query key holds still between renders.
 */
export function createdBounds(search: CreatedRangeSearch) {
  if (!search.created || search.created === "custom") {
    return { since: search.since, before: search.before };
  }
  const minute = 60 * 1000;
  const since = Date.now() - PRESET_MS[search.created];
  return {
    since: new Date(Math.floor(since / minute) * minute).toISOString(),
    before: undefined,
  };
}
