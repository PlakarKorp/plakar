import { Size } from "@plakar-ui/common/api/enrich";
import { SnapshotType } from "@plakar-ui/common/api/kloset";
import { Typography } from "@plakar-ui/ui/components";
import {
  Cell,
  Column,
  Row as TableRow,
  Table,
  TableBody,
  TableHeader,
} from "react-aria-components";

import { Delta } from "./delta";

type CompareSummaryProps = {
  before: SnapshotType;
  after: SnapshotType;
};

type Row = {
  label: string;
  before: number;
  after: number;
  format: (value: number) => string;
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
    <div className="border-secondary bg-primary overflow-x-auto rounded-xl border">
      <Table aria-label="Totals" className="w-full border-collapse">
        <TableHeader className="border-secondary border-b">
          <Column isRowHeader className="px-4 py-2 text-left">
            <span className="sr-only">Total</span>
          </Column>
          <Column className="px-4 py-2 text-right">
            <SideHeader title="Before" snapshot={before} />
          </Column>
          <Column className="px-4 py-2 text-right">
            <SideHeader title="After" snapshot={after} />
          </Column>
          <Column className="px-4 py-2 text-right">
            <Typography size="sm" weight="medium" className="text-tertiary">
              Difference
            </Typography>
          </Column>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow
              key={row.label}
              id={row.label}
              className="border-secondary border-b last:border-0"
            >
              <Cell className="px-4 py-2">
                <Typography size="sm" className="text-tertiary">
                  {row.label}
                </Typography>
              </Cell>
              <Cell className="px-4 py-2 text-right">
                <Typography size="sm" className="tabular-nums">
                  {row.format(row.before)}
                </Typography>
              </Cell>
              <Cell className="px-4 py-2 text-right">
                <Typography size="sm" className="tabular-nums">
                  {row.format(row.after)}
                </Typography>
              </Cell>
              <Cell className="px-4 py-2 text-right">
                <Delta
                  before={row.before}
                  after={row.after}
                  format={row.format}
                  lowerIsBetter={row.lowerIsBetter}
                  className="font-medium"
                />
              </Cell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
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
    <>
      <Typography as="span" size="sm" weight="medium" className="text-primary">
        {title}
      </Typography>{" "}
      <Typography as="span" size="xs" className="text-tertiary" monospace>
        {snapshot.identifier.toHumanString()}
      </Typography>
      <Typography size="xs" className="text-tertiary">
        {snapshot.timestamp.relativeToNow()}
      </Typography>
    </>
  );
}
