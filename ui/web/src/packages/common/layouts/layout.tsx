import { PropsWithChildren } from "react";

import { Navigation, NavigationProps } from "./navigation";

type LayoutProps = PropsWithChildren<NavigationProps>;

export function Layout({ children, ...navigationProps }: LayoutProps) {
  return (
    <div className="relative flex h-full min-h-0 flex-col overflow-clip lg:flex-row">
      <Navigation {...navigationProps} />
      <main className="relative h-full min-h-0 min-w-0 flex-1">{children}</main>
    </div>
  );
}
