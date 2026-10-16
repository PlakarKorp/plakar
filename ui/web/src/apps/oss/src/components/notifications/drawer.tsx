import { Drawer } from "@plakar-ui/ui/components";

import { NotificationsList } from "./notifications";

type NotificationsDrawerProps = {
  repository: string;
  open: boolean;
  setOpen: (open: boolean) => void;
};

export function NotificationsDrawer({
  repository,
  open,
  setOpen,
}: NotificationsDrawerProps) {
  return (
    <Drawer open={open} setOpen={setOpen} title="Notifications">
      <NotificationsList repository={repository} />
    </Drawer>
  );
}
