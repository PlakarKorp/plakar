import { StatCard, Typography } from "@plakar-ui/ui/components";

import {
  STATS_NUMBER_OF_DAYS,
  useRepositoryStats,
} from "~/components/dashboard/use-repository-stats";

export function DashboardStats() {
  const { snapshots, recentTotal } = useRepositoryStats();
  const efficiency = snapshots?.efficiency ?? null;

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard
        title="Restore points"
        value={snapshots?.total ?? "—"}
        label="total"
        footer={
          recentTotal !== undefined && (
            <Typography size="xs" weight="semibold" className="text-success">
              +{recentTotal} over the last {STATS_NUMBER_OF_DAYS} days
            </Typography>
          )
        }
      />
      <StatCard
        title="Logical size"
        value={snapshots?.logical_size?.Pretty ?? "—"}
        label="of data"
        footer={
          <Typography size="xs" weight="semibold" className="text-quaternary">
            Sum of all restore points
          </Typography>
        }
      />
      <StatCard
        title="Storage size"
        value={snapshots?.storage_size?.Pretty ?? "—"}
        label="on disk"
        footer={
          <Typography size="xs" weight="semibold" className="text-quaternary">
            After deduplication and compression
          </Typography>
        }
      />
      <StatCard
        title="Efficiency"
        value={efficiency === null ? "—" : `${Math.round(efficiency)}%`}
        label="space saved"
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
