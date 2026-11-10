import { EnrichedDate } from "../../api/enrich";

/** Groups the rows of a PaginatedTable by calendar day, for `getGroup`. */
export function dayGroup(date: EnrichedDate) {
  return { key: date.format("yyyy-MM-dd"), label: date.toDayLabel() };
}
