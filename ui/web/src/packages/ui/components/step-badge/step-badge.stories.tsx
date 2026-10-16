import { Meta, StoryObj } from "@storybook/react-vite";

import { StepBadge } from "./step-badge";
import { StepBadgeVariant } from "./step-badge.constants";
import { Card } from "../card/card";

const meta = {
  title: "design-system/Step Badge",
  tags: ["autodocs"],
  component: StepBadge,
  args: {
    step: 1,
    variant: "default",
    size: "md",
  },
} satisfies Meta<typeof StepBadge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Variants: Story = {
  render: () => (
    <div className="flex items-center gap-3">
      {StepBadgeVariant.map((variant, index) => (
        <StepBadge key={variant} step={index + 1} variant={variant} />
      ))}
    </div>
  ),
};

export const Sizes: Story = {
  render: () => (
    <div className="flex items-center gap-3">
      <StepBadge step={1} size="sm" />
      <StepBadge step={2} size="md" />
    </div>
  ),
};

export const InCardHeader: Story = {
  render: () => (
    <div className="flex flex-col gap-4">
      <Card>
        <Card.Header
          prefix={<StepBadge step={1} />}
          title="Subject"
          description="Who the permission is granted to."
        />
        <Card.Content padding="lg">Fields go here.</Card.Content>
      </Card>
      <Card>
        <Card.Header
          prefix={<StepBadge step={2} />}
          title="Role"
          description="The set of capabilities to grant."
        />
        <Card.Content padding="lg">Fields go here.</Card.Content>
      </Card>
    </div>
  ),
};
