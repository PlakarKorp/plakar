import clsx from "clsx";
import { PropsWithChildren, ReactNode, useState } from "react";
import {
  Button as AriaButton,
  Disclosure,
  DisclosurePanel,
} from "react-aria-components";

import { Button, Icon, IconButton, TextInput, Typography } from "..";
import { FormElement } from "../../../common/components/form";

function ManagedStringListRoot({ children }: PropsWithChildren) {
  return <div className="flex flex-col gap-3">{children}</div>;
}

type AddFormProps = {
  onAdd: (value: string) => void;
  isAdding?: boolean;
  isDisabled?: boolean;
  placeholder?: string;
  addLabel?: string;
};

function ManagedStringListAddForm({
  onAdd,
  isAdding = false,
  isDisabled = false,
  placeholder = "Add an item…",
  addLabel = "Add",
}: AddFormProps) {
  const [value, setValue] = useState("");
  const trimmed = value.trim();

  const handleAdd = () => {
    if (!trimmed) return;
    setValue("");
    onAdd(trimmed);
  };

  return (
    <FormElement handleSubmit={handleAdd} className="flex gap-2">
      <TextInput
        value={value}
        onChange={setValue}
        placeholder={placeholder}
        aria-label={placeholder}
        isDisabled={isDisabled}
        className="flex-1"
      />
      <Button
        type="submit"
        size="sm"
        variant="primary"
        leftIcon="plus"
        isDisabled={isDisabled || !trimmed}
        isPending={isAdding}
        className="min-w-1/8"
      >
        {addLabel}
      </Button>
    </FormElement>
  );
}

type CollectionProps = PropsWithChildren<{
  isEmpty?: boolean;
  emptyMessage?: string;
}>;

function ManagedStringListCollection({
  children,
  isEmpty = false,
  emptyMessage = "No items yet.",
}: CollectionProps) {
  if (isEmpty) {
    return (
      <Typography size="sm" className="text-tertiary italic">
        {emptyMessage}
      </Typography>
    );
  }

  return <div>{children}</div>;
}

type ItemProps = {
  label: string;
  altText?: ReactNode;
  onRemove: () => Promise<void> | void;
  isDisabled?: boolean;
  expandedContent?: ReactNode;
  confirmRemoveMessage?: string;
};

function ManagedStringListItem({
  label,
  altText,
  onRemove,
  isDisabled = false,
  expandedContent,
  confirmRemoveMessage = "This item is in use. Are you sure you want to delete it?",
}: ItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [isConfirming, setIsConfirming] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

  const isExpandable = !!expandedContent;

  const handleRemove = async () => {
    setIsRemoving(true);
    try {
      await onRemove();
    } finally {
      setIsRemoving(false);
    }
  };

  const handlePressRemove = () => {
    if (isExpandable) {
      setIsExpanded(false);
      setIsConfirming(true);
    } else {
      void handleRemove();
    }
  };

  const handleConfirm = () => {
    setIsConfirming(false);
    void handleRemove();
  };

  const trashButton = (
    <IconButton
      icon="trash"
      variant="error-tertiary"
      size="sm"
      onPress={handlePressRemove}
      isDisabled={isDisabled || isRemoving}
      isPending={isRemoving}
      aria-label={`Remove ${label}`}
      className={clsx(
        isRemoving
          ? "visible"
          : "invisible group-hover/row:visible group-has-[:focus-visible]/row:visible [@media(hover:none)]:visible"
      )}
    />
  );

  if (isExpandable) {
    return (
      <Disclosure
        isExpanded={isExpanded}
        onExpandedChange={setIsExpanded}
        className="group/item border-tertiary data-[expanded]:bg-secondary relative border-b py-2 last:border-b-0 data-[expanded]:rounded-b-md data-[expanded]:border-b-0"
      >
        <div className="group/row flex items-center gap-2">
          <AriaButton
            slot="trigger"
            className="group/trigger ml-2 flex min-w-0 flex-1 cursor-pointer items-center gap-2 text-left outline-offset-4"
          >
            <Icon
              type="chevron-right"
              size="sm"
              className="text-tertiary group-hover/trigger:text-accent-hover group-data-[expanded]/item:text-accent group-data-[expanded]/item:group-hover/trigger:text-accent-hover shrink-0 transition-transform group-data-[expanded]/item:rotate-90"
            />
            <Typography
              size="sm"
              className="text-secondary group-hover/trigger:text-accent-hover group-data-[expanded]/item:text-accent group-data-[expanded]/item:group-hover/trigger:text-accent-hover flex-1"
              weight="medium"
            >
              {label}
            </Typography>
            {altText !== undefined && (
              <Typography
                as="span"
                size="xs"
                className="text-tertiary group-hover/trigger:text-secondary"
              >
                {altText}
              </Typography>
            )}
          </AriaButton>
          {trashButton}
        </div>
        <DisclosurePanel className="mx-2">{expandedContent}</DisclosurePanel>
        {isConfirming && (
          <div className="bg-warning-tertiary absolute inset-0 z-10 flex items-center justify-between gap-2 rounded-md px-2">
            <div className="flex min-w-0 flex-1 items-baseline gap-2">
              <Icon
                type="warning-triangle"
                size="sm"
                className="text-warning shrink-0"
              />
              <Typography
                size="sm"
                weight="medium"
                className="text-warning shrink-0"
              >
                {label}
              </Typography>
              <Typography as="span" size="xs" className="text-warning">
                {confirmRemoveMessage}
              </Typography>
            </div>
            <div className="flex shrink-0 gap-2">
              <Button
                size="sm"
                variant="secondary"
                onPress={() => setIsConfirming(false)}
              >
                Cancel
              </Button>
              <Button size="sm" variant="error" onPress={handleConfirm}>
                Delete
              </Button>
            </div>
          </div>
        )}
      </Disclosure>
    );
  }

  return (
    <div className="group/item border-tertiary relative border-b py-2 last:border-b-0">
      <div className="group/row flex items-center gap-2">
        <div className="ml-2 flex min-w-0 flex-1 items-center gap-2">
          <span className="size-3 shrink-0" />
          <Typography
            size="sm"
            className="text-secondary flex-1"
            weight="medium"
          >
            {label}
          </Typography>
          {altText !== undefined && (
            <Typography as="span" size="xs" className="text-tertiary">
              {altText}
            </Typography>
          )}
        </div>
        {trashButton}
      </div>
    </div>
  );
}

export const ManagedStringList = Object.assign(ManagedStringListRoot, {
  AddForm: ManagedStringListAddForm,
  Collection: ManagedStringListCollection,
  Item: ManagedStringListItem,
});
