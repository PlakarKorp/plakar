import type { Meta, StoryObj } from "@storybook/react-vite";

import { Disclosure } from "./disclosure";

const meta: Meta<typeof Disclosure> = {
  title: "design-system/Disclosure",
  component: Disclosure,
  tags: ["autodocs"],
  argTypes: {
    defaultExpanded: {
      control: "boolean",
      description: "Whether the disclosure is expanded by default",
      table: {
        type: { summary: "boolean" },
        defaultValue: { summary: "false" },
      },
    },
    title: {
      control: "text",
      description: "Label shown in the trigger button",
    },
  },
} satisfies Meta<typeof Disclosure>;

export default meta;

type Story = StoryObj<typeof Disclosure>;

export const Default: Story = {
  args: {
    title: "Show details",
    defaultExpanded: false,
  },
  render: (args) => (
    <Disclosure {...args}>
      <p className="text-secondary ml-6 text-sm">
        Here is the content that was hidden. It can be any React node.
      </p>
    </Disclosure>
  ),
};

export const DefaultExpanded: Story = {
  args: {
    title: "Attributes",
    defaultExpanded: true,
  },
  render: (args) => (
    <Disclosure {...args}>
      <ul className="text-secondary ml-6 text-sm">
        <li>
          <span className="font-semibold">region</span>: us-east-1
        </li>
        <li>
          <span className="font-semibold">availability_zone</span>: us-east-1a
        </li>
        <li>
          <span className="font-semibold">instance_type</span>: t3.medium
        </li>
      </ul>
    </Disclosure>
  ),
};
