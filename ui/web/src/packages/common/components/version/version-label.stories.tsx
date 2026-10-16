import type { Meta, StoryObj } from "@storybook/react-vite";

import { VersionLabel } from "./version-label";
import { Version } from "../../api/enrich";

const meta: Meta<typeof VersionLabel> = {
  title: "components/Version Label",
  tags: ["autodocs"],
  component: VersionLabel,
};

export default meta;

type Story = StoryObj<typeof VersionLabel>;

export const DefaultVersion: Story = {
  args: {
    version: Version("v1.2.3"),
  },
};

export const WithCommitHash: Story = {
  args: {
    version: Version("v1.2.3-devel.c88dc607691070e29b39c3be11c330394bbb431f"),
  },
};

export const WithRCAndCommit: Story = {
  args: {
    version: Version("v2.0.0-rc.1.b7836d3ef6e91a97cdac95e4e447e3488f96a5b6"),
  },
};

export const NoVersionPrefix: Story = {
  args: {
    version: Version("1.0.0-beta.abcdefabcdefabcdef"),
  },
};
