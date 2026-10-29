import { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { IntegrationCard } from "./integration-card";
import { IntegrationCardSkeleton } from "./integration-skeletons";

const meta = {
  title: "components/Integration Card",
  tags: ["autodocs"],
  component: IntegrationCard,
  args: {
    name: "notion",
    displayName: "Notion",
    version: "v1.0.2",
    stage: "stable",
    description: "Back up Notion workspaces, pages and databases.",
    types: { source: true },
    status: "not-installed",
    detailHref: "#",
    onInstall: fn(),
  },
  decorators: [
    (Story) => (
      <div className="max-w-sm">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof IntegrationCard>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Installing: Story = {
  args: {
    pending: true,
  },
};

export const LoginRequired: Story = {
  args: {
    isDisabled: true,
  },
};

export const Unstable: Story = {
  args: {
    name: "mongodb",
    displayName: "MongoDB",
    version: "v1.0.0-beta.2",
    stage: "beta",
    description:
      "Back up and restore MongoDB databases and collections, with point-in-time consistency across replica sets.",
    types: { source: true, destination: true },
  },
};

export const UpdateAvailable: Story = {
  args: {
    name: "ftp",
    displayName: "FTP",
    version: "v1.1.2",
    status: "installed",
    latestVersion: "v1.1.4",
    onUpgrade: fn(),
    onRemove: fn(),
  },
};

export const UpToDate: Story = {
  args: {
    name: "s3",
    displayName: "S3",
    version: "v1.1.10",
    status: "installed",
    onRemove: fn(),
  },
};

export const BuiltIn: Story = {
  args: {
    name: "fs",
    displayName: "Filesystem",
    version: "v1.2.0",
    status: "built-in",
  },
};

export const Loading: Story = {
  render: () => <IntegrationCardSkeleton />,
};
