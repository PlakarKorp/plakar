import { Toolbar } from "react-aria-components";

import { IconButton, Select, Typography } from "..";

const DEFAULT_SIZES = [10, 25, 50, 100];

type PaginationProps = {
  currentPage: number;
  total: {
    count?: number;
    perPage: number;
  };
  onPrev?: () => void;
  onNext?: () => void;
  onPageSizeChange?: (pageSize: number) => void;
};

export function Pagination({
  currentPage,
  total,
  onPrev,
  onNext,
  onPageSizeChange,
}: PaginationProps) {
  const count = total?.count;
  const perPage = total?.perPage;

  // Create options for the per-page select adding the default sizes
  // and the current total perPage if it's not already included.
  const sizeSet = new Set(DEFAULT_SIZES);
  if (perPage && !sizeSet.has(perPage)) {
    sizeSet.add(perPage);
  }
  const options = Array.from(sizeSet)
    .sort((a, b) => a - b)
    .map((n) => ({ id: String(n), name: String(n) }));

  return (
    <Toolbar aria-label="Pagination">
      <div className="h-10" />
      <nav
        aria-label="Pagination"
        className="absolute bottom-0 left-0 z-2 w-full max-lg:px-2 lg:px-4"
      >
        <div className="border-secondary bg-secondary flex h-[55px] w-full items-center justify-between border-t py-2 max-sm:justify-center 2xl:container 2xl:mx-auto">
          <div className="hidden sm:block">
            {!count || !perPage ? (
              <PaginationInfoNoTotal page={currentPage} />
            ) : (
              <PaginationInfo
                page={currentPage}
                count={count}
                rowsPerPage={perPage}
              />
            )}
          </div>
          <div className="flex items-center gap-2">
            {onPageSizeChange && (
              <Typography
                as="span"
                size="sm"
                className="text-tertiary flex items-center gap-1"
              >
                Display:
                <Select
                  value={String(perPage)}
                  placeholder="-"
                  onChange={(key) => onPageSizeChange(Number(key))}
                  isDisabled={!perPage}
                  className="min-w-18"
                >
                  <Select.ListBox>
                    {options.map((option) => (
                      <Select.Item key={option.id} {...option} />
                    ))}
                  </Select.ListBox>
                </Select>
              </Typography>
            )}
            <IconButton
              aria-label="Go to previous page"
              icon="chevron-left"
              variant="secondary"
              isDisabled={onPrev === undefined}
              onPress={onPrev}
            />
            <Typography variant="body" size="sm" className="text-tertiary">
              Page <span className="font-mono">{currentPage + 1}</span> of{" "}
              <span className="font-mono">
                {count && perPage ? Math.ceil(count / perPage) : "-"}
              </span>
            </Typography>
            <IconButton
              aria-label="Go to next page"
              icon="chevron-right"
              variant="secondary"
              isDisabled={onNext === undefined}
              onPress={onNext}
            />
          </div>
        </div>
      </nav>
    </Toolbar>
  );
}

type PaginationInfoProps = {
  count: number;
  page: number;
  rowsPerPage: number;
};

function PaginationInfo({ count, page, rowsPerPage }: PaginationInfoProps) {
  const first = page * rowsPerPage + 1;
  const last = Math.min(count, first + rowsPerPage - 1);

  const text =
    page === 0 && count <= rowsPerPage
      ? `${last} results`
      : `Showing ${first} - ${last} of ${count}`;

  return (
    <Typography variant="body" size="sm" className="text-tertiary">
      {text}
    </Typography>
  );
}

type PaginationInfoNoTotalProps = { page: number };

function PaginationInfoNoTotal({ page }: PaginationInfoNoTotalProps) {
  return (
    <Typography variant="body" size="sm" className="text-tertiary">
      Showing page {page + 1}
    </Typography>
  );
}
