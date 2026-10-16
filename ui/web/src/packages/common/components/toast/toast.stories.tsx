import { Button } from "@plakar-ui/ui/components";
import { Meta, StoryObj } from "@storybook/react-vite";

import { addToast, ToastOptions } from "./queue";
import { Toast } from "./toast";

function Template({ content, sticky, timeout }: ToastOptions) {
  const handleAddToast = () => addToast({ content, sticky, timeout });

  return (
    <>
      <Button onPress={handleAddToast}>Add toast</Button>
      <Toast />
    </>
  );
}

const meta = {
  title: "components/Toast",
  tags: ["autodocs"],
  component: Template,
  argTypes: {
    sticky: {
      description: "Disable auto close",
      table: {
        type: { summary: "boolean" },
        defaultValue: { summary: "false" },
      },
    },
    timeout: {
      description: "Time in milliseconds before auto close",
      table: {
        type: { summary: "number" },
        defaultValue: { summary: "5000" },
      },
    },
  },
} satisfies Meta<typeof Template>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    content: {
      title: "Information",
      children: "This is a default toast message.",
      buttonProps: {
        children: "Action",
        variant: "secondary",
        slot: "close",
        onPress: () => alert("Action triggered"),
      },
    },
    sticky: false,
    timeout: 5000,
  },
};

export const Success: Story = {
  args: {
    content: {
      children: "This is a success toast message.",
      variant: "success",
    },
    sticky: false,
    timeout: 5000,
  },
};

export const Brand: Story = {
  args: {
    content: {
      children: "This is a brand toast message.",
      variant: "brand",
    },
    sticky: false,
    timeout: 5000,
  },
};

export const Warning: Story = {
  args: {
    content: {
      children: "This is a warning toast message.",
      variant: "warning",
    },
    sticky: false,
    timeout: 5000,
  },
};

export const Error: Story = {
  args: {
    content: {
      children: "This is an error toast message.",
      variant: "error",
    },
    sticky: false,
    timeout: 5000,
  },
};
