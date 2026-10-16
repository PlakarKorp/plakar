import { IconType, IconButton } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { ReactNode, useRef, useState } from "react";
import {
  Group,
  Panel,
  PanelImperativeHandle,
  Separator,
  useDefaultLayout,
} from "react-resizable-panels";

import { useSidePanel } from ".";
import { ValidPositions } from "./side-panel-provider.constants";
import { useBreakpoint } from "../../hooks";

type ControlType = {
  icon: IconType;
  tooltip?: string;
  isDisabled?: boolean;
  onPress: () => void;
};

type SidePanelLayoutProps = {
  controls?: ControlType[];
  main: ReactNode;
  sidebar?: ReactNode;
  onCollapse?: () => void;
};

export function SidePanelLayout({
  controls = [],
  main,
  sidebar,
  onCollapse,
}: SidePanelLayoutProps) {
  const { position, moveSidebar } = useSidePanel();

  const sidebarRef = useRef<PanelImperativeHandle>(null);

  const collapseSidebar = () => {
    const panel = sidebarRef.current;
    if (panel) {
      if (panel.isCollapsed()) {
        panel.expand();
      } else {
        if (onCollapse) {
          onCollapse();
        } else {
          panel.collapse();
        }
      }
    }
  };

  const [collapsed, setCollapsed] = useState(false);
  const { isMdDown } = useBreakpoint();
  const { defaultLayout, onLayoutChanged } = useDefaultLayout({
    id: "sidebar",
  });

  // On small screens, the position of the sidebar is always vertical, no matter
  // what is stored in local storage.
  const actualPosition = isMdDown ? "vertical" : position;
  const sidebarMinSize = actualPosition === "horizontal" ? 20 : 3;
  const minSize = actualPosition === "horizontal" ? 30 : 10;

  return (
    <div className="h-full">
      <Group
        orientation={actualPosition}
        defaultLayout={defaultLayout}
        onLayoutChanged={onLayoutChanged}
      >
        <Panel id="main" minSize={minSize} className="relative">
          {main}
        </Panel>

        {sidebar && (
          <>
            <Separator
              className={clsx(
                "border-primary hover:border-accent-heavy relative flex items-center justify-center",
                actualPosition === "horizontal"
                  ? "cursor-ew-resize border-l-2"
                  : "cursor-ns-resize border-t-2"
              )}
            >
              <div className="absolute inset-0 -m-2" />
            </Separator>

            <Panel
              id="sidebar"
              defaultSize="50"
              minSize={sidebarMinSize}
              className="bg-primary min-h-[50px] min-w-[58px] flex-shrink-0 overflow-hidden md:pt-4"
              collapsible
              collapsedSize={sidebarMinSize}
              panelRef={sidebarRef}
              onResize={() =>
                setCollapsed(sidebarRef.current?.isCollapsed() ?? false)
              }
            >
              <div className="flex h-full flex-col gap-4">
                <div className="hidden px-4 lg:block">
                  <PanelControls
                    moveSidebar={moveSidebar}
                    collapseSidebar={collapseSidebar}
                    controls={controls}
                    position={actualPosition}
                    collapsed={collapsed}
                  />
                </div>
                <div className="flex min-h-0 flex-1 flex-col overflow-auto">
                  <div className="h-full pt-4 md:pt-0">{sidebar}</div>
                </div>
              </div>
            </Panel>
          </>
        )}
      </Group>
    </div>
  );
}

type PanelControlsProps = {
  moveSidebar: (newPosition: (typeof ValidPositions)[number]) => void;
  collapseSidebar: () => void;
  controls: ControlType[];
  position: (typeof ValidPositions)[number];
  collapsed: boolean;
};

function PanelControls({
  moveSidebar,
  collapseSidebar,
  controls,
  position,
  collapsed,
}: PanelControlsProps) {
  return (
    <div className="text-action-tertiary flex items-center justify-between gap-x-4">
      {/* Collapse icon + controls  */}
      <div className="flex items-center">
        <IconButton
          tooltip={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          variant="tertiary"
          size="sm"
          icon={
            position === "horizontal"
              ? collapsed
                ? "chevrons-left"
                : "chevrons-right"
              : collapsed
                ? "chevrons-up"
                : "chevrons-down"
          }
          onPress={collapseSidebar}
        />
        {controls.length > 0 && (
          <div className="border-primary ml-2 flex gap-2 border-l pl-2">
            {controls.map((control) => (
              <IconButton
                tooltip={control.tooltip}
                key={control.icon}
                variant="tertiary"
                size="sm"
                icon={control.icon}
                isDisabled={control.isDisabled}
                onPress={control.onPress}
              />
            ))}
          </div>
        )}
      </div>

      {/* Change position */}
      <div className="flex gap-2">
        <IconButton
          tooltip="Move the sidebar to the right"
          variant="tertiary"
          size="sm"
          icon="sidebar-right"
          onPress={() => moveSidebar("horizontal")}
          isDisabled={position === "horizontal"}
        />
        <IconButton
          tooltip="Move the sidebar to the bottom"
          variant="tertiary"
          size="sm"
          icon="sidebar-bottom"
          onPress={() => moveSidebar("vertical")}
          isDisabled={position === "vertical"}
        />
      </div>
    </div>
  );
}
