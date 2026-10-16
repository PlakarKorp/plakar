import {
  RepositoryInfoQueryOptions,
  RepositorySnapshotsQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { AreaChart, Card, Pie, Typography } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";

// Number of days to look back for the subtext "xx over the last N days"
const STATS_NUMBER_OF_DAYS = 30;

type StatsProps = {
  label: string;
  value: string | number;
  footer?: string;
};

function Stats({ label, value, footer }: StatsProps) {
  return (
    <div>
      <Typography variant="body" size="md" className="text-tertiary mb-3">
        {label}
      </Typography>
      <Typography as="p" variant="heading" size="md" className="text-primary">
        {value}
      </Typography>
      {footer && (
        <Typography variant="body" size="sm" className="text-tertiary">
          {footer}
        </Typography>
      )}
    </div>
  );
}

type StatsWithChartProps = StatsProps & {
  chart?: React.ReactNode;
  className?: string;
};

function StatsWithChart({
  label,
  value,
  footer,
  chart,
  className,
}: StatsWithChartProps) {
  return (
    <div className={`flex justify-between gap-2 ${className}`}>
      <Stats label={label} value={value} footer={footer} />
      {chart && (
        <div className="flex w-[120px] items-center justify-center">
          {chart}
        </div>
      )}
    </div>
  );
}

const Divider = () => (
  <div className="bg-border-secondary h-px w-full md:h-full md:w-px" />
);

function getDateNDaysAgo(numberOfDays: number) {
  const d = new Date();
  d.setDate(d.getDate() - numberOfDays);
  d.setHours(0, 0, 0, 0);
  return d;
}

export function CardCharts() {
  const { api, repository } = useKlosetAPI();
  const { data } = useQuery(RepositoryInfoQueryOptions(api, { repository }));
  const snapshotsInfo = data?.item?.snapshots;

  const { data: snapshotsData } = useQuery(
    RepositorySnapshotsQueryOptions(api, {
      repository,
      pagination: {
        page: 0,
        perPage: 1,
      },
      since: getDateNDaysAgo(STATS_NUMBER_OF_DAYS),
    })
  );

  // data.item.snapshots.snapshots_per_day contains a list of numbers
  // representing the number of snapshots taken each day.
  const areaChartData = data?.item.snapshots.snapshots_per_day.map((num) => ({
    value: num,
  }));

  const blocks: React.ReactNode[] = [
    <StatsWithChart
      key="snapshots"
      label="Total restore points"
      value={snapshotsInfo?.total ?? 0}
      footer={
        snapshotsData &&
        `${snapshotsData.total} over the last ${STATS_NUMBER_OF_DAYS} days`
      }
      chart={
        areaChartData && (
          <AreaChart data={areaChartData} width={120} height={88} />
        )
      }
    />,
    snapshotsInfo?.logical_size && (
      <StatsWithChart
        key="size"
        label="Logical size"
        value={snapshotsInfo.logical_size.Pretty}
        footer={
          snapshotsInfo?.storage_size === null
            ? undefined
            : `${snapshotsInfo.storage_size.Pretty} storage size`
        }
        chart={
          snapshotsInfo.efficiency !== null && (
            <Pie value={snapshotsInfo.efficiency} size={88} strokeWidth={8} />
          )
        }
      />
    ),
  ];

  const content = blocks.flatMap((block, index) =>
    index < blocks.length - 1
      ? [block, <Divider key={`divider-${index}`} />]
      : [block]
  );

  const columnTemplate = content
    .map((_, i) => (i % 2 === 0 ? "1fr" : "1px"))
    .join(" ");

  return (
    <Card>
      <Card.Content padding="lg">
        <div
          className={`flex flex-col gap-3 md:grid md:gap-4`}
          style={{ gridTemplateColumns: columnTemplate }}
        >
          {content}
        </div>
      </Card.Content>
    </Card>
  );
}
