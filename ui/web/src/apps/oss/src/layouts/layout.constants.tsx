import { NavListItemProps } from "@plakar-ui/ui/components";

import { UpgradableIntegrations } from "~/components/integration-card/upgradable-integrations";

export const NAV_ITEMS: NavListItemProps[] = [
  {
    label: "Dashboard",
    href: {
      to: "/",
    },
    icon: "house",
  },
  {
    label: "Restore points",
    href: {
      to: "/snapshots",
    },
    icon: "folder-tree",
  },
  {
    label: "Compare",
    href: {
      to: "/compare",
    },
    icon: "not-equal",
  },
  {
    label: "Alerting",
    href: {
      to: "/alerting",
    },
    icon: "list-check",
  },
  {
    label: "Analytics",
    href: {
      to: "/analytics",
    },
    icon: "chart-pie",
  },
  {
    label: "Integrations",
    iconNotification: <UpgradableIntegrations />,
    href: {
      to: "/integrations",
    },
    icon: "plug",
  },
];

export const SUB_NAV_ITEMS: NavListItemProps[] = [
  {
    label: "Login",
    href: {
      to: "/login",
    },
  },
  {
    label: "Logout",
    href: {
      to: "/logout",
    },
  },
  {
    label: "Settings",
    href: {
      to: "/settings",
    },
  },
];
