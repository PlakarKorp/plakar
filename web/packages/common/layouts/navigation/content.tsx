import {
  DemoBanner,
  IconButton,
  LoginBanner,
  Logo,
  LogoVariant,
  NavList,
  NavListItemProps,
  Tag,
} from "@plakar-ui/ui/components";
import clsx from "clsx";
import { cloneElement, useState } from "react";
import { Link } from "react-aria-components";

import { User, UserAvatar } from "./user-avatar";
import { UserMenu } from "./user-menu";
import { type EnrichedVersion } from "../../api/enrich";
import {
  DiscordStats,
  GithubRepo,
  ThemeSwitch,
  VersionLabel,
} from "../../components";

export type ContentProps = {
  navItems: NavListItemProps[];
  subNavItems: NavListItemProps[];
  isExpanded: boolean;
  version: EnrichedVersion;
  plan?: string;
  isFloating: boolean;
  canExpand: boolean;
  isDemo?: boolean;
  user: User;
  controls?: React.ReactNode;
  scope?: React.ReactElement<NavigationScopeProps>;
  footer?: React.ReactElement<{ forceClose?: boolean }>;
  isControlPlane?: boolean;
  isOffline?: boolean;
  onCloseMenu?: () => void;
  handleToggle?: () => void;
  onPopoverOpenChange?: (isOpen: boolean) => void;
};

export type NavigationScopeProps = {
  isCompact?: boolean;
  onOpenChange?: (isOpen: boolean) => void;
};

export function Content({
  navItems,
  subNavItems,
  isExpanded,
  version,
  plan,
  isFloating,
  canExpand,
  isDemo,
  user,
  controls,
  scope,
  footer,
  isControlPlane = false,
  isOffline = false,
  handleToggle,
  onCloseMenu,
  onPopoverOpenChange,
}: ContentProps) {
  const logoVariant: LogoVariant = isExpanded
    ? `full${isControlPlane ? "-control-plane" : ""}`
    : `icon${isControlPlane ? "-control-plane" : ""}`;
  const chevronsIcon =
    isExpanded && !isFloating ? "chevrons-left" : "chevrons-right";
  const login = subNavItems.find((item) => item.label === "Login");
  const settings = subNavItems.find((item) => item.label === "Settings");
  const canSignIn = !user.isAuthenticated && !user.isPending;

  const navNodes = navItems.map((item, i) => (
    <NavNodeRenderer
      key={`${item.label}-${i}`}
      nodeKey={`${item.label}-${i}`}
      node={item as NavNode}
      isSidebarExpanded={isExpanded}
      onAction={onCloseMenu}
    />
  ));

  return (
    <div className="flex min-h-full flex-col justify-between gap-4 overflow-hidden">
      <div className="flex flex-col gap-8">
        <div className={clsx("ml-[3.5px] flex items-center justify-between")}>
          <Link href={navItems[0].href}>
            <Logo variant={logoVariant} className="h-[32px]" />
          </Link>
          {isExpanded && (
            <div
              className={clsx(
                "flex flex-col items-end gap-1",
                isControlPlane && plan && "-mb-2"
              )}
            >
              {isControlPlane && plan && (
                <Tag variant="brand" radius="sqrt" size="sm">
                  <span className="capitalize">{plan}</span> plan
                </Tag>
              )}
              <VersionLabel version={version} channel />
            </div>
          )}
        </div>

        <NavList aria-label="main navigation">
          {scope ? (
            <NavList.Scope
              heading={cloneElement(scope, {
                isCompact: !isExpanded,
                onOpenChange: onPopoverOpenChange,
              })}
            >
              {navNodes}
            </NavList.Scope>
          ) : (
            navNodes
          )}
        </NavList>
      </div>

      <div className="flex flex-col gap-4">
        {isExpanded && !user.isPending && (
          <>
            {isDemo && user.isAuthenticated && <DemoBanner />}
            {!isDemo && !user.isAuthenticated && (
              <LoginBanner href={login?.href} onClick={onCloseMenu} />
            )}
          </>
        )}

        {footer && cloneElement(footer, { forceClose: !isExpanded })}

        <div className="border-secondary flex flex-col gap-4 border-t pt-4">
          <div className="flex flex-col">
            <ThemeSwitch isCompact={!isExpanded} />

            <NavList aria-label="settings navigation">
              {settings && (
                <NavList.Item
                  icon="gear"
                  label="Settings"
                  href={settings?.href}
                  onAction={onCloseMenu}
                  isSelected={settings?.isSelected}
                  isLoading={settings.isLoading}
                  isDisabled={!user.isAuthenticated}
                  altText={settings.altText}
                  iconNotification={settings.iconNotification}
                />
              )}
              {!user.isAuthenticated && (
                <NavList.Item
                  icon="right-to-bracket"
                  label="Sign in"
                  href={login?.href}
                  onAction={onCloseMenu}
                  isSelected={login?.isSelected}
                />
              )}
            </NavList>
          </div>

          <span className="flex min-w-0 items-center justify-between gap-2">
            <UserMenu
              user={user}
              subNavItems={subNavItems}
              isDemo={isDemo}
              showChevron={isExpanded}
              placement="top start"
              onOpenChange={onPopoverOpenChange}
            >
              <UserAvatar user={user} isExpanded={isExpanded} />
            </UserMenu>
            {isExpanded && (canSignIn || controls) && (
              <span className="flex items-center gap-2">
                {canSignIn && (
                  <IconButton
                    icon="right-to-bracket"
                    size="sm"
                    variant="tertiary"
                    tooltip="Sign in"
                    href={login?.href}
                  />
                )}
                {controls}
              </span>
            )}
          </span>

          <div className="flex h-[26px] justify-between gap-2">
            {isExpanded && (
              <div className="flex w-full items-center justify-center gap-2 lg:justify-start">
                {!isOffline && (
                  <>
                    <DiscordStats invite="uuegtnF2Q5" />
                    <GithubRepo repo="PlakarKorp/plakar" />
                  </>
                )}
              </div>
            )}
            {handleToggle && (
              <IconButton
                aria-label="Toggle sidebar"
                icon={chevronsIcon}
                size="sm"
                variant="tertiary"
                onPress={handleToggle}
                tooltip="Toggle sidebar"
                isDisabled={!canExpand}
                className={clsx(!isExpanded && "w-full text-center")}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

type NavNode = NavListItemProps & {
  subItems?: NavNode[];
  isExpandable?: boolean;
};

type NavNodeRendererProps = {
  node: NavNode;
  nodeKey: React.Key;
  isSidebarExpanded: boolean;
  onAction?: () => void;
};

function NavExpandableNode({
  nodeKey,
  node,
  isSidebarExpanded,
  children,
  onAction,
}: {
  nodeKey: React.Key;
  node: Omit<NavNode, "subItems">;
  isSidebarExpanded: boolean;
  children: React.ReactNode;
  onAction?: () => void;
}) {
  const [isItemExpanded, setIsItemExpanded] = useState(node.isSelected);

  const isExpanded = isItemExpanded && isSidebarExpanded;

  return (
    <NavList.Expandable
      key={nodeKey}
      {...node}
      isSelected={node.isSelected && !isExpanded}
      onAction={onAction}
      isExpanded={isExpanded}
      onExpandedChange={setIsItemExpanded}
    >
      {children}
    </NavList.Expandable>
  );
}

function NavNodeRenderer({
  node,
  nodeKey,
  isSidebarExpanded,
  onAction,
}: NavNodeRendererProps) {
  const { subItems, ...rest } = node;

  if (subItems?.length && !rest.isExpandable) {
    return (
      <NavList.Group
        key={nodeKey}
        label={rest.label}
        hideTitle={!isSidebarExpanded}
      >
        {subItems.map((child, i) => (
          <NavNodeRenderer
            key={`${String(nodeKey)}-${i}`}
            nodeKey={`${String(nodeKey)}-${i}`}
            node={child}
            isSidebarExpanded={isSidebarExpanded}
            onAction={onAction}
          />
        ))}
      </NavList.Group>
    );
  }

  if (rest.isExpandable && subItems?.length) {
    return (
      <NavExpandableNode
        nodeKey={nodeKey}
        node={rest}
        isSidebarExpanded={isSidebarExpanded}
      >
        {subItems.map((child, i) => (
          <NavNodeRenderer
            key={`${String(nodeKey)}-${i}`}
            nodeKey={`${String(nodeKey)}-${i}`}
            node={child}
            isSidebarExpanded={isSidebarExpanded}
            onAction={onAction}
          />
        ))}
      </NavExpandableNode>
    );
  }

  return <NavList.Item key={nodeKey} {...rest} onAction={onAction} />;
}
