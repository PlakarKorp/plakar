import type { Meta, StoryObj } from "@storybook/react-vite";

import { FileImportButton } from "./file-import-button";

const meta = {
  title: "design-system/FileImportButton",
  tags: ["autodocs"],
  component: FileImportButton,
  args: {
    onSelect: (file) => console.log("Selected:", file.name),
  },
} satisfies Meta<typeof FileImportButton>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Secondary: Story = {
  args: {
    variant: "secondary",
    size: "md",
  },
};

export const WithAcceptedFileTypes: Story = {
  args: {
    acceptedFileTypes: ["application/json"],
    children: "Import JSON",
  },
};
