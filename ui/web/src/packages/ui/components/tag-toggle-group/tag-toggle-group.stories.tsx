import type { Meta, StoryObj } from "@storybook/react-vite";

import { TagToggleGroup } from "./tag-toggle-group";

const meta: Meta<typeof TagToggleGroup> = {
  title: "design-system/Tag Toggle Group",
  tags: ["autodocs"],
  component: TagToggleGroup,
  argTypes: {
    selectionMode: {
      control: { type: "inline-radio" },
      options: ["single", "multiple"],
    },
  },
};

export default meta;

type Story = StoryObj<typeof TagToggleGroup>;

export const Single: Story = {
  args: {
    selectionMode: "single",
    label: "Environment",
    supportText: "Select the target environment for this deployment.",
  },
  render: (args) => (
    <TagToggleGroup {...args}>
      <TagToggleGroup.Button id="production">Production</TagToggleGroup.Button>
      <TagToggleGroup.Button id="staging">Staging</TagToggleGroup.Button>
      <TagToggleGroup.Button id="development">
        Development
      </TagToggleGroup.Button>
      <TagToggleGroup.Button id="qa">QA</TagToggleGroup.Button>
    </TagToggleGroup>
  ),
};

export const Multiple: Story = {
  args: {
    selectionMode: "multiple",
  },
  render: (args) => (
    <TagToggleGroup {...args} aria-label="Select tags">
      <TagToggleGroup.Button id="compute">Compute</TagToggleGroup.Button>
      <TagToggleGroup.Button id="storage">Block Storage</TagToggleGroup.Button>
      <TagToggleGroup.Button id="database">Database</TagToggleGroup.Button>
      <TagToggleGroup.Button id="network">Network</TagToggleGroup.Button>
      <TagToggleGroup.Button id="security">Security</TagToggleGroup.Button>
      <TagToggleGroup.Button id="monitoring">Monitoring</TagToggleGroup.Button>
    </TagToggleGroup>
  ),
};

export const WithIcons: Story = {
  args: {
    selectionMode: "multiple",
  },
  render: (args) => (
    <TagToggleGroup {...args} aria-label="Select features">
      <TagToggleGroup.Button id="daily" leftIcon="calendar">
        Daily
      </TagToggleGroup.Button>
      <TagToggleGroup.Button id="weekly" leftIcon="calendar">
        Weekly
      </TagToggleGroup.Button>
      <TagToggleGroup.Button id="monthly" leftIcon="calendar">
        Monthly
      </TagToggleGroup.Button>
    </TagToggleGroup>
  ),
};

export const WithDisabled: Story = {
  args: {
    selectionMode: "multiple",
  },
  render: (args) => (
    <TagToggleGroup {...args} aria-label="Select regions">
      <TagToggleGroup.Button id="us-east">US East</TagToggleGroup.Button>
      <TagToggleGroup.Button id="us-west">US West</TagToggleGroup.Button>
      <TagToggleGroup.Button id="eu-west" isDisabled>
        EU West
      </TagToggleGroup.Button>
      <TagToggleGroup.Button id="ap-south" isDisabled>
        AP South
      </TagToggleGroup.Button>
      <TagToggleGroup.Button id="ap-east">AP East</TagToggleGroup.Button>
    </TagToggleGroup>
  ),
};

export const Wrapping: Story = {
  args: {
    selectionMode: "multiple",
  },
  render: (args) => (
    <div className="max-w-80">
      <TagToggleGroup {...args} aria-label="Select categories">
        <TagToggleGroup.Button id="compute">Compute</TagToggleGroup.Button>
        <TagToggleGroup.Button id="storage">
          Block Storage
        </TagToggleGroup.Button>
        <TagToggleGroup.Button id="database">Database</TagToggleGroup.Button>
        <TagToggleGroup.Button id="network">Network</TagToggleGroup.Button>
        <TagToggleGroup.Button id="security">Security</TagToggleGroup.Button>
        <TagToggleGroup.Button id="monitoring">
          Monitoring
        </TagToggleGroup.Button>
        <TagToggleGroup.Button id="logging">Logging</TagToggleGroup.Button>
        <TagToggleGroup.Button id="cdn">CDN</TagToggleGroup.Button>
        <TagToggleGroup.Button id="dns">DNS</TagToggleGroup.Button>
      </TagToggleGroup>
    </div>
  ),
};
