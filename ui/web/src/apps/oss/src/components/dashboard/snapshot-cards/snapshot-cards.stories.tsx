import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ToOptions } from "@tanstack/react-router";
import { ComponentProps } from "react";

import { Snapshots } from "./snapshot-cards";

type SnapshotsStoryArgs = ComponentProps<typeof Snapshots> & {
  id: string;
  dateModified?: string;
  timeSince?: string;
  sourceType?: string;
  sourceOrigin?: string;
  sourceDirectory?: string;
  size?: string;
  itemsLength?: number;
  snapshotLink?: ToOptions;
};

const meta = {
  title: "components/Snapshot cards",
  tags: ["autodocs"],
  component: Snapshots,
  argTypes: {
    id: { control: "text" },
    dateModified: { control: "text" },
    timeSince: { control: "text" },
    sourceType: { control: "text" },
    sourceOrigin: { control: "text" },
    sourceDirectory: { control: "text" },
    size: { control: "text" },
    itemsLength: { control: "number" },
  },
} satisfies Meta<SnapshotsStoryArgs>;

export default meta;

type Story = StoryObj<SnapshotsStoryArgs>;

const renderRows = (count: number, args: SnapshotsStoryArgs) => (
  <Snapshots>
    {[...Array(count)].map((_, i) => (
      <Snapshots.Row
        key={i}
        id={args.id}
        dateModified={args.dateModified}
        timeSince={args.timeSince}
        sources={
          args.sourceType || args.sourceOrigin || args.sourceDirectory
            ? [
                {
                  type: args.sourceType,
                  origin: args.sourceOrigin,
                  directory: args.sourceDirectory,
                },
              ]
            : undefined
        }
        size={args.size}
        itemsLength={args.itemsLength}
        snapshotLink={args.snapshotLink}
      />
    ))}
  </Snapshots>
);

export const ManyRows: Story = {
  args: {
    id: "ea42fcfdc3b9852cbfea3452c9deee794aed005fb79f46e698e2724ee12eb1ad",
    dateModified: "2025/05/15 16:05:16",
    timeSince: "4 minutes ago",
    sourceType: "sftp",
    sourceOrigin: "s3://s3.eu-west-3.amazonaws.com/plakar-backups",
    sourceDirectory: "~/data/Korpus/github.com/OpenBSD/www/papers",
    size: "1.51 GB",
    itemsLength: 315,
    snapshotLink: "/",
  },
  render: (args) => renderRows(5, args),
};

export const OneRow: Story = {
  ...ManyRows,
  render: (args) => renderRows(1, args),
};

export const WithRequiredData: Story = {
  args: {
    id: "ea42fcfdc3b9852cbfea3452c9deee794aed005fb79f46e698e2724ee12eb1ad",
  },
  render: (args) => renderRows(4, args),
};
