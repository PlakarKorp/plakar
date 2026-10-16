import { Main } from "@plakar-ui/common/layouts";
import PreviewErrorDark from "@plakar-ui/ui/assets/dark/alerting/preview_error.svg";
import PreviewListDark from "@plakar-ui/ui/assets/dark/alerting/preview_list.svg";
import PreviewNotificationDark from "@plakar-ui/ui/assets/dark/alerting/preview_notification.svg";
import PreviewError from "@plakar-ui/ui/assets/light/alerting/preview_error.svg";
import PreviewList from "@plakar-ui/ui/assets/light/alerting/preview_list.svg";
import PreviewNotification from "@plakar-ui/ui/assets/light/alerting/preview_notification.svg";
import { Button, Typography } from "@plakar-ui/ui/components";

const CARD_ITEMS = [
  {
    title: "Monitor Your Kloset Store",
    description:
      "We only collect non-sensitive metadata. Your backup contents are never sent to us.",
    src: PreviewList,
    darkSrc: PreviewListDark,
  },
  {
    title: "Alerts",
    description: "Get error notifications so you can keep your backups secure.",
    src: PreviewError,
    darkSrc: PreviewErrorDark,
  },
  {
    title: "Weekly Summaries",
    description: "Receive a concise activity summary in your inbox.",
    src: PreviewNotification,
    darkSrc: PreviewNotificationDark,
  },
] as const;

export function AlertingLogin() {
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
            Alerting
          </Typography>
          <Typography size="md" className="text-secondary">
            Log in to plakar.io to access the alerting features. Get reports and
            notifications about your Kloset store activity. For free.
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
          href={{ to: "/login", search: { returnUrl: "/alerting" } }}
          className="w-full max-w-sm"
        >
          Sign in
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
    <li className="border-primary bg-primary flex w-70 max-w-sm flex-col items-center gap-1 rounded-2xl border p-2">
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
