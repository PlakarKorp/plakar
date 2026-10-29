import { Meta, StoryObj } from "@storybook/react-vite";

import { IntegrationDetails } from "./integration-details";

const meta = {
  title: "components/Integration Details",
  tags: ["autodocs"],
  component: IntegrationDetails,
  args: {
    installedVersion: "v1.1.2",
    latestVersion: "v1.1.4",
    stage: "stable",
    types: { source: true, destination: true },
  },
  decorators: [
    (Story) => (
      <div className="max-w-xs">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof IntegrationDetails>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Installed: Story = {};

export const NotInstalled: Story = {
  args: {
    installedVersion: undefined,
  },
};
