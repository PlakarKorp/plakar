import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { IconTypes } from "..";
import { Button } from "./button";
import { ButtonSizes, ButtonVariants } from "./button.constants";

const meta = {
  title: "design-system/Button",
  tags: ["autodocs"],
  component: Button,
  argTypes: {
    variant: {
      control: { type: "select" },
      options: [undefined, ...ButtonVariants],
    },
    onPress: {
      control: false,
      table: { disable: true },
    },
    isDisabled: {
      control: { type: "boolean" },
      table: {
        type: { summary: "boolean" }, // Without this, the description shows "unknown" for the type
      },
    },
    leftIcon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
    },
    rightIcon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
    },
    size: {
      control: { type: "radio" },
      options: ButtonSizes,
    },
  },
  args: {
    onPress: fn(),
    isDisabled: false,
    size: "md",
    leftIcon: "bars",
    children: "Label",
    rightIcon: "bars",
    tooltip: "This is a button",
    isPending: false,
  },
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: {
    variant: "primary",
  },
};

export const Secondary: Story = {
  args: {
    variant: "secondary",
  },
};

export const Tertiary: Story = {
  args: {
    variant: "tertiary",
  },
};

export const Error: Story = {
  args: {
    variant: "error",
  },
};

export const ErrorTertiary: Story = {
  args: {
    variant: "error-tertiary",
  },
};

export const AccentTertiary: Story = {
  args: {
    variant: "accent-tertiary",
  },
};

export const AccentQuaternary: Story = {
  args: {
    variant: "accent-quaternary",
  },
};

export const WarningQuaternary: Story = {
  args: {
    variant: "warning-quaternary",
  },
};

export const ErrorQuaternary: Story = {
  args: {
    variant: "error-quaternary",
  },
};

export const SuccessQuaternary: Story = {
  args: {
    variant: "success-quaternary",
  },
};

export const Link: Story = {
  args: {
    variant: "primary",
    children: "I am a link",
    href: "https://demo.plakar.io",
    tooltip: "This is a link styled as a button",
  },
};

export const DisabledLink: Story = {
  args: {
    variant: "primary",
    children: "I am a link nobody may follow",
    href: "https://demo.plakar.io",
    isDisabled: true,
  },
  parameters: {
    docs: {
      description: {
        story:
          "A link honours `isDisabled` the way a button does: the destination is not navigable, and the tooltip is dropped as it is for a disabled button.",
      },
    },
  },
};
