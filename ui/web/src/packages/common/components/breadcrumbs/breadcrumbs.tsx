import { Icon } from "@plakar-ui/ui/components";
import { createLink } from "@tanstack/react-router";
import clsx from "clsx";
import { PropsWithChildren } from "react";
import {
  Breadcrumbs as RABreadcrumbs,
  Breadcrumb as RABreadcrumb,
  Link,
  LinkProps,
} from "react-aria-components";

type BreadcrumbsProps = PropsWithChildren;

function BreadcrumbsRoot({ children }: BreadcrumbsProps) {
  return (
    <RABreadcrumbs className="m-0 flex list-none flex-wrap items-center gap-x-2 p-0">
      {children}
    </RABreadcrumbs>
  );
}

type BreadcrumbProps = LinkProps &
  PropsWithChildren<{
    monospace?: boolean;
  }>;

function Breadcrumb({ children, monospace, ...rest }: BreadcrumbProps) {
  return (
    <RABreadcrumb className="text-tertiary group flex items-center gap-2 break-all">
      <Link
        {...rest}
        className={clsx(
          "rounded-sm px-1 py-0.5",
          monospace && "font-mono",
          rest.href && "hover:bg-tertiary-hover"
        )}
      >
        {children}
      </Link>

      <Icon
        type="chevron-right"
        size="sm"
        className="group-last:hidden"
        aria-hidden
      />
    </RABreadcrumb>
  );
}

export const Breadcrumbs = Object.assign(BreadcrumbsRoot, {
  Crumb: createLink(Breadcrumb),
});
