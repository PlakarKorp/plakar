import { Size } from "@plakar-ui/common/api/enrich";
import { SnapshotType } from "@plakar-ui/common/api/kloset";
import { Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";

type CompareSummaryProps = {
  before: SnapshotType;
  after: SnapshotType;
};

type Row = {
  label: string;
  before: number;
  after: number;
  format: (value: number) => string;
  // Whether a smaller value is the good direction, as for errors.
  lowerIsBetter?: boolean;
};

/**
 * Sets the totals of two restore points side by side, with the difference.
 */
export function CompareSummary({ before, after }: CompareSummaryProps) {
  const count = (n: number) => n.toLocaleString();
  const rows: Row[] = [
    {
      label: "Size",
      before: before.total.size.Value,
      after: after.total.size.Value,
      format: (n) => Size(n).Pretty,
    },
    {
      label: "Files",
      before: before.total.files,
      after: after.total.files,
      format: count,
    },
    {
      label: "Directories",
      before: before.total.directories,
      after: after.total.directories,
      format: count,
    },
    {
      label: "Errors",
      before: before.total.errors,
      after: after.total.errors,
      format: count,
      lowerIsBetter: true,
    },
  ];

  return (
    <div className="border-secondary overflow-x-auto rounded-xl border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-secondary text-tertiary border-b text-left">
            <th className="px-4 py-2 font-medium" />
            <SideHeader title="Before" snapshot={before} />
            <SideHeader title="After" snapshot={after} />
            <th className="px-4 py-2 text-right font-medium">Difference</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const delta = row.after - row.before;
            const isGood = row.lowerIsBetter ? delta < 0 : delta > 0;
            return (
              <tr
                key={row.label}
                className="border-secondary border-b last:border-0"
              >
                <td className="text-tertiary px-4 py-2">{row.label}</td>
                <td className="px-4 py-2 text-right tabular-nums">
                  {row.format(row.before)}
                </td>
                <td className="px-4 py-2 text-right tabular-nums">
                  {row.format(row.after)}
                </td>
                <td
                  className={clsx(
                    "px-4 py-2 text-right font-medium tabular-nums",
                    delta === 0
                      ? "text-tertiary"
                      : isGood
                        ? "text-success"
                        : "text-error"
                  )}
                >
                  {delta === 0
                    ? "="
                    : `${delta > 0 ? "+" : "−"}${row.format(Math.abs(delta))}`}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function SideHeader({
  title,
  snapshot,
}: {
  title: string;
  snapshot: SnapshotType;
}) {
  return (
    <th className="px-4 py-2 text-right font-medium">
      <Typography as="span" size="sm" weight="medium" className="text-primary">
        {title}
      </Typography>{" "}
      <Typography as="span" size="xs" className="text-tertiary" monospace>
        {snapshot.identifier.toHumanString()}
      </Typography>
      <Typography size="xs" className="text-tertiary font-normal">
        {snapshot.timestamp.relativeToNow()}
      </Typography>
    </th>
  );
}
