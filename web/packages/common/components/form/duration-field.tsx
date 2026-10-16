import {
  NumberInput,
  Radio,
  RadioGroup,
  Select,
} from "@plakar-ui/ui/components";
import clsx from "clsx";
import { useState } from "react";

import { FieldSupportText } from "../../../ui/components/field/field-support-text";
import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";
import { getDurationFromSeconds } from "../../utils/duration";

type Unit = "days" | "hours" | "minutes";
type Option = {
  label: string;
  value: string;
  seconds: number | null;
};

const DEFAULT_OPTIONS = [{ label: "Custom", value: "custom", seconds: null }];

type DurationFieldProps = {
  label?: string;
  options?: Option[];
  type?: "radio" | "select";
  placeholder?: string;
  supportText?: string;
};

export function DurationField({
  label,
  options = DEFAULT_OPTIONS,
  type = "radio",
  placeholder,
  supportText,
}: DurationFieldProps) {
  const field = useFieldContext<number | null>();
  const isReadOnly = useFormReadOnly();

  const [option, setOption] = useState(
    getOptionFromValue(field.state.value, options)
  );
  const isCustom = option?.value === "custom";
  const [custom, setCustom] = useState(
    getDefaultCustom(field.state.value, isCustom)
  );
  const singleOption = options.length === 1;

  const handleChangeOption = (value: string) => {
    if (singleOption) return;

    const selectedOption = options.find((option) => option.value === value)!;

    setOption(selectedOption);

    field.handleChange(
      value ? selectedOption.seconds || getSecondsFromCustom(custom) : null
    );
  };

  const handleChangeCustom = (unit: Unit, value: number) => {
    const newCustom = { ...custom, [unit]: value };
    const seconds = getSecondsFromCustom(newCustom);

    field.handleChange(seconds);
    setCustom(newCustom);
  };

  return (
    <div
      className={clsx("flex flex-col", type === "select" ? "gap-1" : "gap-3")}
    >
      <div
        className={clsx(
          "flex w-full items-start",
          type === "select" ? "gap-4" : "gap-8"
        )}
      >
        {!singleOption &&
          {
            radio: (
              <RadioGroup
                label={label || "Duration"}
                value={option?.value || ""}
                onChange={handleChangeOption}
                className="w-32 shrink-0"
                isDisabled={isReadOnly}
              >
                {options.map(({ value, label }) => (
                  <Radio key={value} value={value}>
                    {label}
                  </Radio>
                ))}
              </RadioGroup>
            ),
            select: (
              <>
                <Select
                  label={label || "Duration"}
                  value={option?.value || ""}
                  placeholder={placeholder}
                  onChange={(key) => handleChangeOption(String(key))}
                  className="w-46"
                  isClearable
                  isReadOnly={isReadOnly}
                >
                  <Select.ListBox>
                    {options.map((option) => (
                      <Select.Item
                        key={option.value}
                        id={option.value}
                        name={option.label}
                      />
                    ))}
                  </Select.ListBox>
                </Select>
                {isCustom && <span className="mt-7.5 hidden md:block">:</span>}
              </>
            ),
          }[type]}
        {isCustom && (
          <div className="flex flex-col gap-4 self-end md:flex-row">
            <NumberInput
              label="Days"
              value={custom.days}
              onChange={(value) => handleChangeCustom("days", value)}
              minValue={0}
              maxValue={365}
              className="max-w-40"
              isReadOnly={isReadOnly}
            />
            <NumberInput
              label="Hours"
              value={custom.hours}
              onChange={(value) => handleChangeCustom("hours", value)}
              minValue={0}
              maxValue={23}
              className="max-w-40"
              isReadOnly={isReadOnly}
            />
            <NumberInput
              label="Minutes"
              value={custom.minutes}
              onChange={(value) => handleChangeCustom("minutes", value)}
              minValue={0}
              maxValue={59}
              className="max-w-40"
              isReadOnly={isReadOnly}
            />
          </div>
        )}
      </div>
      {supportText && (
        <FieldSupportText className="text-tertiary">
          {supportText}
        </FieldSupportText>
      )}
    </div>
  );
}

function getOptionFromValue(seconds: number | null, options: Option[]) {
  if (seconds === null) {
    return undefined;
  }

  return (
    options.find((option) => option.seconds === seconds) ||
    options.find((option) => option.value === "custom")!
  );
}

function getDefaultCustom(seconds: number | null, isCustom: boolean) {
  if (!isCustom || seconds === null) {
    return { days: 0, hours: 0, minutes: 0 };
  }

  return getDurationFromSeconds(seconds);
}

function getSecondsFromCustom({
  days,
  hours,
  minutes,
}: {
  days: number;
  hours: number;
  minutes: number;
}) {
  return ((days * 24 + hours) * 60 + minutes) * 60;
}
