import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { Toaster } from "./toaster";

type TemplateProps = ComponentProps<typeof Toaster> & {
  buttonText?: string;
  onClick?: () => void;
};

const Template = (props: TemplateProps) => {
  const { buttonText, onClick, ...toasterProps } = props;

  if (buttonText) {
    toasterProps.buttonProps = {
      children: buttonText,
      onPress: onClick,
    };
  }

  return <Toaster {...toasterProps} />;
};

const meta = {
  title: "design-system/Toaster",
  component: Template,
  tags: ["autodocs"],
  argTypes: {
    variant: { table: { disable: true }, control: false },
    buttonText: {
      control: "text",
      description: "Action button label",
      defaultValue: "Label",
    },
    onClick: {
      action: "button-clicked",
      table: { disable: true },
    },
  },
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

const baseArgs = {
  title: "This is a message.",
  children: "Lorem ipsum dolor sit amet",
  closable: true,
  buttonText: "Label",
};

export const Default: Story = {
  args: {
    ...baseArgs,
    variant: "default",
  },
};

export const Brand: Story = {
  args: {
    ...baseArgs,
    variant: "brand",
  },
};

export const Success: Story = {
  args: {
    ...baseArgs,
    variant: "success",
  },
};

export const Warning: Story = {
  args: {
    ...baseArgs,
    variant: "warning",
  },
};

export const Error: Story = {
  args: {
    ...baseArgs,
    variant: "error",
  },
};
