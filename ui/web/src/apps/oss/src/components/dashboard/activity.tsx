import { ActivityHeatmap, Card } from "@plakar-ui/ui/components";

import {
  STATS_NUMBER_OF_DAYS,
  useRepositoryStats,
} from "~/components/dashboard/use-repository-stats";

function perDayToActivity(perDay: number[]) {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return perDay.map((count, index) => {
    const date = new Date(today);
    date.setDate(today.getDate() - (perDay.length - 1 - index));
    return { date, count };
  });
}

export function CardActivity() {
  const { snapshots, recentTotal } = useRepositoryStats();

  return (
    <Card>
      <Card.Header
        title="Activity"
        description={
          recentTotal !== undefined &&
          `${recentTotal} over the last ${STATS_NUMBER_OF_DAYS} days`
        }
      />
      <Card.Content>
        {snapshots && (
          <ActivityHeatmap
            days={perDayToActivity(snapshots.snapshots_per_day)}
          />
        )}
      </Card.Content>
    </Card>
  );
}
