import { StatCard, Typography } from "@plakar-ui/ui/components";

import {
  STATS_NUMBER_OF_DAYS,
  useRepositoryStats,
} from "~/components/dashboard/use-repository-stats";

function formatEfficiency(efficiency: number) {
  return `${Math.floor(efficiency * 10) / 10}%`;
}

function RecentFooter({ recentTotal }: { recentTotal: number }) {
  const period = `over the last ${STATS_NUMBER_OF_DAYS} days`;

  if (recentTotal === 0) {
    return (
      <Typography size="xs" weight="semibold" className="text-quaternary">
        None {period}
      </Typography>
    );
  }

  return (
    <Typography size="xs" weight="semibold" className="text-success">
      +{recentTotal.toLocaleString("en-US")} {period}
    </Typography>
  );
}

export function DashboardStats() {
  const { snapshots, recentTotal } = useRepositoryStats();
  const efficiency = snapshots?.efficiency ?? null;
  const logicalSize = snapshots?.logical_size ?? null;
  const storageSize = snapshots?.storage_size ?? null;

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard
        title="Restore points"
        value={snapshots ? snapshots.total.toLocaleString("en-US") : "—"}
        label="total"
        footer={
          recentTotal !== undefined && (
            <RecentFooter recentTotal={recentTotal} />
          )
        }
      />
      <StatCard
        title="Logical size"
        value={logicalSize?.Pretty ?? "—"}
        label={logicalSize ? "of data" : undefined}
        footer={
          <Typography size="xs" weight="semibold" className="text-quaternary">
            Sum of all restore points
          </Typography>
        }
      />
      <StatCard
        title="Storage size"
        value={storageSize?.Pretty ?? "—"}
        label={storageSize ? "on disk" : undefined}
        footer={
          <Typography size="xs" weight="semibold" className="text-quaternary">
            After deduplication and compression
          </Typography>
        }
      />
      <StatCard
        title="Efficiency"
        value={efficiency === null ? "—" : formatEfficiency(efficiency)}
        label={efficiency === null ? undefined : "space saved"}
        footer={
          efficiency !== null && (
            <div className="bg-tertiary mt-1.5 h-2 w-full rounded-full">
              <div
                className="bg-success h-2 rounded-full"
                style={{ width: `${Math.min(efficiency, 100)}%` }}
              />
            </div>
          )
        }
      />
    </div>
  );
}
