import { Icon, IconType, Typography } from "@plakar-ui/ui/components";
import { BaseEdge, EdgeLabelRenderer, EdgeProps } from "@xyflow/react";
import clsx from "clsx";

// Self loop dimensions for check task
const SELF_LOOP_WIDTH = 120;

export type TaskEdgeProps = EdgeProps & {
  data: {
    label: string;
    icon: IconType;
    isError?: boolean;
  };
};

export function TaskEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  selected,
  data,
  source,
  target,
}: TaskEdgeProps) {
  let edgePath: string;
  let labelX: number;
  let labelY: number;
  let translateX = "-50%";

  // self loop path for check task
  if (source === target) {
    const rightX = sourceX + SELF_LOOP_WIDTH;

    edgePath = `M ${sourceX} ${sourceY} L ${rightX} ${sourceY}`;
    labelX = rightX;
    labelY = sourceY;
    translateX = "-100%";
  }
  // step path for other tasks
  else {
    const centerY = (targetY - sourceY) / 2 + sourceY;

    edgePath = `M ${sourceX} ${sourceY} L ${sourceX} ${centerY} L ${targetX} ${centerY} L ${targetX} ${targetY}`;
    labelX = (sourceX + targetX) / 2;
    labelY = centerY;
  }

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        interactionWidth={30}
        style={{
          stroke: selected
            ? "var(--background-color-accent)"
            : data.isError
              ? "var(--background-color-error)"
              : "var(--background-color-success)",
          strokeWidth: 2,
        }}
      />
      <EdgeLabelRenderer>
        <div
          className={clsx(
            "bg-secondary flex items-center gap-2 rounded-md px-1.5 py-0.5",
            selected ? "text-secondary" : "text-tertiary"
          )}
          style={{
            position: "absolute",
            transform: `translate(${translateX}, -50%) translate(${labelX}px, ${labelY}px)`,
            zIndex: 3,
          }}
        >
          <Icon type={data.icon} size="md" />
          <Typography weight="medium">{data.label}</Typography>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}
