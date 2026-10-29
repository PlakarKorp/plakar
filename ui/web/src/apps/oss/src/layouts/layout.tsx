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
import { getAssistantStatusQueryOptions } from "~/api/assistant/queries";
import { getApiInfoQueryOptions } from "~/api/info/queries";
import { useQuit } from "~/api/quit/mutations";
import { AssistantDrawer } from "~/components/assistant/assistant-drawer";
import { AssistantIcon } from "~/components/assistant/assistant-icon";
import { KlosetDescription } from "~/components/kloset-description/kloset-description";
import { NotificationsDrawer } from "~/components/notifications/drawer";
import { NotificationsIcon } from "~/components/notifications/icon";
import { QuitDialog } from "~/components/quit/quit-dialog";
import { StoppedScreen } from "~/components/quit/stopped-screen";
import { useApi } from "~/hooks/use-api";
import { useNotificationToast } from "~/hooks/use-notification-toast";

type LayoutProps = PropsWithChildren<{
  repository: string;
  version: EnrichedVersion;
  canQuit?: boolean;
}>;

export function Layout({
  repository,
  version,
  canQuit = false,
  children,
}: LayoutProps) {
  const quit = useQuit();
  const [isQuitDialogOpen, setIsQuitDialogOpen] = useState(false);

  const handleQuitDialogOpenChange = (isOpen: boolean) => {
    setIsQuitDialogOpen(isOpen);
    if (!isOpen) {
      quit.reset();
    }
  };

  // Unmount everything: the queries would poll a dead server.
  if (quit.isSuccess) {
    return <StoppedScreen />;
  }

  return (
    <>
      <Shell
        repository={repository}
        version={version}
        onQuit={canQuit ? () => setIsQuitDialogOpen(true) : undefined}
      >
        {children}
      </Shell>

      <QuitDialog
        isOpen={isQuitDialogOpen}
        onOpenChange={handleQuitDialogOpenChange}
        onConfirm={() => quit.mutate()}
        isPending={quit.isPending}
        error={quit.error?.message}
      />
    </>
  );
}

type ShellProps = PropsWithChildren<{
  repository: string;
  version: EnrichedVersion;
  onQuit?: () => void;
}>;

function Shell({ repository, version, onQuit, children }: ShellProps) {
  const [open, setOpen] = useState(false);
  const [assistantOpen, setAssistantOpen] = useState(false);

  useApi();
  useNotificationToast(repository);
  const query = useQuery(getAccountMeQueryOptions());
  const { data: { demo_mode } = {} } = useQuery(getApiInfoQueryOptions());
  const { data: assistant } = useQuery(getAssistantStatusQueryOptions());

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
          <>
            {assistant?.enabled && (
              <AssistantIcon onPress={() => setAssistantOpen(true)} />
            )}
            <NotificationsIcon
              repository={repository}
              onPress={() => setOpen((prev) => !prev)}
            />
          </>
        }
        footer={<KlosetDescription />}
        isDemo={demo_mode}
        onQuit={onQuit}
      >
        {children}
      </CommonLayout>

      <NotificationsDrawer
        open={open}
        setOpen={setOpen}
        repository={repository}
      />

      {assistant?.enabled && (
        <AssistantDrawer open={assistantOpen} setOpen={setAssistantOpen} />
      )}
    </SidePanelProvider>
  );
}
