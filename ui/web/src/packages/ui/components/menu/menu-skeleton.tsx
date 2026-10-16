import { Skeleton } from "..";

export function MenuSkeleton() {
  return (
    <div>
      {Array.from({ length: 5 }).map((_, idx) => (
        <div key={idx} className="flex w-full items-center gap-2 p-2">
          <Skeleton variant="icon" size="md" />
          <Skeleton variant="text" size="sm" width="100%" />
          <Skeleton variant="icon" size="md" />
        </div>
      ))}
    </div>
  );
}
