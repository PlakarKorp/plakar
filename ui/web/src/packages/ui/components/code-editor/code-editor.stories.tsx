import type { Meta, StoryObj } from "@storybook/react-vite";

import { CodeEditor } from "./code-editor";

const meta = {
  title: "design-system/Code Editor",
  tags: ["autodocs"],
  argTypes: {
    suffix: {
      control: false,
      table: { disable: true },
    },
  },
  args: {
    label: "Configuration",
    isRequired: true,
    supportText: "Enter your JSON configuration here",
    isInvalid: false,
    placeholder: "Enter your code here…",
  },
  component: CodeEditor,
} satisfies Meta<typeof CodeEditor>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => {
    return (
      <div className="h-128">
        <CodeEditor {...args} />
      </div>
    );
  },
};

export const WithRows: Story = {
  args: {
    rows: 3,
  },
};

const PREVIOUS = JSON.stringify(
  {
    name: "AWS staging S3",
    region: "eu-west-1",
    complete: false,
    ready: false,
    retries: 3,
    timeout: "30s",
  },
  null,
  2
);

const CURRENT = JSON.stringify(
  {
    name: "AWS staging S3",
    region: "eu-west-3",
    complete: true,
    ready: true,
    retries: 3,
    timeout: "30s",
  },
  null,
  2
);

/**
 * Passing `original` turns the editor into a single unified diff: removals sit
 * above the line that replaced them, and unchanged stretches collapse.
 */
export const Diff: Story = {
  args: {
    label: "Payload",
    isRequired: false,
    supportText: undefined,
    value: CURRENT,
    original: PREVIOUS,
    readOnly: true,
  },
  render: (args) => {
    return (
      <div className="h-128">
        <CodeEditor {...args} />
      </div>
    );
  },
};
