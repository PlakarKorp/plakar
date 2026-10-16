import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { Button } from "..";
import { Alert } from "./alert";

type TemplateProps = ComponentProps<typeof Alert> & {
  numberOfActions: number;
};

function Template({ ...props }: TemplateProps) {
  return (
    <Alert
      actions={Array.from({ length: props.numberOfActions }, (_, idx) => (
        <Button key={idx} variant="secondary">
          Action
        </Button>
      ))}
      {...props}
    />
  );
}

const meta = {
  title: "design-system/Alert",
  tags: ["autodocs"],
  component: Template,
  argTypes: {
    variant: {
      control: false, // Hide "variant" (since we create one story for each variant)
      table: { disable: true }, // Remove from the args table
    },
    size: {
      control: { type: "select" },
      options: [undefined, "sm", "md"],
    },
  },
  args: {
    numberOfActions: 2,
  },
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Brand: Story = {
  args: {
    variant: "brand",
    title: "This is a message.",
    children: "Lorem ipsum dolor sit amet",
  },
};

export const Default: Story = {
  args: {
    variant: "default",
    title: "This is a message.",
    children: "Lorem ipsum dolor sit amet",
  },
};

export const Success: Story = {
  args: {
    variant: "success",
    title: "This is a message.",
    children: "Lorem ipsum dolor sit amet",
  },
};

export const Warning: Story = {
  args: {
    variant: "warning",
    title: "This is a message.",
    children: "Lorem ipsum dolor sit amet",
  },
};

export const Error: Story = {
  args: {
    variant: "error",
    title: "This is a message.",
    children: "Lorem ipsum dolor sit amet",
  },
};

export const WithOwnIcon: Story = {
  args: {
    variant: "default",
    icon: "filter",
    title: "Covers what the table is showing",
    children: "Lorem ipsum dolor sit amet",
    numberOfActions: 0,
  },
};
