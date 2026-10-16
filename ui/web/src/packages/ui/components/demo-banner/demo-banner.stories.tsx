import { Meta, StoryObj } from "@storybook/react-vite";

import { DemoBanner } from "./demo-banner";

const meta = {
  title: "design-system/Demo Banner",
  tags: ["autodocs"],
  component: DemoBanner,
} satisfies Meta<typeof DemoBanner>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
