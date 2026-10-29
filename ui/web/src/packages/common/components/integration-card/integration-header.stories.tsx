import { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { IntegrationHeader } from "./integration-header";

const meta = {
  title: "components/Integration Header",
  tags: ["autodocs"],
  component: IntegrationHeader,
  args: {
    name: "ftp",
    displayName: "FTP",
    description:
      "Integration providing import and export capabilities for FTP servers.",
    version: "v1.1.4",
    stage: "stable",
    types: { source: true, destination: true },
    status: "installed",
    onRemove: fn(),
  },
} satisfies Meta<typeof IntegrationHeader>;

export default meta;
type Story = StoryObj<typeof meta>;

export const UpToDate: Story = {};

export const UpdateAvailable: Story = {
  args: {
    version: "v1.1.2",
    latestVersion: "v1.1.4",
    onUpgrade: fn(),
  },
};

export const NotInstalled: Story = {
  args: {
    status: "not-installed",
    onInstall: fn(),
    onRemove: undefined,
  },
};

export const BuiltIn: Story = {
  args: {
    name: "fs",
    displayName: "Filesystem",
    description: "Back up and restore local directories.",
    status: "built-in",
    types: { storage: true, source: true, destination: true },
    onRemove: undefined,
  },
};
