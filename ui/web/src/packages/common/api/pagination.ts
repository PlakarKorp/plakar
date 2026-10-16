export type PaginationParams = {
  page: number;
  perPage: number;
};

export function getPaginationParams(params: PaginationParams) {
  return {
    offset: (params.page * params.perPage).toString(),
    limit: params.perPage.toString(),
  };
}

/**
 * All the paginated endpoints return this response, either with a total count
 * or a boolean indicating if there are more items.
 */
type PaginationResponseWithTotal<T> = {
  items: T[];
  total: number;
};

type PaginationResponseWithHasNext<T> = {
  items: T[];
  has_next: boolean;
};

export type PaginationResponse<T> =
  PaginationResponseWithTotal<T> | PaginationResponseWithHasNext<T>;

export function isPaginationResponseWithTotal<T>(
  response: PaginationResponse<T>
): response is PaginationResponseWithTotal<T> {
  return (response as PaginationResponseWithTotal<T>).total !== undefined;
}
