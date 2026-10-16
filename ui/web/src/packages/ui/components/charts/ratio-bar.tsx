import clsx from "clsx";

type RatioBarProps = {
  label: string;
  completed: number;
  failed: number;
};

export function RatioBar({ label, completed, failed }: RatioBarProps) {
  const clampedRatio = clampRatio((completed / (completed + failed)) * 100);

  return (
    <div className="space-y-0.5">
      <div className="text-primary flex justify-between gap-2">
        <span className="text-secondary text-sm font-medium">{label}</span>

        <div className="flex gap-3 text-sm">
          <span>
            <span
              className={clsx(
                "font-medium",
                completed > 0 ? "text-primary" : "text-tertiary"
              )}
            >
              {completed}
            </span>
            <span className="text-tertiary"> completed</span>
          </span>

          <span className={clsx(failed > 0 ? "text-error" : "text-tertiary")}>
            <span className="font-medium">{failed}</span>
            <span> failed</span>
          </span>
        </div>
      </div>

      <div
        className={clsx(
          "flex h-2 gap-1.5 rounded-full",
          !completed && !failed && "bg-success-tertiary"
        )}
      >
        {clampedRatio > 0 && (
          <div
            className="bg-success h-2 rounded-full"
            style={{
              width: `${clampedRatio}%`,
            }}
          />
        )}
        {clampedRatio < 100 && (
          <div
            className="bg-error h-2 w-full overflow-hidden rounded-full"
            style={{
              width: `${100 - clampedRatio}%`,
            }}
          />
        )}
      </div>
    </div>
  );
}

const clampRatio = (ratio: number): number => {
  if (!isFinite(ratio) || isNaN(ratio)) {
    return ratio;
  }
  return Math.min(100, Math.max(0, ratio));
};
