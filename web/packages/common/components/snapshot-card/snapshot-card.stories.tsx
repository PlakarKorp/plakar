import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { SnapshotCard } from "./snapshot-card";

const meta = {
  title: "components/Snapshot card",
  tags: ["autodocs"],
  component: SnapshotCard,
} satisfies Meta<ComponentProps<typeof SnapshotCard>>;

export default meta;

type Story = StoryObj<ComponentProps<typeof SnapshotCard>>;

export const Default: Story = {
  args: {
    id: "ea42fcfdc3b9852cbfea3452c9deee794aed005fb79f46e698e2724ee12eb1ad",
    sources: [
      {
        type: "SFTP",
        origin: "sftp://example.com",
        directory: "/path/to/snapshot",
        icon: "sftp",
      },
      {
        type: "S3",
        origin: "s3://example.com",
        directory: "/path/to/snapshot",
        icon: "s3",
      },
      {
        type: "FS",
        origin: "nestor-air.lan",
        directory: "/Users/nestor",
        icon: "puzzle-piece",
      },
    ],
    size: "1.2 GB",
    itemsLength: 3,
    tags: ["Production", "Backup"],
    dateModified: "2025/05/15 16:05:16",
    timeSince: "2 days ago",
  },
};

export const TextOverflow: Story = {
  args: {
    ...Default.args,
    sources: [
      {
        type: "Google Cloud Storage",
        origin:
          "sftp://example.com/this/is/a/very/long/path/to/a/snapshot/that/should/cause/text/overflow",
        directory:
          "/path/to/snapshot/this/is/a/very/long/path/to/a/snapshot/that/should/cause/text/overflow",
        icon: "gcs",
      },
      {
        type: "S3",
        origin:
          "s3://example.com/this/is/a/very/long/path/to/a/snapshot/that/should/cause/text/overflow",
        directory:
          "/path/to/snapshot/this/is/a/very/long/path/to/a/snapshot/that/should/cause/text/overflow",
        icon: "s3",
      },
    ],
    tags: [
      "Production",
      "Backup",
      "This is a very long tag",
      "Another long tag",
      "Short",
    ],
  },
};
