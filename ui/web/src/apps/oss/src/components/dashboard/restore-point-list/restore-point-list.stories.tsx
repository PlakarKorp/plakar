import { Date as EnrichedDate } from "@plakar-ui/common/api/enrich";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { RestorePointList, RestorePointListItem } from "./restore-point-list";

function hoursAgo(hours: number) {
  return EnrichedDate(new Date(Date.now() - hours * 3_600_000).toISOString());
}

const source = {
  type: "fs",
  origin: "laptop.local",
  directory: "/Users/alice/Documents",
};

const points: RestorePointListItem[] = [
  {
    id: "93bc74b3",
    createdAt: hoursAgo(1),
    sources: [source],
    duration: "<1s",
    size: "545 B",
    items: 3,
    errors: 0,
    link: { to: "/" },
  },
  {
    id: "3a5b15b0",
    createdAt: hoursAgo(26),
    sources: [source, { ...source, type: "s3", directory: "/bucket/photos" }],
    duration: "19s",
    size: "8.09 GB",
    items: 2894,
    errors: 2,
    link: { to: "/" },
  },
  {
    id: "4bf60c9c",
    createdAt: hoursAgo(240),
    sources: [{ ...source, type: "unknown-importer" }],
    duration: "9s",
    size: "7.29 GB",
    items: 2832,
    errors: 0,
    link: { to: "/" },
  },
];

const meta = {
  title: "components/Restore point list",
  tags: ["autodocs"],
  component: RestorePointList,
  args: { points },
} satisfies Meta<typeof RestorePointList>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
