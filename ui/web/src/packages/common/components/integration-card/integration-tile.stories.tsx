import { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { IntegrationTileSkeleton } from "./integration-skeletons";
import { IntegrationTile } from "./integration-tile";

const meta = {
  title: "components/Integration Tile",
  tags: ["autodocs"],
  component: IntegrationTile,
  args: {
    name: "s3",
    displayName: "Amazon S3",
    version: "v1.1.2",
    stage: "stable",
    detailHref: "#",
    onRemove: fn(),
  },
  decorators: [
    (Story) => (
      <div className="max-w-xs">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof IntegrationTile>;

export default meta;
type Story = StoryObj<typeof meta>;

export const UpToDate: Story = {};

export const UpdateAvailable: Story = {
  args: {
    latestVersion: "v1.1.10",
    onUpgrade: fn(),
  },
};

export const Updating: Story = {
  args: {
    latestVersion: "v1.1.10",
    onUpgrade: fn(),
    pending: true,
  },
};

export const WithQuickActions: Story = {
  args: {
    quickActions: [
      { label: "Add inventory", href: "#" },
      { label: "Add secret provider", href: "#" },
    ],
  },
};

export const Unstable: Story = {
  args: {
    name: "oci",
    displayName: "OCI registry",
    version: "v1.1.0-beta.4",
    stage: "beta",
  },
};

export const BuiltIn: Story = {
  args: {
    name: "fs",
    displayName: "Filesystem",
    version: "v1.2.0",
    isBuiltIn: true,
    onRemove: undefined,
  },
};

export const Loading: Story = {
  render: () => <IntegrationTileSkeleton />,
};
