import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useState } from "react";
import { fn } from "storybook/test";

import { Pagination } from "./pagination";

type TemplateProps = {
  currentPage: number;
  hasPrev: boolean;
  hasNext: boolean;
  hasTotal: boolean;
  count: number;
  perPage: number;
  onPrev: () => void;
  onNext: () => void;
  onPageSizeChange?: (perPage: number) => void;
  pageSizes?: number[];
};

function Template({
  currentPage,
  hasPrev,
  hasNext,
  hasTotal,
  count,
  perPage,
  onPrev,
  onNext,
  pageSizes,
}: TemplateProps) {
  const [pageSize, setPageSize] = useState(perPage);

  useEffect(() => {
    setPageSize(perPage);
  }, [perPage]);

  return (
    <Pagination
      currentPage={currentPage}
      total={{ count: hasTotal ? count : undefined, perPage: pageSize }}
      onPrev={hasPrev ? onPrev : undefined}
      onNext={hasNext ? onNext : undefined}
      onPageSizeChange={setPageSize}
      pageSizes={pageSizes}
    />
  );
}

const meta = {
  title: "design-system/Pagination",
  tags: ["autodocs"],
  argTypes: {
    onPrev: {
      control: false,
      table: { disable: true },
    },
    onNext: {
      control: false,
      table: { disable: true },
    },
    perPage: {
      description: "If the value is not in the options, it will be added.",
    },
  },
  component: Template,
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    currentPage: 0,
    hasPrev: true,
    hasNext: true,
    hasTotal: true,
    count: 100,
    perPage: 10,
    onPrev: fn(),
    onNext: fn(),
    onPageSizeChange: fn(),
  },
};

export const CustomPerPage: Story = {
  args: {
    ...Default.args,
    perPage: 42,
  },
};

export const GridPageSizes: Story = {
  args: {
    ...Default.args,
    perPage: 12,
    pageSizes: [12, 24, 48, 96],
  },
};
