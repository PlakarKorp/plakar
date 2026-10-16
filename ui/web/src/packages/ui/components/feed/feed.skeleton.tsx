import { Skeleton } from "..";

export function FeedSkeleton() {
  return (
    <ol className="flex flex-col gap-4">
      {Array.from({ length: 5 }).map((_, index) => (
        <li key={index} className="group relative flex gap-3 py-2">
          <span
            aria-hidden="true"
            className="bg-border-secondary absolute top-10 left-4 -ml-px h-full w-0.5 group-last:hidden"
          />

          <div className="flex flex-1 gap-3">
            <div className="bg-tertiary flex h-8 w-8 shrink-0 items-center justify-center rounded-full">
              <Skeleton variant="circle" width="100%" />
            </div>

            <div className="flex w-full flex-col gap-1">
              <Skeleton
                variant="text"
                size="md"
                width="100%"
                className="leading-none"
              />
              <Skeleton variant="text" size="sm" width="100%" />
            </div>
          </div>

          <div>
            <Skeleton variant="text" size="xs" width="100px" />
          </div>
        </li>
      ))}
    </ol>
  );
}
