import { Skeleton } from "@plakar-ui/ui/components";

export function IntegrationCardSkeleton() {
  return (
    <div className="border-secondary bg-primary flex flex-col gap-4 rounded-xl border p-4">
      {/* Top */}
      <div className="flex gap-2">
        <div className="h-14 w-14 shrink-0">
          <Skeleton
            variant="rectangle"
            width="100%"
            height="100%"
            className="shrink-0"
          />
        </div>

        <div className="flex w-full flex-col justify-around">
          <Skeleton variant="text" width="100%" />
          <Skeleton variant="text" width="100%" />
        </div>
      </div>

      {/* Content */}
      <div className="flex flex-col gap-3">
        <Skeleton variant="rectangle" height="38px" width="100%" />
        <div className="flex flex-col gap-0.5">
          <Skeleton variant="text" size="md" width="100%" />
          <Skeleton variant="text" size="md" width="100%" />
          <Skeleton variant="text" size="md" width="100%" />
        </div>
      </div>

      {/* Actions */}
      <div>
        <Skeleton variant="rectangle" height="38px" width="100%" />
      </div>
    </div>
  );
}
