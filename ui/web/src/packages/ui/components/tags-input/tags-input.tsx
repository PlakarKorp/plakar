import clsx from "clsx";
import { useState } from "react";
import { TextField, Input, type TextFieldProps } from "react-aria-components";

import { RemovableTags, Typography } from "..";
import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";

type TagItem = { id: number; label: string };

export type TagsInputProps = Omit<TextFieldProps, "onChange"> & {
  label?: string;
  placeholder?: string;
  supportText?: string;
  tags?: string[];
  onChange?: (tags: string[]) => void;
  /** Optional validator for new tags; return false to block add */
  validateTag?: (value: string) => boolean;
};

export function TagsInput({
  label,
  placeholder = "Add a tag…",
  supportText,
  className,
  tags = [],
  onChange,
  validateTag,
  ...props
}: TagsInputProps) {
  // keep duplicate initial tags by giving each its own ID
  const items = tags.map((t, i) => ({
    id: i,
    label: String(t).trim(),
  }));

  const [value, setValue] = useState("");

  const setTags = (next: TagItem[]) => {
    onChange?.(next.map((t) => t.label));
  };

  const addTag = (text?: string) => {
    const value = String(text ?? "").trim();

    if (!value) return;

    setValue("");

    if (validateTag && !validateTag(value)) return;
    // block new duplicates (case-insensitive)
    if (items.some((t) => t.label.toLowerCase() === value.toLowerCase())) {
      return;
    }

    setTags([...items, { id: items.length, label: value }]);
  };

  const removeTag = (id: number) => {
    setTags(items.filter((t) => t.id !== id));
  };

  return (
    <TextField
      {...props}
      className={clsx("group flex flex-col gap-1", className)}
    >
      <FieldLabel isRequired={props.isRequired}>{label}</FieldLabel>

      <Typography
        as="div"
        size="sm"
        className={clsx(
          "border-primary bg-primary text-primary",
          "flex flex-wrap items-center gap-2 rounded-lg border px-3 py-1 -outline-offset-1",
          !props.isDisabled && "hover:bg-primary-hover",
          "group-data-invalid:border-error",
          "group-data-disabled:border-tertiary group-data-disabled:bg-disable group-data-disabled:text-disable",
          props.isReadOnly
            ? "bg-tertiary text-primary border-tertiary hover:bg-tertiary"
            : "has-data-focused:outline-focus has-data-focused:outline-2"
        )}
      >
        {items.length > 0 && (
          <RemovableTags
            label={label || "Tags"}
            disabledKeys={
              props.isDisabled || props.isReadOnly
                ? items.map((t) => t.id)
                : undefined
            }
            onRemove={(keys) => setTags(items.filter((t) => !keys.has(t.id)))}
          >
            {items.map(({ id, label }) => (
              <RemovableTags.Item id={id} key={id} value={label} size="sm" />
            ))}
          </RemovableTags>
        )}

        <Input
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === "," || e.key === ";") {
              e.preventDefault();
              addTag(value);
            } else if (e.key === "Backspace" && !value && items.length) {
              removeTag(items[items.length - 1].id);
            }
          }}
          onBlur={() => addTag(value)}
          placeholder={!props.isReadOnly ? placeholder : undefined}
          className="placeholder:text-tertiary w-full min-w-24 flex-1 py-1 outline-none"
        />
      </Typography>

      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}
    </TextField>
  );
}
