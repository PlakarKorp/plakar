import type { Meta, StoryObj } from "@storybook/react-vite";

import { JobStatus } from "./job-status";
import { JOB_STATUSES } from "./job-status.constants";

const meta: Meta<typeof JobStatus> = {
  title: "Components/Job Status",
  tags: ["autodocs"],
  component: JobStatus,
  argTypes: {
    status: {
      control: { type: "select" },
      options: JOB_STATUSES,
    },
  },
};

export default meta;

type Story = StoryObj<typeof JobStatus>;

export const Default: Story = {
  args: {
    status: "succeeded",
  },
};

export const All: Story = {
  render: () => {
    return (
      <div className="grid gap-4">
        {JOB_STATUSES.map((s) => (
          <JobStatus key={s} status={s} />
        ))}
      </div>
    );
  },
};
