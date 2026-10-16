import { Modal } from "@plakar-ui/ui/components";
import { Button } from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";
import type { Node, Edge, Connection } from "@xyflow/react";
import { ComponentProps, useState } from "react";

import { Graph } from "./graph";
import type { ConfigurationNodeData } from "./nodes";

const initialNodes: Node<ConfigurationNodeData>[] = [
  {
    id: "source1",
    position: { x: -300, y: 0 },
    data: {
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
        urn: "arn:aws:ec2:eu-west-3:331208495627:instance/i-0a1b2c3d4e5f6a7b8",
        tags: ["production", "critical"],
        residency: { alpha2: "FR", flagEmoji: "🇫🇷", name: "France" },
      },
      connector: {
        type: "source",
        name: "S3 Source",
        integrationIcon: "s3",
      },
    },
    type: "source",
  },
  {
    id: "source2",
    position: { x: 300, y: 0 },
    data: {
      inventory: {
        icon: "hard-drive",
        iconBadgeType: "default",
        name: "On-premise Inventory",
      },
      resource: {
        icon: "folder",
        iconBadgeType: "default",
        name: "legacy-share",
        resourceClass: "Filesystem",
        urn: "ftp://ftp.example.com/plakar",
        tags: ["legacy", "to-migrate"],
        residency: { alpha2: "DE", flagEmoji: "🇩🇪", name: "Germany" },
      },
      connector: {
        type: "source",
        name: "FTP Source",
        integrationIcon: "ftp",
      },
    },
    type: "source",
  },
  {
    id: "store1",
    position: { x: 0, y: 175 },
    data: {
      inventory: {
        icon: "azblob",
        iconBadgeType: "default",
        name: "Azure Inventory",
      },
      resource: {
        icon: "database",
        iconBadgeType: "default",
        name: "plakar-archive",
        resourceClass: "Blob container",
        urn: "https://myaccount.blob.core.windows.net/plakar",
        tags: ["archive", "compliance"],
        residency: { alpha2: "NL", flagEmoji: "🇳🇱", name: "Netherlands" },
      },
      connector: {
        type: "store",
        name: "Azure Blob Store",
        integrationIcon: "azblob",
      },
      isOriginError: true,
    },
    type: "store",
  },
  {
    id: "store2",
    position: { x: 0, y: 350 },
    data: {
      inventory: {
        icon: "aws",
        iconBadgeType: "aws",
        name: "AWS Inventory",
      },
      resource: {
        icon: "database",
        iconBadgeType: "default",
        name: "archive.example.com",
        resourceClass: "Bucket",
        urn: "arn:aws:s3:::archive.example.com",
        tags: ["backup", "long-term"],
        residency: { alpha2: "IE", flagEmoji: "🇮🇪", name: "Ireland" },
      },
      connector: {
        type: "store",
        name: "S3 Archive Store",
        integrationIcon: "s3",
      },
      isTargetError: true,
    },
    type: "store",
  },
  {
    id: "destination1",
    position: { x: 0, y: 525 },
    data: {
      inventory: {
        icon: "gcp",
        iconBadgeType: "default",
        name: "GCP Inventory",
      },
      resource: {
        icon: "database",
        iconBadgeType: "default",
        name: "data.example.com",
        resourceClass: "Bucket",
        urn: "gs://data.example.com/plakar",
        tags: ["gcp", "fast-access"],
        residency: { alpha2: "BE", flagEmoji: "🇧🇪", name: "Belgium" },
      },
      connector: {
        type: "destination",
        name: "GCS Destination",
        integrationIcon: "gcs",
      },
    },
    type: "destination",
  },
];

const initialEdges: Edge[] = [
  {
    id: "source1-store1",
    source: "source1",
    sourceHandle: "backup-source",
    target: "store1",
    targetHandle: "store-target",
    data: { label: "Backup", icon: "arrow-up-from-bracket" },
    type: "task",
  },
  {
    id: "store1-store2",
    source: "store1",
    sourceHandle: "store-source",
    target: "store2",
    targetHandle: "store-target",
    data: { label: "Sync", icon: "arrows-rotate", isError: true },
    type: "task",
  },
  {
    id: "store2-check",
    source: "store2",
    sourceHandle: "check-source",
    target: "store2",
    data: { label: "Check", icon: "check" },
    type: "task",
  },
];

const meta: Meta<typeof Graph> = {
  title: "Components/Graph",
  component: Graph,
  parameters: {
    layout: "fullscreen",
  },
  decorators: [
    (Story) => (
      <div className="bg-secondary" style={{ width: "100%", height: "800px" }}>
        <Story />
      </div>
    ),
  ],
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    initialNodes,
    initialEdges,
    onEdgeSelect: (edgeId) => {
      console.log("Edge clicked:", edgeId);
    },
    onConnect: async (params) => {
      console.log("Connected:", params);
      return true;
    },
  },
};

const WithModalTemplate = (args: ComponentProps<typeof Graph>) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [connectionParams, setConnectionParams] = useState<Connection | null>(
    null
  );
  const [resolveConnection, setResolveConnection] = useState<
    ((value: boolean) => void) | null
  >(null);

  const handleConnect = async (params: Connection): Promise<boolean> => {
    return new Promise((resolve) => {
      setConnectionParams(params);
      setIsModalOpen(true);
      setResolveConnection(() => resolve);
    });
  };

  const handleConfirm = () => {
    if (resolveConnection) {
      resolveConnection(true);
    }
    setIsModalOpen(false);
    setConnectionParams(null);
    setResolveConnection(null);
  };

  const handleCancel = () => {
    if (resolveConnection) {
      resolveConnection(false);
    }
    setIsModalOpen(false);
    setConnectionParams(null);
    setResolveConnection(null);
  };

  return (
    <>
      <Graph {...args} onConnect={handleConnect} />
      <Modal
        isOpen={isModalOpen}
        onOpenChange={setIsModalOpen}
        aria-label="Confirm Task"
      >
        <Modal.Dialog className="max-w-md">
          <Modal.Header
            title="Confirm Task"
            description="Do you want to create this task?"
          />
          <Modal.Content>
            {connectionParams && (
              <div>
                <p>You are about to connect:</p>
                <p>
                  <strong>From:</strong> {connectionParams.source}
                </p>
                <p>
                  <strong>To:</strong> {connectionParams.target}
                </p>
              </div>
            )}
          </Modal.Content>
          <Modal.Footer>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onPress={handleCancel}>
                Cancel
              </Button>
              <Button variant="primary" onPress={handleConfirm}>
                Confirm
              </Button>
            </div>
          </Modal.Footer>
        </Modal.Dialog>
      </Modal>
    </>
  );
};

export const WithModal: Story = {
  render: WithModalTemplate,
  args: {
    initialNodes,
    initialEdges,
  },
};
