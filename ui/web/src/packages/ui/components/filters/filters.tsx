import { useState } from "react";
import { DialogTrigger } from "react-aria-components";

import { Button, Modal, Select, SelectOption, SelectProps } from "..";
import { MultiSelect } from "../multi-select/multi-select";

type FilterItem = SelectProps & {
  options: SelectOption[];
  isMulti?: boolean;
};

type FiltersProps = {
  items: FilterItem[];
  defaultSelected: Record<string, string[] | null>;
  onApply: (selected: Record<string, string[] | null>) => void;
};

export function Filters({ items, defaultSelected, onApply }: FiltersProps) {
  const [localSelected, setLocalSelected] =
    useState<Record<string, string[] | null>>(defaultSelected);

  const handleChange = (id: string, value: string[] | null) => {
    setLocalSelected((prev) => ({ ...prev, [id]: value }));
  };

  const handleApply = () => {
    onApply(localSelected);
  };

  const handleReset = () => {
    const reset = Object.fromEntries(items.map((item) => [item.id, null]));
    setLocalSelected(reset);
  };

  const handleOpenChange = (isOpen: boolean) => {
    if (isOpen) {
      setLocalSelected(defaultSelected);
    }
  };

  const selectedLength = Object.values(defaultSelected).filter(Boolean).length;
  const localSelectedLength =
    Object.values(localSelected).filter(Boolean).length;
  const description = selectedLength
    ? `${selectedLength} applied ${selectedLength > 1 ? "filters" : "filter"}`
    : "No filters applied";

  return (
    <DialogTrigger onOpenChange={handleOpenChange}>
      <Button
        aria-label="Open filters"
        leftIcon="filter"
        size="md"
        variant="secondary"
      >
        Filter
      </Button>
      <Modal isDismissable aria-label="Filters">
        <Modal.Dialog>
          <Modal.Header title="Filters" description={description} />
          <Modal.Content className="flex flex-col gap-4">
            {items.map(({ options, isMulti, ...item }) => {
              const label = `Filter by ${item.label?.toLowerCase()}`;

              if (isMulti) {
                return (
                  <MultiSelect
                    key={item.id}
                    label={item.label}
                    value={
                      (localSelected[String(item.id)] as string[] | null) || []
                    }
                    onChange={(keys) =>
                      handleChange(
                        String(item.id),
                        keys.length ? keys.map(String) : null
                      )
                    }
                    aria-label={label}
                    placeholder={label}
                    className="w-80"
                  >
                    <MultiSelect.ListBox>
                      {options.map((option) => (
                        <MultiSelect.Item key={option.id} {...option} />
                      ))}
                    </MultiSelect.ListBox>
                  </MultiSelect>
                );
              }

              return (
                <Select
                  key={item.id}
                  {...item}
                  isInvalid={item.isInvalid && !!localSelected[String(item.id)]}
                  aria-label={label}
                  placeholder={label}
                  className="w-80"
                  value={localSelected[String(item.id)]}
                  onChange={(key) =>
                    handleChange(String(item.id), key ? [String(key)] : null)
                  }
                >
                  <Select.ListBox>
                    {options.map((option) => (
                      <Select.Item key={option.id} {...option} />
                    ))}
                  </Select.ListBox>
                </Select>
              );
            })}
          </Modal.Content>
          <Modal.Footer className="flex justify-between">
            <Button
              onPress={handleReset}
              variant="tertiary"
              isDisabled={!localSelectedLength}
            >
              Reset filters
            </Button>
            <Button onPress={handleApply} slot="close">
              Apply
            </Button>
          </Modal.Footer>
        </Modal.Dialog>
      </Modal>
    </DialogTrigger>
  );
}
