import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { Card } from "./card";
import { Button } from "../button/button";
import { IllustrationType } from "../illustration/illustration.constants";

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
