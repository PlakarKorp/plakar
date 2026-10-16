import {
  Icon,
  Menu,
  NavListItemProps,
  Typography,
} from "@plakar-ui/ui/components";
import clsx from "clsx";
import { PropsWithChildren, useState } from "react";
import {
  Button as RAButton,
  MenuTrigger,
  Popover,
  PopoverProps,
} from "react-aria-components";

import { User } from "./user-avatar";

type UserMenuProps = PropsWithChildren<{
  user: User;
  subNavItems: NavListItemProps[];
  isDemo?: boolean;
  showChevron?: boolean;
  placement?: PopoverProps["placement"];
  onOpenChange?: (isOpen: boolean) => void;
}>;

export function UserMenu({
  user,
  subNavItems,
  isDemo = false,
  showChevron = false,
  placement = "bottom start",
  onOpenChange,
  children,
}: UserMenuProps) {
  const [isOpen, setIsOpen] = useState(false);

  const logout = subNavItems.find((item) => item.label === "Logout");

  if (!user.isAuthenticated || user.isPending) {
    return <span className="flex min-w-0 items-center gap-2">{children}</span>;
  }

  const handleOpenChange = (open: boolean) => {
    setIsOpen(open);
    onOpenChange?.(open);
  };

  return (
    <MenuTrigger isOpen={isOpen} onOpenChange={handleOpenChange}>
      <RAButton
        aria-label="User menu"
        className={clsx(
          "group/user flex min-w-0 cursor-pointer items-center gap-2 rounded-md text-left outline-0",
          "data-focus-visible:outline-focus data-focus-visible:outline-2 data-focus-visible:outline-offset-2",
          showChevron && "grow justify-between"
        )}
      >
        <span className="flex min-w-0 items-center gap-2">{children}</span>
        {showChevron && (
          <Icon
            type="caret-down"
            size="sm"
            className={clsx(
              "text-quaternary group-data-hovered/user:text-tertiary shrink-0 transition-transform duration-150",
              isOpen && "rotate-180"
            )}
          />
        )}
      </RAButton>
      <Popover
        placement={placement}
        offset={4}
        className="w-[var(--trigger-width)] min-w-56"
      >
        <Menu
          header={
            user.email && (
              <div className="border-secondary min-w-0 border-b px-2 pt-1 pb-2">
                <Typography size="xs" className="text-secondary truncate">
                  {user.email}
                </Typography>
              </div>
            )
          }
        >
          {user.href && (
            <Menu.Item
              leftIcon="user"
              label="Account"
              href={user.href}
              className="cursor-pointer"
            />
          )}
          <Menu.Item
            leftIcon="right-to-bracket"
            label="Sign out"
            href={logout?.href}
            isDisabled={isDemo}
            className="cursor-pointer"
          />
        </Menu>
      </Popover>
    </MenuTrigger>
  );
}
