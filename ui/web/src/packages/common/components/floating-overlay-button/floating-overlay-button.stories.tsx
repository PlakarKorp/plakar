import { IconTypes } from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { FloatingOverlayButton } from "./floating-overlay-button";

const meta = {
  title: "Components/Floating overlay button",
  tags: ["autodocs"],
  argTypes: {
    icon: {
      control: { type: "select" },
      options: IconTypes,
      table: {
        type: { summary: "IconType" },
      },
    },
  },
  component: FloatingOverlayButton,
} satisfies Meta<typeof FloatingOverlayButton>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    icon: "info-circle",
    content: <p className="p-4">Welcome to the Floating Overlay Button!</p>,
  },
  render: (args) => (
    <div className="h-30">
      <FloatingOverlayButton {...args} />
    </div>
  ),
};
