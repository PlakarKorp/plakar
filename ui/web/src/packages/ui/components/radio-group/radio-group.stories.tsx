import { Meta, StoryObj } from "@storybook/react-vite";

import {
  CardRadio,
  CompactCardRadio,
  ExpandableCardRadio,
  Radio,
  RadioGroup,
} from "./radio-group";
import { Tag } from "../tag/tag";

const meta = {
  title: "design-system/Radio Group",
  tags: ["autodocs"],
  component: RadioGroup,
  args: {
    label: "Recurrence",
    children: (
      <>
        <Radio value="option1">Twice daily</Radio>
        <Radio value="option2">Daily</Radio>
        <Radio value="option3">Weekly</Radio>
        <Radio value="option4">Every 30 days</Radio>
        <Radio value="option5">Custom</Radio>
      </>
    ),
    supportText: "Optional support text",
    isRequired: false,
  },
} satisfies Meta<typeof RadioGroup>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Disabled: Story = {
  args: {
    defaultValue: "option3",
    isDisabled: true,
  },
};

export const Invalid: Story = {
  args: {
    defaultValue: "option3",
    isInvalid: true,
  },
};

export const WithDescriptions: Story = {
  args: {
    label: "Default access for new resources",
    supportText: undefined,
    defaultValue: "open",
    spacing: 4,
    children: (
      <>
        <Radio value="open" description="Everyone in the organization">
          Open
        </Radio>
        <Radio
          value="restricted"
          description="Its creator and the organization owners"
        >
          Restricted
        </Radio>
      </>
    ),
  },
};

export const CardRadios: Story = {
  args: {
    label: "Plan",
    supportText: "Pick one option",
    defaultValue: "pro",
    isDisabled: false,
    isInvalid: false,
    children: (
      <div className="flex flex-col gap-2">
        <CardRadio
          value="starter"
          label="Starter"
          description="Basic features for personal use"
        />

        <CardRadio
          value="pro"
          label="Pro"
          description="Advanced features for teams"
        />

        <CardRadio
          value="enterprise"
          label="Enterprise"
          description="Security + compliance at scale"
        />
      </div>
    ),
  },
};

export const ExpandableCardRadios: Story = {
  args: {
    label: "Plan",
    supportText: "The selected plan states what it includes",
    defaultValue: "pro",
    isDisabled: false,
    isInvalid: false,
    children: (
      <div className="flex max-h-72 flex-col gap-2 overflow-y-auto p-px">
        <ExpandableCardRadio
          value="starter"
          label="Starter"
          scrollsIntoViewOnSelect
          expandedContent={
            <ul className="text-secondary list-inside list-disc text-sm">
              <li>1 project</li>
              <li>Community support</li>
            </ul>
          }
        >
          <span className="text-tertiary text-sm">
            Basic features for personal use
          </span>
        </ExpandableCardRadio>

        <ExpandableCardRadio
          value="pro"
          label="Pro"
          scrollsIntoViewOnSelect
          expandedContent={
            <ul className="text-secondary list-inside list-disc text-sm">
              <li>Unlimited projects</li>
              <li>Shared workspaces</li>
              <li>Email support within a day</li>
            </ul>
          }
        >
          <span className="text-tertiary text-sm">
            Advanced features for teams
          </span>
        </ExpandableCardRadio>

        <ExpandableCardRadio
          value="enterprise"
          label="Enterprise"
          scrollsIntoViewOnSelect
          expandedContent={
            <ul className="text-secondary list-inside list-disc text-sm">
              <li>Single sign-on</li>
              <li>Audit trail</li>
              <li>Dedicated support</li>
            </ul>
          }
        >
          <span className="text-tertiary text-sm">
            Security and compliance at scale
          </span>
        </ExpandableCardRadio>
      </div>
    ),
  },
};

export const WithSuffix: Story = {
  args: {
    label: "Plan",
    supportText: "The badge gives way to the check once a card is chosen",
    defaultValue: "pro",
    isDisabled: false,
    isInvalid: false,
    children: (
      <div className="flex flex-col gap-2 p-px">
        <ExpandableCardRadio
          value="starter"
          label="Starter"
          suffix={
            <Tag size="sm" radius="sqrt" variant="blue">
              Read
            </Tag>
          }
        >
          <span className="text-tertiary text-sm">
            Basic features for personal use
          </span>
        </ExpandableCardRadio>

        <ExpandableCardRadio
          value="pro"
          label="Pro"
          suffix={
            <Tag size="sm" radius="sqrt" variant="green">
              Write
            </Tag>
          }
        >
          <span className="text-tertiary text-sm">
            Advanced features for teams
          </span>
        </ExpandableCardRadio>

        <ExpandableCardRadio
          value="enterprise"
          label="Enterprise"
          suffix={
            <Tag size="sm" radius="sqrt" variant="brand">
              Full
            </Tag>
          }
        >
          <span className="text-tertiary text-sm">
            Security and compliance at scale
          </span>
        </ExpandableCardRadio>
      </div>
    ),
  },
};

export const CompactCardRadios: Story = {
  args: {
    label: "Plan",
    supportText: "Pick one option",
    defaultValue: "pro",
    isDisabled: false,
    isInvalid: false,
    children: (
      <div className="flex flex-col gap-2">
        <CompactCardRadio
          value="starter"
          label="Starter"
          description="Basic features for personal use"
        />

        <CompactCardRadio
          value="pro"
          label="Pro"
          description="Advanced features for teams"
        />

        <CompactCardRadio
          value="enterprise"
          label="Enterprise"
          description="Security + compliance at scale"
        />
      </div>
    ),
  },
};
