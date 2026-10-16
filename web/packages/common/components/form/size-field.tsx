import { FieldLabel, NumberInput, Select } from "@plakar-ui/ui/components";
import { useState } from "react";

import { FieldSupportText } from "../../../ui/components/field/field-support-text";
import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";
import {
  getAmountFromBytes,
  getBytesFromSize,
  getSizeFromBytes,
  SizeUnit,
  SizeUnits,
} from "../../utils/size";

const DEFAULT_UNIT: SizeUnit = "GB";

type SizeFieldProps = {
  label?: string;
  placeholder?: string;
  supportText?: string;
};

/**
 * A byte count, entered as an amount and a unit. An empty amount is `null`, the
 * way an empty duration is, so a field left blank says "no value" rather than
 * zero bytes.
 */
export function SizeField({ label, placeholder, supportText }: SizeFieldProps) {
  const field = useFieldContext<number | null>();
  const isReadOnly = useFormReadOnly();

  const [unit, setUnit] = useState<SizeUnit>(
    field.state.value === null
      ? DEFAULT_UNIT
      : getSizeFromBytes(field.state.value).unit
  );

  const amount =
    field.state.value === null
      ? NaN
      : getAmountFromBytes(field.state.value, unit);

  const handleChangeAmount = (value: number) => {
    field.handleChange(
      Number.isNaN(value) ? null : getBytesFromSize(value, unit)
    );
  };

  const handleChangeUnit = (value: SizeUnit) => {
    setUnit(value);

    if (!Number.isNaN(amount)) {
      field.handleChange(getBytesFromSize(amount, value));
    }
  };

  const isInvalid = field.state.meta.errors.length > 0;

  return (
    <div
      className="group flex flex-col gap-1"
      data-invalid={isInvalid || undefined}
    >
      {label && <FieldLabel>{label}</FieldLabel>}

      <div className="flex items-start gap-2">
        <NumberInput
          aria-label={label ?? "Amount"}
          value={amount}
          onChange={handleChangeAmount}
          onBlur={field.handleBlur}
          isInvalid={isInvalid}
          minValue={0}
          placeholder={placeholder}
          className="flex-1"
          isReadOnly={isReadOnly}
        />

        <Select
          aria-label="Unit"
          value={unit}
          onChange={(key) => handleChangeUnit(String(key) as SizeUnit)}
          className="w-28"
          isReadOnly={isReadOnly}
        >
          <Select.ListBox>
            {SizeUnits.map((candidate) => (
              <Select.Item key={candidate} id={candidate} name={candidate} />
            ))}
          </Select.ListBox>
        </Select>
      </div>

      {(field.state.meta.errors[0] || supportText) && (
        <FieldSupportText>
          {field.state.meta.errors[0] || supportText}
        </FieldSupportText>
      )}
    </div>
  );
}
