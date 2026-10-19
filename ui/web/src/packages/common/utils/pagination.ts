import { z } from "zod";

export const DEFAULT_PER_PAGE = 50;
const MAX_PER_PAGE = 100;

/**
 * Builds the schema validating the `page` and `per_page` URL search params of a
 * paginated route.
 */
export function createPaginationSearchSchema(
  perPage: number = DEFAULT_PER_PAGE
) {
  return z.object({
    page: z.coerce.number().nonnegative().catch(0).default(0),
    per_page: z.coerce
      .number()
      .nonnegative()
      .catch(perPage)
      .default(perPage)
      .transform((n) => Math.min(n, MAX_PER_PAGE)),
  });
}
