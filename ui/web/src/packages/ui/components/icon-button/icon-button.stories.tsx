import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";
import { fn } from "storybook/test";

import { IconTypes } from "..";
import { IconButton } from "./icon-button";
import { ButtonSizes } from "../button/button.constants";

type Story = StoryObj<typeof meta>;

function Template(args: Omit<ComponentProps<typeof IconButton>, "icon">) {
  return (
    <div className="flex gap-2">
      {IconTypes.map((type) => (
        <IconButton {...args} icon={type} />
      ))}
    </div>
  );
}

const meta = {
  title: "design-system/IconButton",
  tags: ["autodocs"],
  component: Template,
  argTypes: {
    variant: {
      control: false, // Hide "variant" (since we create one story for each variant)
      table: { disable: true }, // Remove from the args table
    },
    onPress: {
      control: false, // Hide "variant" (since we create one story for each variant)
      table: { disable: true }, // Remove from the args table
    },
    isDisabled: {
      control: { type: "boolean" },
      table: {
        type: { summary: "boolean" },
      },
    },
    size: {
      control: { type: "radio" },
      options: ButtonSizes,
    },
  },
  args: {
    onPress: fn(),
    size: "md",
    tooltip: "This is an icon button",
  },
} satisfies Meta<typeof Template>;

export default meta;

export const Primary: Story = {
  args: {
    variant: "primary",
  },
  render: (args) => {
    return <Template {...args} />;
  },
};

export const Secondary: Story = {
  args: {
    variant: "secondary",
  },
  render: (args) => {
    return <Template {...args} />;
  },
};

export const Tertiary: Story = {
  args: {
    variant: "tertiary",
  },
  render: (args) => {
    return <Template {...args} />;
  },
};

export const Error: Story = {
  args: {
    variant: "error",
  },
  render: (args) => {
    return <Template {...args} />;
  },
};

export const AccentTertiary: Story = {
  args: {
    variant: "accent-tertiary",
  },
  render: (args) => {
    return <Template {...args} />;
  },
};

export const ErrorTertiary: Story = {
  args: {
    variant: "error-tertiary",
  },
  render: (args) => {
    return <Template {...args} />;
  },
};

export const WarningQuaternary: Story = {
  args: {
    variant: "warning-quaternary",
  },
  render: (args) => {
    return <Template {...args} />;
  },
};

export const ErrorQuaternary: Story = {
  args: {
    variant: "error-quaternary",
  },
  render: (args) => {
    return <Template {...args} />;
  },
};
