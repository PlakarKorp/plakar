import { DescriptionList } from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { ConnectorCard } from "./connector-card";

function Template(props: ComponentProps<typeof ConnectorCard.Content>) {
  return (
    <div className="w-140">
      <ConnectorCard isFullWidth>
        <ConnectorCard.Content {...props} />
      </ConnectorCard>
    </div>
  );
}

const meta = {
  title: "Components/Connector Card",
  tags: ["autodocs"],
  args: {
    inventory: {
      icon: "aws",
      iconBadgeType: "aws",
      name: "AWS Inventory",
    },
    resource: {
      icon: "server",
      iconBadgeType: "default",
      name: "EC2 instance",
      resourceClass: "Compute",
      urn: "arn:aws:resource-explorer-2:eu-west-3:331208495627:view/eu-west-3/72bd6517-7f1e-4bc6-a9c6-ed4eb7901f28",
      tags: ["tag1", "tag2", "tag3"],
      residency: {
        alpha2: "FR",
        flagEmoji: "🇫🇷",
        name: "France",
      },
    },
    connector: {
      type: "source",
      name: "My Connector",
      integrationIcon: "s3",
      search: "",
    },
    details: (
      <DescriptionList size="xs" direction="row" singleColumn>
        <DescriptionList.Item
          title="Connector Id"
          description="7af3bb46-9071-4025-b486-7d9ad9f5c971"
          truncate={false}
          textOnBrand
        />
        <DescriptionList.Item
          title="URN"
          description="arn:aws:resource-explorer-2:eu-west-3:331208495627:view/eu-west-3/72bd6517-7f1e-4bc6-a9c6-ed4eb7901f28"
          truncate={false}
          textOnBrand
        />
      </DescriptionList>
    ),
  },
  component: Template,
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithoutCard: Story = {
  render: (args) => {
    return (
      <div className="w-140">
        <ConnectorCard.Content {...args} />
      </div>
    );
  },
};

const LONG_NAME = "averylongconnectornamewithoutanyspacesorhyphenstotruncate";

export const ConnectorTruncate: Story = {
  render: (args) => {
    return (
      <div className="flex w-70 flex-col gap-4">
        <div>
          <p className="text-tertiary mb-1 text-xs">default</p>
          <ConnectorCard.Connector {...args.connector} name={LONG_NAME} />
        </div>
        <div>
          <p className="text-tertiary mb-1 text-xs">truncate</p>
          <ConnectorCard.Connector
            {...args.connector}
            name={LONG_NAME}
            truncate
          />
        </div>
      </div>
    );
  },
};
