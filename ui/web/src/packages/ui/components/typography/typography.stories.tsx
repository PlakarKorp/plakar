import type { Meta, StoryObj } from "@storybook/react-vite";

import { Typography } from "./typography";
import {
  TypographySizes,
  TypographyVariants,
  TypographyWeights,
} from "./typography.constants";

const meta = {
  title: "design-system/Typography",
  tags: ["autodocs"],
  component: Typography,
  argTypes: {
    variant: {
      control: { type: "radio" },
      options: TypographyVariants,
    },
    size: {
      control: { type: "radio" },
      options: TypographySizes,
    },
    weight: {
      control: { type: "radio" },
      options: TypographyWeights,
    },
    monospace: {
      control: { type: "boolean" },
    },
    as: {
      control: { type: "select" },
      options: [undefined, "p", "span", "h1", "h2", "h3", "h4", "h5", "h6"],
      table: {
        type: { summary: `"p" | "span" | "h1" | "h2" | ...` },
      },
    },
    children: { control: "text" },
  },
} satisfies Meta<typeof Typography>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    variant: "heading",
    size: "md",
    children: "Heading",
    monospace: false,
  },
};

export const AllHeadings = {
  argTypes: {
    as: { table: { disable: true } },
  },
  render: () => (
    <div className="flex flex-col">
      {TypographySizes.map((size) =>
        TypographyWeights.map((weight) => (
          <Typography
            key={`${size}-${weight}`}
            variant="heading"
            size={size}
            weight={weight}
          >
            Size {size.toUpperCase()} {weight.toUpperCase()}
          </Typography>
        ))
      )}
    </div>
  ),
};

export const AllBody = {
  argTypes: {
    as: { table: { disable: true } },
  },
  render: () => (
    <div className="flex flex-col">
      {TypographySizes.map((size) =>
        TypographyWeights.map((weight) => (
          <Typography
            key={`${size}-${weight}`}
            variant="body"
            size={size}
            weight={weight}
          >
            Size {size.toUpperCase()} {weight.toUpperCase()}
          </Typography>
        ))
      )}
    </div>
  ),
};

export const AllMono = {
  argTypes: {
    as: { table: { disable: true } },
  },
  render: () => (
    <div className="flex flex-col">
      {TypographySizes.map((size) =>
        TypographyWeights.map((weight) => (
          <Typography
            key={`${size}-${weight}`}
            variant="body"
            size={size}
            weight={weight}
            monospace
          >
            Size {size.toUpperCase()} {weight.toUpperCase()}
          </Typography>
        ))
      )}
    </div>
  ),
};
