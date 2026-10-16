import type { Meta, StoryObj } from "@storybook/react-vite";

import { NavList } from "./nav-list";
import { Avatar } from "../avatar/avatar";
import { Select } from "../select/select";
import { Tag } from "../tag/tag";

const meta: Meta = {
  title: "design-system/Nav List",
  tags: ["autodocs"],
};
export default meta;

const TAB_ITEMS = [
  {
    id: "dashboard",
    title: "Dashboard",
    icon: "house",
  },
  {
    id: "snapshots",
    title: "Snapshots",
    icon: "box-archive",
  },
  {
    id: "alerting",
    title: "Alerting",
    icon: "list-check",
  },
  {
    id: "analytics",
    title: "Analytics",
    icon: "chart-pie",
  },
  {
    id: "integrations",
    title: "Integrations",
    icon: "plug",
  },
] as const;

export const Default: StoryObj = {
  render: () => (
    <div className="max-w-xs">
      <NavList>
        {TAB_ITEMS.map((item) => (
          <NavList.Item
            key={item.id}
            icon={item.icon}
            label={item.title}
            isSelected={item.id === "dashboard"}
            isDisabled={item.id === "integrations"}
            onAction={() => alert(`Navigating to ${item.id}`)}
          />
        ))}
      </NavList>
    </div>
  ),
};

export const Buttons: StoryObj = {
  render: () => (
    <div className="max-w-xs">
      <NavList>
        <NavList.Item
          icon="copy"
          label="Some action"
          onAction={() => alert("Action triggered")}
        />
        <NavList.Item
          icon="bell"
          label="Another action"
          onAction={() => alert("Another action triggered")}
        />
      </NavList>
    </div>
  ),
};

export const WithSublevels: StoryObj = {
  render: () => (
    <div className="max-w-xs">
      <NavList aria-label="Main navigation">
        <NavList.Item
          icon="house"
          label="Dashboard"
          isSelected
          onAction={() => alert("Navigating to dashboard")}
        />

        <NavList.Group label="Analytics">
          <NavList.Item
            icon="lightbulb"
            label="Overview"
            onAction={() => alert("Analytics → Overview")}
          />
          <NavList.Item
            icon="list-check"
            label="Reports"
            onAction={() => alert("Analytics → Reports")}
          />
        </NavList.Group>

        <NavList.Group label="BACKUPS">
          <NavList.Item
            icon="shield-halved"
            label="Rules"
            onAction={() => alert("Alerting → Rules")}
          />
          <NavList.Item
            icon="clock"
            label="History"
            onAction={() => alert("Alerting → History")}
          />
          <NavList.Item
            icon="box-archive"
            label="Snapshots"
            onAction={() => alert("Navigating to snapshots")}
          />
          <NavList.Item
            icon="plug"
            label="Integrations"
            isDisabled
            onAction={() => alert("This should not fire (disabled)")}
          />
        </NavList.Group>
      </NavList>
    </div>
  ),
};

export const Loading: StoryObj = {
  render: () => (
    <div className="max-w-xs">
      <NavList aria-label="Main navigation">
        <NavList.Item icon="house" label="Dashboard" isLoading />

        <NavList.Group label="Analytics">
          <NavList.Item icon="lightbulb" label="Overview" isLoading />
          <NavList.Item icon="list-check" label="Reports" isLoading />
        </NavList.Group>
      </NavList>
    </div>
  ),
};

export const WithExpandable: StoryObj = {
  render: () => (
    <div className="max-w-xs">
      <NavList aria-label="Main navigation">
        <NavList.Item
          icon="house"
          label="Dashboard"
          onAction={() => alert("Navigating to dashboard")}
        />

        <NavList.Expandable icon="chart-pie" label="Analytics" isSelected>
          <NavList.Item
            icon="lightbulb"
            label="Overview"
            onAction={() => alert("Analytics → Overview")}
            altText={<Tag size="sm">New</Tag>}
          />
          <NavList.Item
            icon="list-check"
            label="Reports"
            isSelected
            onAction={() => alert("Analytics → Reports")}
          />
        </NavList.Expandable>

        <NavList.Expandable icon="floppy-disk" label="Backups">
          <NavList.Item
            icon="shield-halved"
            label="Rules"
            onAction={() => alert("Alerting → Rules")}
          />
          <NavList.Item
            icon="clock"
            label="History"
            onAction={() => alert("Alerting → History")}
          />
          <NavList.Item
            icon="box-archive"
            label="Snapshots"
            onAction={() => alert("Navigating to snapshots")}
          />
          <NavList.Item
            icon="plug"
            label="Integrations"
            isDisabled
            onAction={() => alert("This should not fire (disabled)")}
          />
        </NavList.Expandable>
      </NavList>
    </div>
  ),
};

const ORGANIZATIONS = ["Acme", "Acme Europe", "Acme France"];

export const WithScope: StoryObj = {
  render: () => (
    <div className="max-w-xs">
      <NavList aria-label="Main navigation">
        <NavList.Scope
          heading={
            <Select
              aria-label="Organization"
              variant="nav"
              defaultValue="Acme"
              SelectValuePrefix={
                <Avatar text="Acme" size="sm" shape="square" variant="group" />
              }
            >
              <Select.ListBox>
                {ORGANIZATIONS.map((name) => (
                  <Select.Item key={name} id={name} name={name} />
                ))}
              </Select.ListBox>
            </Select>
          }
        >
          <NavList.Item icon="house" label="Dashboard" isSelected />
          <NavList.Group label="Infrastructure">
            <NavList.Item icon="cloud" label="Inventories" />
            <NavList.Item icon="server" label="Edges" />
          </NavList.Group>
          <NavList.Group label="Apps">
            <NavList.Item icon="box-archive" label="Snapshots" />
            <NavList.Item icon="plug" label="Integrations" />
          </NavList.Group>
        </NavList.Scope>
      </NavList>
    </div>
  ),
};
