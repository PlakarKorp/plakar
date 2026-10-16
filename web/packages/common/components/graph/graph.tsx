import {
  ReactFlow,
  applyNodeChanges,
  applyEdgeChanges,
  addEdge,
  EdgeChange,
  Connection,
  NodeChange,
  Node,
  Edge,
  Background,
  Panel,
} from "@xyflow/react";
import React, { useState, useCallback } from "react";

import "@xyflow/react/dist/style.css";
import { TaskEdge } from "./edges";
import { DestinationNode, SourceNode, StoreNode } from "./nodes";

const nodeTypes = {
  source: SourceNode,
  store: StoreNode,
  destination: DestinationNode,
};

const edgeTypes = {
  task: TaskEdge,
};

type GraphProps = {
  initialNodes: Node[];
  initialEdges: Edge[];
  controls?: React.ReactNode;
  overlay?: React.ReactNode;
  onEdgeSelect: (edgeId?: string) => void;
  onConnect: (params: Connection) => Promise<boolean> | boolean;
  onRemove: (changes: EdgeChange[]) => Promise<boolean> | boolean;
};

export function Graph({
  initialNodes,
  initialEdges,
  controls,
  overlay,
  onEdgeSelect,
  onConnect,
  onRemove,
}: GraphProps) {
  const [nodes, setNodes] = useState<Node[]>(initialNodes);
  const [edges, setEdges] = useState<Edge[]>(initialEdges);

  const handleNodesChange = useCallback(
    (changes: NodeChange[]) =>
      setNodes((nodesSnapshot) => applyNodeChanges(changes, nodesSnapshot)),
    []
  );
  const handleEdgesChange = useCallback(
    async (changes: EdgeChange[]) => {
      const selectChanges = changes.filter(
        (change) => change.type === "select"
      );
      if (selectChanges.length > 0) {
        const selectedEdgeId = selectChanges.find((c) => c.selected)?.id;
        onEdgeSelect(selectedEdgeId);

        // Update nodes based on edge selection
        if (selectedEdgeId) {
          const selectedEdge = edges.find((edge) => edge.id === selectedEdgeId);
          if (selectedEdge) {
            const isCheckEdge = selectedEdge.source === selectedEdge.target;

            setNodes((nodesSnapshot) =>
              nodesSnapshot.map((node) => ({
                ...node,
                data: {
                  ...node.data,
                  isOriginSelected:
                    !isCheckEdge && node.id === selectedEdge.source,
                  isTargetSelected:
                    !isCheckEdge && node.id === selectedEdge.target,
                  isCheckSelected:
                    isCheckEdge && node.id === selectedEdge.source,
                },
              }))
            );

            // Update edges to bring selected edge on top
            setEdges((edgesSnapshot) =>
              edgesSnapshot.map((edge) => ({
                ...edge,
                zIndex: edge.id === selectedEdgeId ? 2 : 1,
              }))
            );
          }
        } else {
          // No edge selected, reset all nodes and edges
          setNodes((nodesSnapshot) =>
            nodesSnapshot.map((node) => ({
              ...node,
              data: {
                ...node.data,
                isOriginSelected: false,
                isTargetSelected: false,
                isCheckSelected: false,
              },
            }))
          );

          setEdges((edgesSnapshot) =>
            edgesSnapshot.map((edge) => ({
              ...edge,
              zIndex: 1,
            }))
          );
        }
      }

      const removalChanges = changes.filter(
        (change) => change.type === "remove"
      );
      if (removalChanges.length > 0) {
        const result = await onRemove(changes);
        if (!result) {
          return;
        }
      }

      setEdges((edgesSnapshot) => applyEdgeChanges(changes, edgesSnapshot));
    },
    [onEdgeSelect, onRemove, edges]
  );
  const handleConnect = useCallback(
    async (params: Connection) => {
      const sourceNode = nodes.find((node) => node.id === params.source);
      const targetNode = nodes.find((node) => node.id === params.target);

      // Connections to prevent
      if (sourceNode?.type === "source" && targetNode?.type === "destination") {
        return;
      }

      const result = await onConnect(params);
      if (!result) {
        return;
      }

      setEdges((edgesSnapshot) => {
        let data;
        if (targetNode?.type === "destination") {
          data = { label: "Restore", icon: "download" };
        } else if (
          sourceNode?.type === "store" &&
          targetNode?.type === "store"
        ) {
          data = { label: "Sync", icon: "arrows-rotate" };
        } else {
          data = { label: "Backup", icon: "arrow-up-from-bracket" };
        }

        return addEdge(
          {
            ...params,
            data,
            type: "task",
          },
          edgesSnapshot
        );
      });
    },
    [nodes, onConnect]
  );

  return (
    <div style={{ width: "100%", height: "100%" }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        onConnect={handleConnect}
        multiSelectionKeyCode={null}
        fitView
        fitViewOptions={{
          padding: 0.25,
          maxZoom: 1,
        }}
      >
        <Background />
        {controls && (
          <Panel position="top-right" style={{ margin: 0 }}>
            {controls}
          </Panel>
        )}
        {overlay && (
          <Panel position="bottom-center" style={{ width: "100%" }}>
            {overlay}
          </Panel>
        )}
      </ReactFlow>
    </div>
  );
}
