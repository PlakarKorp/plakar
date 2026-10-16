import clsx from "clsx";

type UsageBarProps = {
  used: number;
  limit: number;
  className?: string;
};

/**
 * Consumption against a limit, with what is past the limit in error red. The
 * bar rescales when the limit is exceeded, so the whole overrun stays visible.
 */
export function UsageBar({ used, limit, className }: UsageBarProps) {
  const isOverLimit = used > limit;
  const total = isOverLimit ? used : limit;
  const usedPercent = total > 0 ? (used / total) * 100 : 0;
  const limitPercent = total > 0 ? (limit / total) * 100 : 0;

  return (
    <div
      className={clsx(
        "bg-tertiary relative h-4 w-full overflow-hidden rounded-full",
        className
      )}
      aria-hidden
    >
      {isOverLimit ? (
        <div className="flex h-full w-full gap-px">
          <div
            className="bg-accent h-full"
            style={{ width: `${Math.min(100, limitPercent)}%` }}
          />
          <div
            className="bg-error h-full"
            style={{ width: `${Math.max(0, 100 - limitPercent)}%` }}
          />
        </div>
      ) : (
        <div
          className="bg-accent h-full rounded-full"
          style={{ width: `${Math.min(100, usedPercent)}%` }}
        />
      )}
    </div>
  );
}
