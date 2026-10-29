import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { Card } from "./card";
import { Button } from "../button/button";
import { IllustrationType } from "../illustration/illustration.constants";
import { StatusDot } from "../status-dot/status-dot";
import { Typography } from "../typography/typography";

type CardStoryArgs = ComponentProps<typeof Card> & {
  title?: string;
  content?: string;
  buttonLabel?: string;
  illustration?: IllustrationType;
};

const meta = {
  title: "design-system/Card",
  tags: ["autodocs"],
  component: Card,
  args: {
    isCollapsible: false,
    defaultCollapsed: false,
    isError: false,
  },
  argTypes: {
    className: {
      control: "text",
      table: { type: { summary: "string" } },
    },
    title: {
      control: "text",
      table: { type: { summary: "string" } },
    },
    content: {
      control: "text",
      table: { type: { summary: "string" } },
    },
    buttonLabel: {
      control: "text",
      table: { type: { summary: "string" } },
    },
  },
} satisfies Meta<CardStoryArgs>;

export default meta;

type Story = StoryObj<CardStoryArgs>;

export const WithHeader: Story = {
  args: {
    title: "Card Title",
    content: "This is the content inside the card with a header and button.",
    buttonLabel: "View all",
  },
  argTypes: {
    className: { table: { disable: true } },
  },
  render: ({
    title = "",
    content = "",
    buttonLabel = "",
    isCollapsible = false,
    defaultCollapsed = false,
    isError = false,
  }) => (
    <Card
      isCollapsible={isCollapsible}
      defaultCollapsed={defaultCollapsed}
      isError={isError}
    >
      <Card.Header
        title={title}
        buttonProps={{
          children: buttonLabel,
          rightIcon: "arrow-up-right",
          onPress: () => alert("Button clicked"),
        }}
      />
      <Card.Content>
        <p className="text-primary">{content}</p>
      </Card.Content>
    </Card>
  ),
};

export const WithFooter: Story = {
  args: {
    title: "Card Title",
    content: "This is the content inside a card that ends on a footer.",
    buttonLabel: "Continue",
  },
  argTypes: {
    className: { table: { disable: true } },
    defaultCollapsed: { table: { disable: true } },
    isCollapsible: { table: { disable: true } },
  },
  render: ({ title = "", content = "", buttonLabel = "", isError = false }) => (
    <Card isError={isError}>
      <Card.Header title={title} />
      <Card.Content>
        <p className="text-primary">{content}</p>
      </Card.Content>
      <Card.Footer className="text-right">
        <Button>{buttonLabel}</Button>
      </Card.Footer>
    </Card>
  ),
};

export const ContentOnly: Story = {
  args: {
    content: "This is the content of a card without header.",
  },
  argTypes: {
    title: { table: { disable: true } },
    buttonLabel: { table: { disable: true } },
    defaultCollapsed: { table: { disable: true } },
    isError: { table: { disable: true } },
    isCollapsible: { table: { disable: true } },
    className: { table: { disable: true } },
  },
  render: ({ content = "" }) => (
    <Card>
      <Card.Content>
        <p className="text-primary">{content}</p>
      </Card.Content>
    </Card>
  ),
};

export const Collapsible: Story = {
  args: {
    title: "Card Title",
    content: "This is the content inside the collapsible card.",
    isCollapsible: true,
    defaultCollapsed: true,
    isError: false,
  },
  argTypes: {
    buttonLabel: { table: { disable: true } },
    className: { table: { disable: true } },
  },
  render: ({
    title = "",
    content = "",
    isCollapsible = false,
    defaultCollapsed = false,
    isError = false,
  }) => (
    <Card
      isCollapsible={isCollapsible}
      defaultCollapsed={defaultCollapsed}
      isError={isError}
    >
      <Card.Header title={title} />
      <Card.Content>
        <p className="text-primary">{content}</p>
      </Card.Content>
    </Card>
  ),
};

export const WithTitleSuffix: Story = {
  args: {
    title: "Enrollment",
    content: "A status sits next to the title it qualifies.",
  },
  render: ({ title = "", content = "" }) => (
    <Card>
      <Card.Header
        title={title}
        titleSuffix={
          <span className="inline-flex items-center gap-2">
            <StatusDot variant="success" />
            <Typography as="span" size="sm" className="text-secondary">
              Open
            </Typography>
          </span>
        }
        description="Shown inline with the title, before the description."
      />
      <Card.Content>
        <p className="text-primary">{content}</p>
      </Card.Content>
    </Card>
  ),
};

export const FullWidthDivider: Story = {
  args: {
    title: "Latest restore points",
  },
  render: ({ title = "" }) => (
    <Card className="overflow-hidden">
      <Card.Header
        title={title}
        description="The divider spans the card when a full-width band follows."
        isDividerFullWidth
      />
      <Typography
        size="xs"
        weight="medium"
        className="bg-secondary border-secondary text-tertiary border-b px-4 py-1.5"
      >
        Today
      </Typography>
      <Card.Content>
        <p className="text-primary">Rows go here.</p>
      </Card.Content>
    </Card>
  ),
};

export const CustomStyle: Story = {
  args: {
    title: "Card Title",
    content: "This is the content inside the card with a header and button.",
    buttonLabel: "View all",
    className: "max-w-[250px]",
  },
  argTypes: {
    defaultCollapsed: { table: { disable: true } },
    isCollapsible: { table: { disable: true } },
    isError: { table: { disable: true } },
  },
  render: ({ title = "", content = "", buttonLabel = "", className = "" }) => (
    <Card className={className}>
      <Card.Header
        title={title}
        buttonProps={{
          children: buttonLabel,
          rightIcon: "arrow-up-right",
          onPress: () => alert("Button clicked"),
        }}
      />
      <Card.Content>
        <p className="text-primary">{content}</p>
      </Card.Content>
    </Card>
  ),
};
