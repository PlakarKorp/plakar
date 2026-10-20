import { AsyncSelect, AsyncSelectProps } from "@plakar-ui/ui/components";
import { useEffect } from "react";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

type AsyncSelectFieldProps = {
  autoSelectSingleOption?: boolean;
} & Omit<AsyncSelectProps, "value" | "onChange" | "onBlur" | "isInvalid">;

export function AsyncSelectField({
  list,
  supportText,
  autoSelectSingleOption = false,
  ...props
}: AsyncSelectFieldProps) {
  const field = useFieldContext<string>();
  const isReadOnly = useFormReadOnly();

  // Set when there is only one option in the list, and no filter is applied.
  const singleOptionId =
    list.filterText === "" && list.items.length === 1
      ? list.items[0].id
      : undefined;

  // Auto-select the only option.
  useEffect(() => {
    if (!autoSelectSingleOption) {
      return;
    }
    // Wait for the list to finish loading before auto-selecting.
    if (list.loadingState !== "idle") {
      return;
    }
    if (!singleOptionId) {
      return;
    }
    if (!field.state.value) {
      field.setValue(singleOptionId, { dontUpdateMeta: true });
    }
  }, [
    singleOptionId,
    field,
    field.state.value,
    list.loadingState,
    autoSelectSingleOption,
  ]);

  return (
    <AsyncSelect
      list={list}
      value={field.state.value ?? null}
      onChange={(key) => field.handleChange(key ? String(key) : "")}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isReadOnly={isReadOnly || props.isReadOnly}
    />
  );
}
