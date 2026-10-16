import { type EnrichedVersion } from "@plakar-ui/common/api/enrich";
import { useCurrentRoute } from "@plakar-ui/common/hooks";
import {
  Layout as CommonLayout,
  SidePanelProvider,
} from "@plakar-ui/common/layouts";
import { useQuery } from "@tanstack/react-query";
import { PropsWithChildren, useMemo, useState } from "react";

import { NAV_ITEMS, SUB_NAV_ITEMS } from "./layout.constants";

import { getAccountMeQueryOptions } from "~/api/account/queries";
import { getApiInfoQueryOptions } from "~/api/info/queries";
import { KlosetDescription } from "~/components/kloset-description/kloset-description";
import { NotificationsDrawer } from "~/components/notifications/drawer";
import { NotificationsIcon } from "~/components/notifications/icon";
import { useApi } from "~/hooks/use-api";
import { useNotificationToast } from "~/hooks/use-notification-toast";

type LayoutProps = PropsWithChildren<{
  repository: string;
  version: EnrichedVersion;
}>;

export function Layout({ repository, version, children }: LayoutProps) {
  const [open, setOpen] = useState(false);

  useApi();
  useNotificationToast(repository);
  const query = useQuery(getAccountMeQueryOptions());
  const { data: { demo_mode } = {} } = useQuery(getApiInfoQueryOptions());

  const currentRoute = useCurrentRoute(
    [...NAV_ITEMS, ...SUB_NAV_ITEMS].map((item) => String(item.href?.to))
  );

  const navItems = useMemo(
    () =>
      NAV_ITEMS.map((item) => ({
        ...item,
        isSelected: currentRoute === item.href?.to,
      })),
    [currentRoute]
  );
  const subNavItems = useMemo(
    () =>
      SUB_NAV_ITEMS.map((item) => ({
        ...item,
        isSelected: currentRoute === item.href?.to,
      })),
    [currentRoute]
  );

  return (
    <SidePanelProvider>
      <CommonLayout
        navItems={navItems}
        subNavItems={subNavItems}
        user={{
          name: query.data?.name,
          avatarUrl: query.data?.avatar_url,
          isPending: query.isPending,
          isAuthenticated: !!query.data,
        }}
        version={version}
        controls={
          <NotificationsIcon
            repository={repository}
            onPress={() => setOpen((prev) => !prev)}
          />
        }
        footer={<KlosetDescription />}
        isDemo={demo_mode}
      >
        {children}
      </CommonLayout>

      <NotificationsDrawer
        open={open}
        setOpen={setOpen}
        repository={repository}
      />
    </SidePanelProvider>
  );
}
