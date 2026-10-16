import {
  TagToggleGroup,
  TagToggleGroupRootProps,
  Typography,
} from "@plakar-ui/ui/components";
import { ReactNode } from "react";
import { Selection } from "react-aria-components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

type TagToggleGroupFieldProps = Omit<
  TagToggleGroupRootProps,
  | "children"
  | "selectedKeys"
  | "defaultSelectedKeys"
  | "onSelectionChange"
  | "isInvalid"
> & {
  items: { id: string; label: string }[];
  emptyMessage?: ReactNode;
};

export function TagToggleGroupField({
  items,
  supportText,
  selectionMode,
  emptyMessage,
  ...props
}: TagToggleGroupFieldProps) {
  const field = useFieldContext<string | string[]>();
  const isReadOnly = useFormReadOnly();

  function handleSelectionChange(keys: Selection) {
    const selected =
      keys === "all" ? items.map((item) => item.id) : [...keys].map(String);
    field.handleChange(
      selectionMode === "single" ? (selected[0] ?? "") : selected
    );
  }

  const selectedKeys = Array.isArray(field.state.value)
    ? field.state.value
    : field.state.value
      ? [field.state.value]
      : [];

  return (
    <TagToggleGroup
      selectionMode={selectionMode}
      selectedKeys={selectedKeys}
      onSelectionChange={handleSelectionChange}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isDisabled={isReadOnly || props.isDisabled}
    >
      {items.map(({ id, label }) => (
        <TagToggleGroup.Button key={id} id={id}>
          {label}
        </TagToggleGroup.Button>
      ))}
      {items.length === 0 && (
        <Typography size="sm" className="text-tertiary italic">
          {emptyMessage}
        </Typography>
      )}
    </TagToggleGroup>
  );
}
