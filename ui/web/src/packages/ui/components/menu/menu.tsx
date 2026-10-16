import clsx from "clsx";
import { ComponentProps, PropsWithChildren, ReactNode } from "react";
import { Menu as RAMenu, MenuItem as RAMenuItem } from "react-aria-components";

import { Icon, IconType, Typography } from "..";
import { MenuSkeleton } from "./menu-skeleton";

type MenuRootProps = PropsWithChildren<
  ComponentProps<typeof RAMenu> & {
    loading?: boolean;
    header?: ReactNode;
    className?: string;
    unstyled?: boolean;
  }
>;

function MenuRoot({
  loading = false,
  header,
  children,
  className,
  unstyled = false,
  ...props
}: MenuRootProps) {
  return (
    <div
      className={clsx(
        "flex max-h-[inherit] flex-col gap-1",
        !unstyled &&
          "border-primary bg-primary rounded-lg border p-1 shadow-lg",
        className
      )}
    >
      {header}
      {loading ? (
        <MenuSkeleton />
      ) : (
        <RAMenu {...props} className="max-h-80 overflow-y-auto outline-none">
          {children}
        </RAMenu>
      )}
    </div>
  );
}

type MenuItemProps = ComponentProps<typeof RAMenuItem> & {
  leftIcon?: IconType;
  label: ReactNode;
  description?: ReactNode;
  altText?: ReactNode;
  rightIcon?: IconType;
  error?: boolean;
  isSelected?: boolean;
};

function MenuItem({
  leftIcon,
  label,
  description,
  altText,
  rightIcon,
  error,
  isSelected = false,
  ...props
}: MenuItemProps) {
  return (
    <RAMenuItem
      {...props}
      className={clsx(
        "flex cursor-default items-center justify-between gap-4 rounded-sm p-2 outline-0",
        isSelected
          ? "bg-accent-heavy text-on-accent"
          : "text-primary data-disabled:text-disable data-focused:bg-primary-hover data-hovered:bg-primary-hover data-disabled:cursor-not-allowed",
        props.className
      )}
    >
      {({ hasSubmenu }) => (
        <>
          <div className="flex min-w-0 items-center gap-2">
            {leftIcon && (
              <Icon
                type={leftIcon}
                size="md"
                className={clsx(error && "text-error")}
              />
            )}

            <div className="flex min-w-0 flex-col">
              <Typography
                as="span"
                size="sm"
                weight="medium"
                className={clsx("truncate", error && "text-error")}
              >
                {label}
              </Typography>
              {description && (
                <Typography
                  as="span"
                  size="xs"
                  className={clsx(
                    "truncate",
                    isSelected ? "text-on-accent" : "text-secondary"
                  )}
                >
                  {description}
                </Typography>
              )}
            </div>
          </div>

          <div className="flex shrink-0 items-center gap-2">
            <Typography as="span" size="xs" className="text-secondary">
              {altText}
            </Typography>

            {rightIcon && <Icon type={rightIcon} size="md" />}
            {hasSubmenu && (
              <Icon
                type="chevron-right"
                size="sm"
                className={clsx("text-tertiary")}
              />
            )}
          </div>
        </>
      )}
    </RAMenuItem>
  );
}

export const Menu = Object.assign(MenuRoot, {
  Item: MenuItem,
});
