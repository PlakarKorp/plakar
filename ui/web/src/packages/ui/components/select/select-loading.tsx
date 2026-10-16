import clsx from "clsx";
import {
  ListBoxLoadMoreItem,
  ListBoxLoadMoreItemProps,
} from "react-aria-components";

import { Spinner } from "../spinner/spinner";

export function SelectLoading({
  className,
  ...props
}: ListBoxLoadMoreItemProps) {
  return (
    <ListBoxLoadMoreItem
      {...props}
      className={clsx("flex items-center justify-center py-1", className)}
    >
      <Spinner size="lg" />
    </ListBoxLoadMoreItem>
  );
}
