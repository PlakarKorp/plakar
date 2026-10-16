import { ReactNode, useState } from "react";
import type { Key, Selection } from "react-aria-components";
import {
  Tree as AriaTree,
  TreeItem as AriaTreeItem,
  TreeItemContent,
  Collection,
} from "react-aria-components";

import { Checkbox } from "../checkbox/checkbox";
import { IconButton } from "../icon-button/icon-button";
import { Typography } from "../typography/typography";

type Node = {
  id: string;
  label: string;
  description?: string;
  content?: ReactNode;
  children?: Node[];
};

type TreeProps = {
  items: Node[];
  selectedKeys?: Set<Key>;
  defaultSelectedKeys?: Iterable<Key>;
  onSelectionChange?: (keys: Set<Key>) => void;
};

export function Tree({
  items,
  selectedKeys: controlledSelectedKeys,
  defaultSelectedKeys,
  onSelectionChange,
}: TreeProps) {
  const [uncontrolledSelectedKeys, setUncontrolledSelectedKeys] = useState<
    Set<Key>
  >(() => new Set(defaultSelectedKeys ?? []));
  const [expandedKeys, setExpandedKeys] = useState<Set<Key>>(new Set());

  const selectedKeys = controlledSelectedKeys ?? uncontrolledSelectedKeys;

  function updateSelectedKeys(next: Set<Key>) {
    if (controlledSelectedKeys === undefined) {
      setUncontrolledSelectedKeys(next);
    }
    onSelectionChange?.(next);
  }

  return (
    <AriaTree
      aria-label="Items"
      selectionMode="multiple"
      selectedKeys={selectedKeys}
      onSelectionChange={(keys: Selection) => {
        if (keys === "all") return;
        updateSelectedKeys(new Set(keys));
      }}
      expandedKeys={expandedKeys}
      onExpandedChange={(keys) => setExpandedKeys(new Set(keys))}
      items={items}
      dependencies={[selectedKeys, expandedKeys]}
    >
      {function renderItem(item: Node) {
        const hasChildren = !!item.children?.length;

        const descendantKeys = getDescendantKeys(item);
        const selectedDescendantCount = descendantKeys.filter((k) =>
          selectedKeys.has(k)
        ).length;

        const allSelected = hasChildren
          ? descendantKeys.length > 0 &&
            selectedDescendantCount === descendantKeys.length
          : selectedKeys.has(item.id);

        const someSelected =
          hasChildren &&
          selectedDescendantCount > 0 &&
          selectedDescendantCount < descendantKeys.length;

        return (
          <AriaTreeItem id={item.id} textValue={item.label}>
            <TreeItemContent>
              {({ isExpanded }) => (
                <div className="my-2 flex items-center gap-1 pl-[calc((var(--tree-item-level)-1)*1.5rem)]">
                  {hasChildren ? (
                    <IconButton
                      slot="chevron"
                      icon={isExpanded ? "chevron-down" : "chevron-right"}
                      variant="tertiary"
                      size="sm"
                    />
                  ) : (
                    <span aria-hidden="true" className="w-6 shrink-0" />
                  )}

                  <div className="flex items-baseline gap-1.5">
                    <Checkbox
                      slot="selection"
                      isSelected={allSelected || someSelected}
                      isIndeterminate={someSelected}
                      onChange={(checked) =>
                        updateSelectedKeys(
                          toggleBranch(item, checked, selectedKeys)
                        )
                      }
                      label={item.content ?? item.label}
                    />

                    <Typography size="sm" className="text-tertiary">
                      {item.description}
                    </Typography>
                  </div>
                </div>
              )}
            </TreeItemContent>

            {hasChildren ? (
              <Collection items={item.children}>{renderItem}</Collection>
            ) : null}
          </AriaTreeItem>
        );
      }}
    </AriaTree>
  );
}

function getDescendantKeys(node: Node): string[] {
  const result: string[] = [];
  for (const child of node.children ?? []) {
    result.push(child.id);
    result.push(...getDescendantKeys(child));
  }
  return result;
}

function getBranchKeys(node: Node): string[] {
  return [node.id, ...getDescendantKeys(node)];
}

function toggleBranch(node: Node, checked: boolean, selectedKeys: Set<Key>) {
  const next = new Set(selectedKeys);
  const keys = getBranchKeys(node);

  if (checked) {
    for (const key of keys) next.add(key);
  } else {
    for (const key of keys) next.delete(key);
  }

  return next;
}
