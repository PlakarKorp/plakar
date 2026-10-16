import {
  Icon,
  IconType,
  Illustration,
  IllustrationType,
  Typography,
} from "@plakar-ui/ui/components";
import clsx from "clsx";
import { ReactNode } from "react";

type ErrorProps = {
  illustration: IllustrationType;
  title: string;
  icon?: IconType;
  content: ReactNode;
  action?: ReactNode;
  error?: string | Error;
};

export function Error({
  illustration,
  title,
  icon,
  content,
  action,
  error,
}: ErrorProps) {
  return (
    <div
      className="mx-auto flex max-w-6xl flex-col justify-center gap-12 p-4 lg:p-8"
      style={{ minHeight: "calc(100vh - 49px)" }}
    >
      <div
        className={clsx(
          "flex flex-col items-center gap-8 md:flex-row",
          error && "md:my-[20vh]"
        )}
      >
        <Illustration type={illustration} className="w-58 md:w-70" />
        <div className="flex max-w-2xl flex-col gap-8">
          <div className="flex flex-col gap-4">
            <Typography
              as="h1"
              variant="heading"
              size="md"
              className="text-primary flex items-center gap-2"
            >
              {icon && <Icon type={icon} size="lg" />}
              {title}
            </Typography>
            <Typography as="div" size="lg" className="text-secondary">
              {content}
            </Typography>
          </div>
          {action && (
            <div className="flex items-center justify-center gap-2 md:justify-start">
              {action}
            </div>
          )}
        </div>
      </div>
      {error && (
        <div className="border-tertiary bg-primary self-stretch rounded-2xl border p-4 md:p-6">
          <Typography as="div" size="sm" className="text-secondary">
            {typeof error === "string" ? (
              error
            ) : (
              <>
                <strong>
                  {error.name}: {error.message}
                </strong>
                <pre className="mt-2 break-all whitespace-pre-wrap">
                  {error.stack}
                </pre>
              </>
            )}
          </Typography>
        </div>
      )}
    </div>
  );
}
