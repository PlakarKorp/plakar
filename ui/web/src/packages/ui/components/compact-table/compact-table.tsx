import clsx from "clsx";
import { ReactNode } from "react";

import { Typography } from "../typography/typography";

type CompactTableProps = {
  ariaLabel: string;
  headers: string[];
  /** The cells of each row, with a key that identifies the row. */
  rows: { key: string; cells: ReactNode[] }[];
  className?: string;
};

/**
 * A small read-only table, with the first column on the left and the others
 * on the right. It scrolls within its parent when its rows do not fit.
 */
export function CompactTable({
  ariaLabel,
  headers,
  rows,
  className,
}: CompactTableProps) {
  return (
    <div className={clsx("h-full overflow-auto", className)}>
      <table aria-label={ariaLabel} className="w-full">
        <thead>
          <tr>
            {headers.map((header, index) => (
              <th
                key={header}
                className={clsx(
                  "border-primary border-b pb-2 not-first:pl-4",
                  index === 0 ? "text-left" : "text-right"
                )}
              >
                <Typography size="xs" weight="medium" className="text-tertiary">
                  {header}
                </Typography>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.key} className="border-primary border-b last:border-0">
              {row.cells.map((cell, index) => (
                <td
                  key={index}
                  className={clsx(
                    "py-2 not-first:pl-4",
                    index === 0 ? "text-left" : "text-right"
                  )}
                >
                  <Typography as="span" size="sm" className="text-primary">
                    {cell}
                  </Typography>
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
