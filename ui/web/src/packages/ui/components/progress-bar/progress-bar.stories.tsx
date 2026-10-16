import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button } from "@plakar-ui/ui/components";

import { ProgressBar } from "./progress-bar";

const meta: Meta<typeof ProgressBar> = {
  title: "design-system/Progress Bar",
  component: ProgressBar,
  tags: ["autodocs"],
};

export default meta;

type Story = StoryObj<typeof ProgressBar>;

export const Default: Story = {
  args: {
    isLoading: true,
  },
};

export const SimulateLoading: Story = {
  render: () => {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    const [loading, setLoading] = useState(false);

    return (
      <>
        <Button
          size="sm"
          onPress={() => setLoading((loading) => !loading)}
          className="mb-4"
        >
          {loading ? "Stop" : "Start"} Loading
        </Button>
        <ProgressBar isLoading={loading} />
      </>
    );
  },
};
