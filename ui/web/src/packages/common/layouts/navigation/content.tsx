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
import { cloneElement, ComponentType, useEffect, useState } from "react";
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
import { useScrollEdges } from "../../hooks/use-scroll-edges";

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
  /** Logo of the product, defaults to the Plakar logo. */
  logo?: ComponentType<{ variant: LogoVariant; className?: string }>;
  isOffline?: boolean;
  onQuit?: () => void;
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
  logo: ProductLogo = Logo,
  isOffline = false,
  onQuit,
  handleToggle,
  onCloseMenu,
  onPopoverOpenChange,
}: ContentProps) {
  const logoVariant: LogoVariant = isExpanded ? "full" : "icon";
  const chevronsIcon =
    isExpanded && !isFloating ? "chevrons-left" : "chevrons-right";
  const login = subNavItems.find((item) => item.label === "Login");
  const settings = subNavItems.find((item) => item.label === "Settings");
  const canSignIn = !user.isAuthenticated && !user.isPending;
  const [scrollRef, { isCutTop, isCutBottom }] =
    useScrollEdges<HTMLDivElement>();
  const selectedPath = selectedNavPath(navItems);

  // Keyed on the selected entry, not on every render, so a manual scroll of
  // the list is left alone.
  useEffect(() => {
    scrollRef.current
      ?.querySelector('[aria-current="page"]')
      ?.scrollIntoView({ block: "nearest" });
  }, [scrollRef, selectedPath]);

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
    <div className="flex h-full flex-col gap-4">
      <div className="flex shrink-0 items-center justify-between pb-4 pl-[3.5px]">
        <Link href={navItems[0].href}>
          <ProductLogo variant={logoVariant} className="h-[32px]" />
        </Link>
        {isExpanded && (
          <div
            className={clsx("flex flex-col items-end gap-1", plan && "-mb-2")}
          >
            {plan && (
              <Tag variant="brand" radius="sqrt" size="sm">
                <span className="capitalize">{plan}</span> plan
              </Tag>
            )}
            <VersionLabel version={version} channel />
          </div>
        )}
      </div>

      {/* Only the entries scroll: the logo and the account stay in reach. */}
      <div
        ref={scrollRef}
        className={clsx(
          "scrollbar-hide min-h-0 flex-1 overflow-y-auto overscroll-contain [&_[aria-current=page]]:scroll-my-8",
          isCutTop &&
            isCutBottom &&
            "[mask-image:linear-gradient(to_bottom,transparent,black_2rem,black_calc(100%-2rem),transparent)]",
          isCutTop &&
            !isCutBottom &&
            "[mask-image:linear-gradient(to_bottom,transparent,black_2rem)]",
          !isCutTop &&
            isCutBottom &&
            "[mask-image:linear-gradient(to_top,transparent,black_2rem)]"
        )}
      >
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

      <div className="flex shrink-0 flex-col gap-4">
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
              {onQuit && (
                <NavList.Item
                  icon="power-off"
                  label="Quit"
                  onAction={() => {
                    onCloseMenu?.();
                    onQuit();
                  }}
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

/**
 * Where the selected entry sits in the tree, as its indexes joined. A group is
 * marked selected along with its entry, so only entries count; none while they
 * load, as their placeholders are not marked as the current page.
 */
function selectedNavPath(items: NavNode[]): string | undefined {
  for (const [index, item] of items.entries()) {
    const path = item.subItems?.length
      ? selectedNavPath(item.subItems)
      : item.isSelected && !item.isLoading
        ? ""
        : undefined;
    if (path !== undefined) {
      return path ? `${index}.${path}` : String(index);
    }
  }
  return undefined;
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
