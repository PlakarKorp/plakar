import clsx from "clsx";
import { ComponentProps, PropsWithChildren, ReactNode } from "react";
import {
  Button,
  Disclosure,
  DisclosurePanel,
  DisclosureProps,
  Link,
  LinkProps,
} from "react-aria-components";

import { Typography, Icon, IconType, Skeleton } from "..";

type NavListRootProps = ComponentProps<"nav">;

function NavListRoot({ children, ...props }: NavListRootProps) {
  return (
    <nav {...props}>
      <ul className="flex flex-col">{children}</ul>
    </nav>
  );
}

export type NavListItemProps = Pick<LinkProps, "href"> & {
  icon?: IconType;
  label: ReactNode;
  isSelected?: boolean;
  isDisabled?: boolean;
  isExpandable?: boolean;
  isLoading?: boolean;
  altText?: ReactNode;
  iconNotification?: ReactNode;
  onAction?: () => void;
  className?: string;
  subItems?: NavListItemProps[];
};

function NavListItem({
  icon,
  label,
  isSelected = false,
  isDisabled = false,
  isExpandable = false,
  isLoading = false,
  altText,
  iconNotification,
  href,
  onAction,
  className,
  ...props
}: NavListItemProps & Omit<ComponentProps<"li">, "children">) {
  if (isLoading) {
    return (
      <li {...props}>
        <div className="flex w-full items-center gap-3 px-3 py-2">
          <Skeleton variant="icon" size="md" />
          <Skeleton variant="text" size="sm" width="60%" />
        </div>
      </li>
    );
  }

  const sharedClasses = clsx(
    "flex w-full items-center gap-4 rounded-sm px-3 py-2 outline-0",
    "hover:bg-accent-tertiary focus-visible:bg-accent-tertiary",
    isSelected
      ? "bg-tertiary text-primary"
      : "text-secondary data-[disabled=true]:text-disable data-[disabled=true]:cursor-not-allowed",
    className
  );

  return (
    <li {...props}>
      {href ? (
        <Link
          href={href}
          aria-current={isSelected ? "page" : undefined}
          onPress={onAction}
          className={sharedClasses}
          isDisabled={isDisabled}
        >
          <Content
            icon={icon}
            label={label}
            altText={altText}
            iconNotification={iconNotification}
          />
        </Link>
      ) : (
        <Button
          type="button"
          slot={isExpandable ? "trigger" : undefined}
          isDisabled={isDisabled}
          onPress={onAction}
          className={sharedClasses}
        >
          <Content
            icon={icon}
            label={label}
            altText={altText}
            iconNotification={iconNotification}
          />
          {isExpandable && (
            <Icon
              type="chevron-right"
              aria-hidden
              size="sm"
              className="transition-transform duration-200 group-data-[expanded]/expandable:rotate-90"
            />
          )}
        </Button>
      )}
    </li>
  );
}

export type NavListGroupProps = PropsWithChildren<{
  label: ReactNode;
  hideTitle?: boolean;
}>;

function NavListGroup({
  label,
  children,
  hideTitle = false,
}: NavListGroupProps) {
  return (
    <li>
      {hideTitle ? (
        <hr
          aria-hidden="true"
          className="after:border-secondary relative my-2 h-4 border-0 after:absolute after:inset-x-0 after:top-1/2 after:-translate-y-1/2 after:border-t after:content-['']"
        />
      ) : (
        <Typography
          as="p"
          size="xs"
          weight="semibold"
          className="text-placeholder mt-3 mb-1 ml-3 uppercase"
        >
          {label}
        </Typography>
      )}
      <ul className="flex flex-col">{children}</ul>
    </li>
  );
}

export type NavListScopeProps = PropsWithChildren<{
  heading: ReactNode;
}>;

function NavListScope({ heading, children }: NavListScopeProps) {
  return (
    <li className="flex flex-col gap-3">
      {heading}
      <ul className="flex flex-col">{children}</ul>
    </li>
  );
}

type ContentProps = Pick<
  NavListItemProps,
  "icon" | "label" | "altText" | "iconNotification"
>;

function Content({ icon, label, altText, iconNotification }: ContentProps) {
  return (
    <div className="flex flex-1 items-center justify-between gap-3">
      <div className="flex items-center gap-3">
        <span className="relative flex items-center justify-center">
          {icon && <Icon type={icon} size="md" />}
          {iconNotification && (
            <div className="absolute -top-1.5 -right-2.5 flex">
              {iconNotification}
            </div>
          )}
        </span>
        <Typography as="span" size="sm" weight="medium" className="truncate">
          {label}
        </Typography>
      </div>

      {altText && (
        <Typography as="span" size="xs" className="text-secondary -mr-1">
          {altText}
        </Typography>
      )}
    </div>
  );
}

type NavListExpandableProps = PropsWithChildren<
  Omit<NavListItemProps, "isExpandable" | "href"> &
    Pick<DisclosureProps, "isExpanded" | "onExpandedChange">
>;

export function NavListExpandable({
  icon,
  label,
  isSelected = false,
  isDisabled = false,
  altText,
  iconNotification,
  onAction,
  className,
  children,
  isExpanded,
  onExpandedChange,
}: NavListExpandableProps) {
  return (
    <Disclosure
      className="group/expandable"
      isExpanded={isExpanded}
      onExpandedChange={onExpandedChange}
    >
      <NavListItem
        icon={icon}
        label={label}
        isSelected={isSelected}
        isDisabled={isDisabled}
        altText={altText}
        iconNotification={iconNotification}
        onAction={onAction}
        className={className}
        isExpandable
      />
      <DisclosurePanel>
        <ul className="border-secondary ml-5 border-l pl-3">{children}</ul>
      </DisclosurePanel>
    </Disclosure>
  );
}

export const NavList = Object.assign(NavListRoot, {
  Item: NavListItem,
  Group: NavListGroup,
  Scope: NavListScope,
  Expandable: NavListExpandable,
});
