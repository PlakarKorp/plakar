import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { Accordion } from "./accordion";

type StoryProps = ComponentProps<typeof Accordion.Item> &
  ComponentProps<typeof Accordion>;

const meta: Meta<StoryProps> = {
  title: "design-system/Accordion",
  component: Accordion,
  tags: ["autodocs"],
  argTypes: {
    allowsMultipleExpanded: {
      control: "boolean",
      defaultValue: false,
      description: "Whether multiple items can be expanded at the same time",
      table: {
        type: { summary: "boolean" },
        defaultValue: { summary: "false" },
        disabled: true,
      },
    },
    truncateTitle: {
      control: "boolean",
      defaultValue: true,
      description: "Whether to truncate long titles with an ellipsis",
      table: {
        type: { summary: "boolean" },
        defaultValue: { summary: "true" },
        disabled: true,
      },
    },
    defaultExpandedKeys: {
      control: "object",
      defaultValue: [],
      description:
        "Array of item keys (ids) that should be expanded by default",
      table: {
        type: { summary: "string[]" },
        defaultValue: { summary: "[]" },
        disabled: true,
      },
    },
  },
} satisfies Meta<StoryProps>;

export default meta;

type Story = StoryObj<StoryProps>;

const Paragraph = ({ children }: { children: React.ReactNode }) => (
  <p className="text-secondary text-sm">{children}</p>
);

export const Default: Story = {
  args: {
    allowsMultipleExpanded: false,
    defaultExpandedKeys: [],
    truncateTitle: true,
    className: "max-w-md",
  },
  render: ({ truncateTitle, ...args }) => (
    <Accordion {...args}>
      <Accordion.Item
        id="general"
        title="General settings"
        truncateTitle={truncateTitle}
      >
        <Paragraph>
          Configure general behaviors, preferences, and account options.
        </Paragraph>
      </Accordion.Item>

      <Accordion.Item
        id="notifications"
        title="Notifications"
        truncateTitle={truncateTitle}
        defaultExpanded
      >
        <Paragraph>
          Choose which channels to receive updates from (email, Discord,
          webhook).
        </Paragraph>
      </Accordion.Item>

      <Accordion.Item
        id="integrations"
        title="Integrations"
        truncateTitle={truncateTitle}
      >
        <Paragraph>
          Connect external services such as S3, GitHub or custom webhooks.
        </Paragraph>
      </Accordion.Item>
    </Accordion>
  ),
};

export const ExpandedByDefault: Story = {
  args: {
    defaultExpandedKeys: ["overview"],
    className: "max-w-md",
  },
  argTypes: {
    allowsMultipleExpanded: { table: { disable: true } },
    truncateTitle: { table: { disable: true } },
  },
  render: (args) => (
    <Accordion
      defaultExpandedKeys={args.defaultExpandedKeys}
      className={args.className}
    >
      <Accordion.Item id="overview" title="Overview">
        <Paragraph>Overview content…</Paragraph>
      </Accordion.Item>
      <Accordion.Item id="details" title="Details">
        <Paragraph>Details content…</Paragraph>
      </Accordion.Item>
      <Accordion.Item id="advanced" title="Advanced">
        <Paragraph>Additional options…</Paragraph>
      </Accordion.Item>
    </Accordion>
  ),
};

export const MultipleExpandedAllowed: Story = {
  args: {
    allowsMultipleExpanded: true,
    defaultExpandedKeys: ["overview", "details"],
    className: "max-w-md",
  },
  argTypes: {
    truncateTitle: { table: { disable: true } },
  },
  render: (args) => (
    <Accordion
      allowsMultipleExpanded={args.allowsMultipleExpanded}
      defaultExpandedKeys={args.defaultExpandedKeys}
      className={args.className}
    >
      <Accordion.Item id="overview" title="Overview">
        <Paragraph>Overview content…</Paragraph>
      </Accordion.Item>
      <Accordion.Item id="details" title="Details">
        <Paragraph>Details content…</Paragraph>
      </Accordion.Item>
      <Accordion.Item id="advanced" title="Advanced">
        <Paragraph>Additional options…</Paragraph>
      </Accordion.Item>
    </Accordion>
  ),
};

export const LongTitlesTruncated: Story = {
  args: {
    truncateTitle: true,
    className: "max-w-xs",
  },
  argTypes: {
    allowsMultipleExpanded: { table: { disable: true } },
    defaultExpandedKeys: { table: { disable: true } },
  },
  render: (args) => (
    <Accordion className={args.className}>
      <Accordion.Item
        title="A very long section title that will likely overflow the container width and should be truncated with an ellipsis"
        truncateTitle={args.truncateTitle}
      >
        <Paragraph>
          The title above should ellipsize when there’s not enough room.
        </Paragraph>
      </Accordion.Item>
      <Accordion.Item
        title="Another extremely verbose and descriptive title to stress the truncation behavior in narrow layouts"
        truncateTitle={args.truncateTitle}
      >
        <Paragraph>Same behavior here, in a narrow container.</Paragraph>
      </Accordion.Item>
    </Accordion>
  ),
};

export const ItemWithoutContent: Story = {
  args: {
    className: "max-w-md",
  },
  argTypes: {
    allowsMultipleExpanded: { table: { disable: true } },
    defaultExpandedKeys: { table: { disable: true } },
    truncateTitle: { table: { disable: true } },
  },
  render: (args) => (
    <Accordion className={args.className}>
      <Accordion.Item title="This item has content">
        <Paragraph>So the trigger is enabled.</Paragraph>
      </Accordion.Item>
      <Accordion.Item title="This item has no content (disabled trigger)" />
    </Accordion>
  ),
};

export const ItemWithoutGroup: Story = {
  args: {
    defaultExpanded: true,
    className: "max-w-md",
  },
  argTypes: {
    defaultExpandedKeys: { table: { disable: true } },
    truncateTitle: { table: { disable: true } },
    allowsMultipleExpanded: { table: { disable: true } },
  },
  render: (args) => (
    <Accordion.Item
      title="Single item without a group"
      defaultExpanded={args.defaultExpanded}
      className={args.className}
    >
      <Paragraph>So the trigger is enabled.</Paragraph>
    </Accordion.Item>
  ),
};
