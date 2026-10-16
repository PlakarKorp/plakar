import { Button, Logo } from "@plakar-ui/ui/components";
import { useState } from "react";
import { Dialog, Link, Modal, ModalOverlay } from "react-aria-components";

import { Content, ContentProps } from "../content";
import { UserAvatarImage } from "../user-avatar";
import { UserMenu } from "../user-menu";

type HeaderProps = Pick<
  ContentProps,
  | "navItems"
  | "subNavItems"
  | "user"
  | "version"
  | "plan"
  | "isDemo"
  | "controls"
  | "scope"
  | "footer"
  | "isControlPlane"
  | "isOffline"
>;

export function Header(contentProps: HeaderProps) {
  const [open, setOpen] = useState(false);

  const { navItems, subNavItems, user, isDemo } = contentProps;
  const currentRoute = navItems
    .flatMap((item) => item.subItems || item)
    .find((item) => item.isSelected);

  const logo = contentProps.isControlPlane ? "icon-control-plane" : "icon";

  return (
    <>
      <header className="border-primary bg-primary flex items-center justify-between border-b p-2 pl-4 lg:hidden">
        <div className="flex items-center gap-2">
          <Link href={navItems[0].href}>
            <Logo variant={logo} className="h-8.5" />
          </Link>
          <Button
            variant="tertiary"
            leftIcon="bars"
            size="sm"
            onPress={() => setOpen(true)}
          >
            {currentRoute?.label}
          </Button>
        </div>
        <div className="flex items-center gap-4">
          <UserMenu subNavItems={subNavItems} user={user} isDemo={isDemo}>
            <UserAvatarImage user={user} />
          </UserMenu>
        </div>
      </header>
      <ModalOverlay
        isOpen={open}
        onOpenChange={setOpen}
        className="fixed inset-0 z-2 bg-black/50 data-[entering]:animate-[modal-fade_150ms] data-[exiting]:animate-[modal-fade_reverse_150ms] xl:hidden"
        isDismissable
      >
        <Modal className="bg-primary fixed left-0 h-full w-70 overflow-auto shadow-xl data-[entering]:animate-[modal-slide-right_200ms_ease-out] data-[exiting]:animate-[modal-slide-right_150ms_reverse_ease-in]">
          <Dialog
            aria-label="Navigation"
            className="h-full overflow-y-auto p-4"
          >
            <Content
              {...contentProps}
              isExpanded
              isFloating={false}
              canExpand={false}
              onCloseMenu={() => setOpen(false)}
            />
          </Dialog>
        </Modal>
      </ModalOverlay>
    </>
  );
}
