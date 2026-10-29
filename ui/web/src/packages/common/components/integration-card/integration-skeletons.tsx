import { Skeleton } from "@plakar-ui/ui/components";

export function IntegrationTileSkeleton() {
  return (
    <div className="border-secondary bg-primary flex min-w-0 items-center gap-3 rounded-xl border py-2.5 pr-1.5 pl-3">
      <Skeleton
        variant="rectangle"
        width={32}
        height={32}
        className="shrink-0 rounded-lg"
      />
      <span className="flex min-w-0 flex-1 flex-col">
        <Skeleton
          variant="rectangle"
          width="55%"
          height={14}
          className="my-[3px]"
        />
        <Skeleton
          variant="rectangle"
          width="30%"
          height={10}
          className="my-[5px]"
        />
      </span>
    </div>
  );
}

export function IntegrationCardSkeleton() {
  return (
    <div className="border-secondary bg-primary flex flex-col gap-3 rounded-xl border p-4">
      <div className="flex items-center gap-3">
        <Skeleton
          variant="rectangle"
          width={48}
          height={48}
          className="shrink-0 rounded-xl"
        />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <Skeleton
            variant="rectangle"
            width="50%"
            height={16}
            className="my-1"
          />
          <Skeleton
            variant="rectangle"
            width="25%"
            height={10}
            className="my-[3px]"
          />
        </span>
      </div>
      <span className="flex min-h-10 flex-col">
        <Skeleton
          variant="rectangle"
          width="100%"
          height={12}
          className="my-1"
        />
        <Skeleton
          variant="rectangle"
          width="70%"
          height={12}
          className="my-1"
        />
      </span>
      <span className="flex gap-1">
        <Skeleton variant="rectangle" width={56} height={20} />
        <Skeleton variant="rectangle" width={56} height={20} />
      </span>
      <div className="mt-auto grid grid-cols-2 gap-2">
        <Skeleton
          variant="rectangle"
          width="100%"
          height={38}
          className="rounded-lg"
        />
        <Skeleton
          variant="rectangle"
          width="100%"
          height={38}
          className="rounded-lg"
        />
      </div>
    </div>
  );
}
