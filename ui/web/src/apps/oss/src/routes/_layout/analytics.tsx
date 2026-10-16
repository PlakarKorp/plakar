import { Main } from "@plakar-ui/common/layouts";
import PreviewListDark from "@plakar-ui/ui/assets/dark/alerting/preview_list.svg";
import PreviewChartDark from "@plakar-ui/ui/assets/dark/analytics/preview_chart.svg";
import PreviewList from "@plakar-ui/ui/assets/light/alerting/preview_list.svg";
import PreviewChart from "@plakar-ui/ui/assets/light/analytics/preview_chart.svg";
import { Button, Tag, Typography } from "@plakar-ui/ui/components";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_layout/analytics")({
  component: RouteComponent,
});

const CARD_ITEMS = [
  {
    title: "Graphs and charts",
    description: "Visualize your usage at a glance",
    src: PreviewChart,
    darkSrc: PreviewChartDark,
  },
  {
    title: "Dashboards",
    description: "Get insights into your data",
    src: PreviewList,
    darkSrc: PreviewListDark,
  },
] as const;

function RouteComponent() {
  return (
    <Main className="h-full">
      <div className="m-auto flex h-full w-full flex-col items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <Typography
            variant="heading"
            size="xl"
            weight="medium"
            className="text-primary"
          >
            Analytics
          </Typography>

          <Typography as="div" size="md" className="text-secondary">
            This feature is <Tag variant="brand">coming soon</Tag>
          </Typography>
        </div>
        <div className="mt-8 mb-16 w-full overflow-auto text-center">
          <ol
            aria-label="Authenticated features"
            className="inline-flex list-none gap-4"
          >
            {CARD_ITEMS.map((props, index) => (
              <Card key={index} {...props} />
            ))}
          </ol>
        </div>
        <Button
          variant="secondary"
          href={{ to: "/" }}
          className="w-full max-w-sm"
        >
          Return to Dashboard
        </Button>
      </div>
    </Main>
  );
}

function Card({
  title,
  description,
  src,
  darkSrc,
}: {
  title: string;
  description: string;
  src: string;
  darkSrc: string;
}) {
  return (
    <li className="border-secondary bg-primary flex w-full max-w-sm flex-col items-center gap-1 rounded-2xl border p-2">
      <img
        src={src}
        alt={title}
        loading="lazy"
        className="block h-40 w-66 max-w-none dark:hidden"
      />
      <img
        src={darkSrc}
        alt={title}
        loading="lazy"
        className="hidden h-40 w-66 max-w-none dark:block"
      />
      <div className="flex flex-col items-center py-2">
        <Typography size="md" className="text-secondary" weight="medium">
          {title}
        </Typography>
        <Typography size="sm" className="text-tertiary">
          {description}
        </Typography>
      </div>
    </li>
  );
}
