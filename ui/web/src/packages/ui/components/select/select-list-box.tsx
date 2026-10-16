import { ListBox, ListBoxProps } from "react-aria-components";

export function SelectListBox<T extends object>(props: ListBoxProps<T>) {
  return (
    <ListBox
      className="flex max-h-100 flex-col gap-1 overflow-auto p-1 outline-0"
      {...props}
    />
  );
}
