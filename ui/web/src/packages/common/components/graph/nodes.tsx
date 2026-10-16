import {
  Handle,
  HandleProps,
  Position,
  useNodeConnections,
} from "@xyflow/react";
import clsx from "clsx";

import {
  ConnectorCard,
  ConnectorCardContentProps,
} from "../connector-card/connector-card";

export type ConfigurationNodeData = ConnectorCardContentProps & {
  isOriginError?: boolean;
  isTargetError?: boolean;
  isOriginSelected?: boolean;
  isTargetSelected?: boolean;
  isCheckError?: boolean;
  isCheckSelected?: boolean;
};

interface ConfigurationNodeProps {
  data: ConfigurationNodeData;
}

export function SourceNode({ data }: ConfigurationNodeProps) {
  const sourceConnections = useNodeConnections({
    handleType: "source",
  });

  return (
    <div
      className={clsx(
        "w-140",
        data.isOriginSelected &&
          "outline-accent rounded-lg outline -outline-offset-1"
      )}
    >
      <ConnectorCard isFullWidth>
        <ConnectorCard.Content {...data} />
      </ConnectorCard>

      <CustomHandle
        id="backup-source"
        type="source"
        position={Position.Bottom}
        isConnected={sourceConnections.length > 0}
        isError={data.isOriginError}
        isSelected={data.isOriginSelected}
      />
    </div>
  );
}

export function StoreNode({ data }: ConfigurationNodeProps) {
  const backupTargetConnections = useNodeConnections({
    handleType: "target",
    handleId: "store-target",
  });
  const restoreSourceConnections = useNodeConnections({
    handleType: "source",
    handleId: "store-source",
  });
  const checkSourceConnections = useNodeConnections({
    handleType: "source",
    handleId: "check-source",
  });

  return (
    <div
      className={clsx(
        "w-140",
        (data.isOriginSelected ||
          data.isTargetSelected ||
          data.isCheckSelected) &&
          "outline-accent rounded-lg outline -outline-offset-1"
      )}
    >
      <ConnectorCard isFullWidth>
        <ConnectorCard.Content {...data} />
      </ConnectorCard>

      <CustomHandle
        id="store-source"
        type="source"
        position={Position.Bottom}
        isConnected={restoreSourceConnections.length > 0}
        isError={data.isOriginError}
        isSelected={data.isOriginSelected}
      />
      <CustomHandle
        id="store-target"
        type="target"
        position={Position.Top}
        isConnected={backupTargetConnections.length > 0}
        isError={data.isTargetError}
        isSelected={data.isTargetSelected}
      />
      <CustomHandle
        id="check-source"
        type="source"
        position={Position.Right}
        isConnected={checkSourceConnections.length > 0}
        isError={data.isCheckError}
        isSelected={data.isCheckSelected}
      />
    </div>
  );
}

export function DestinationNode({ data }: ConfigurationNodeProps) {
  const targetConnections = useNodeConnections({
    handleType: "target",
  });

  return (
    <div
      className={clsx(
        "w-140",
        data.isTargetSelected &&
          "outline-accent rounded-lg outline -outline-offset-1"
      )}
    >
      <ConnectorCard isFullWidth>
        <ConnectorCard.Content {...data} />
      </ConnectorCard>

      <CustomHandle
        id="restore-target"
        type="target"
        position={Position.Top}
        isConnected={targetConnections.length > 0}
        isError={data.isTargetError}
        isSelected={data.isTargetSelected}
      />
    </div>
  );
}

type CustomHandleProps = HandleProps & {
  isConnected?: boolean;
  isError?: boolean;
  isSelected?: boolean;
};

function CustomHandle({
  isConnected,
  isError,
  isSelected,
  ...props
}: CustomHandleProps) {
  return (
    <Handle
      {...props}
      style={{
        background: "none",
        border: "none",
        width: "0.625em",
        height: "0.625em",
        ...props.style,
      }}
    >
      <div
        className={clsx(
          "h-2.5 w-2.5 rounded-full border-2",
          isSelected
            ? "border-accent bg-accent-tertiary"
            : isConnected
              ? isError
                ? "border-error bg-error-tertiary"
                : "border-success bg-success-tertiary"
              : "border-accent-heavy bg-tertiary"
        )}
      />
    </Handle>
  );
}
