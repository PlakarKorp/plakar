import { SearchField } from "react-aria-components";

import { FieldButton } from "./field-button";
import { Input } from "../text-input/text-input";

type FieldSearchProps = {
  /** Accessible name of the search input. */
  label?: string;
  placeholder?: string;
};

/**
 * Search input used at the top of the listbox popovers (Select, MultiSelect,
 * Tab.Dropdown). Expects to be rendered inside a React Aria `Autocomplete`.
 */
export function FieldSearch({
  label = "Search",
  placeholder,
}: FieldSearchProps) {
  return (
    <SearchField aria-label={label} autoFocus className="group">
      <Input
        icon="magnifying-glass"
        placeholder={placeholder}
        suffix={
          <FieldButton
            slot="close"
            icon="close"
            aria-hidden
            className="-mr-2 group-empty:invisible"
          />
        }
        className="outline-focus m-1 outline-2"
      />
    </SearchField>
  );
}
