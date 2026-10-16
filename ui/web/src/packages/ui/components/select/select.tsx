import clsx from "clsx";
import {
  ComponentProps,
  createContext,
  PropsWithChildren,
  ReactNode,
  useContext,
} from "react";
import {
  Autocomplete,
  AutocompleteProps,
  Button,
  Popover,
  Select as RASelect,
  SelectValue,
  Text,
} from "react-aria-components";

import { Icon, IconType, Spinner, Typography } from "..";
import { SelectListBoxItem } from "./select-item";
import { SelectListBox } from "./select-list-box";
import { SelectLoading } from "./select-loading";
import { SelectSection } from "./select-section";
import { FieldButton } from "../field/field-button";
import { FieldLabel } from "../field/field-label";
import { FieldSearch } from "../field/field-search";
import { FieldSupportText } from "../field/field-support-text";

export type SelectOption = {
  id: string;
  name: string;
  onPress?: () => void;
} & (
  | {
      icon?: IconType;
      item?: never;
    }
  | {
      icon?: never;
      item: ReactNode;
    }
);
export type SelectVariant = "default" | "terminal" | "nav" | "heading";

function listVariantOf(variant: SelectVariant) {
  return variant === "terminal" ? "terminal" : "default";
}

const Context = createContext<{
  isDisabled?: boolean;
  variant: SelectVariant;
}>({
  variant: "default",
});

export type SelectProps = PropsWithChildren<
  ComponentProps<typeof RASelect> & {
    icon?: IconType;
    label?: string;
    supportText?: ReactNode;
    variant?: SelectVariant;
    isReadOnly?: boolean;
    isClearable?: boolean;
    selectedFallbackOption?: SelectOption;
    isPending?: boolean;
    SelectValuePrefix?: ReactNode;
    triggerDescription?: ReactNode;
    popoverFooter?: ReactNode;
    triggerClassName?: string;
    beforeLabelAction?: React.ReactNode;
    labelAction?: React.ReactNode;
  }
>;

function SelectRoot({
  label,
  supportText,
  isClearable,
  className,
  triggerClassName,
  variant = "default",
  children,
  selectedFallbackOption,
  isPending = false,
  SelectValuePrefix,
  triggerDescription,
  popoverFooter,
  beforeLabelAction,
  labelAction,
  ...props
}: SelectProps) {
  const hasClearButton = isClearable && !props.isReadOnly && props.value;

  return (
    <Context.Provider value={{ isDisabled: props.isDisabled, variant }}>
      <RASelect
        {...props}
        className={clsx(
          "group flex flex-col gap-1",
          variant === "heading" && "max-w-full min-w-0",
          className
        )}
        isDisabled={props.isDisabled}
        aria-label={
          label || props["aria-labelledby"]
            ? undefined
            : (props.placeholder ?? "Select an item")
        }
        placeholder={props.isReadOnly ? "" : props.placeholder}
      >
        {label && (
          <FieldLabel
            isRequired={props.isRequired}
            beforeAction={beforeLabelAction}
            action={labelAction}
          >
            {label}
          </FieldLabel>
        )}

        <div
          className={clsx(
            "relative flex flex-1",
            variant === "heading" ? "min-w-0" : "min-w-fit",
            triggerClassName
          )}
        >
          <Button
            className={clsx(
              "flex flex-1 items-center justify-between gap-2 rounded-lg outline-0 -outline-offset-2",
              {
                default: clsx(
                  "px-3 py-2",
                  "border-primary bg-primary text-tertiary hover:bg-primary-hover border",
                  "group-data-invalid:border-error",
                  "data-focus-visible:outline-focus data-pressed:border-accent-heavy data-pressed:text-primary data-focus-visible:outline-2",
                  props.isReadOnly
                    ? "bg-tertiary text-primary border-tertiary hover:bg-tertiary"
                    : "data-disabled:border-tertiary data-disabled:bg-disable data-disabled:text-disable data-disabled:cursor-not-allowed",
                  "group-data-invalid:data-focus:border-error group-data-invalid:data-pressed:border-error"
                ),
                terminal: "text-terminal-secondary justify-end px-3 py-2",
                nav: "border-secondary bg-secondary text-primary hover:bg-tertiary data-pressed:bg-tertiary data-focus-visible:outline-focus data-disabled:hover:bg-secondary border py-1.5 pr-3 pl-[7px] data-disabled:cursor-default data-focus-visible:outline-2",
                heading:
                  "text-primary border-secondary bg-primary hover:border-primary hover:bg-primary-hover group-data-open:border-accent data-focus-visible:outline-focus min-w-0 cursor-pointer gap-4 rounded-xl border p-2.5 text-left transition-colors data-focus-visible:outline-2",
              }[variant]
            )}
            isDisabled={props.isReadOnly || props.isDisabled}
          >
            {variant === "heading" ? (
              <>
                <span className="flex min-w-0 items-center gap-3">
                  {SelectValuePrefix}
                  <span className="flex min-w-0 flex-col items-start gap-0.5">
                    <Typography
                      as="span"
                      variant="heading"
                      size="md"
                      weight="semibold"
                      className="block max-w-full truncate"
                    >
                      <SelectValue>
                        {({ selectedText }) =>
                          selectedText || selectedFallbackOption?.item
                        }
                      </SelectValue>
                    </Typography>
                    {triggerDescription && (
                      <Text
                        slot="description"
                        elementType="span"
                        className="flex"
                      >
                        <Typography
                          as="span"
                          size="sm"
                          className="text-secondary"
                        >
                          {triggerDescription}
                        </Typography>
                      </Text>
                    )}
                  </span>
                </span>
                <span
                  aria-hidden="true"
                  className="text-secondary flex size-9 shrink-0 items-center justify-center"
                >
                  {isPending ? (
                    <Spinner size="md" inheritColor />
                  ) : (
                    <Icon type="sort" size="md" />
                  )}
                </span>
              </>
            ) : (
              <>
                <Typography
                  as="span"
                  variant="body"
                  size="sm"
                  weight={variant === "nav" ? "semibold" : undefined}
                  className="flex min-w-0 flex-1 items-center gap-3"
                >
                  {SelectValuePrefix}
                  <SelectValue className="grid w-full grid-cols-1 text-left">
                    {({ defaultChildren, selectedText }) =>
                      selectedText
                        ? defaultChildren
                        : selectedFallbackOption?.item
                    }
                  </SelectValue>
                </Typography>

                <span
                  aria-hidden="true"
                  className={clsx(
                    "my-0.5 flex items-center",
                    (props.isReadOnly ||
                      (variant === "nav" && props.isDisabled)) &&
                      "invisible",
                    variant === "nav" && "text-quaternary",
                    isClearable && "ml-7"
                  )}
                >
                  {isPending ? (
                    <Spinner size="md" inheritColor />
                  ) : (
                    <Icon type="caret-down" size="md" />
                  )}
                </span>
              </>
            )}
          </Button>

          {hasClearButton && (
            <div className="absolute top-0 right-7 flex h-full items-center p-0.5">
              <FieldButton
                slot={null}
                icon="close"
                aria-label="Clear selection"
                onPress={() => props.onChange?.("")}
              />
            </div>
          )}
        </div>

        {supportText && <FieldSupportText>{supportText}</FieldSupportText>}

        <Popover
          offset={4}
          containerPadding={8}
          className={clsx(
            "flex flex-col",
            "rounded-lg border shadow-lg",
            variant === "heading"
              ? "w-max max-w-[min(28rem,calc(100vw-1rem))] min-w-[var(--trigger-width)]"
              : "w-[var(--trigger-width)]",
            {
              default: "border-primary bg-primary",
              terminal: "border-terminal bg-terminal",
            }[listVariantOf(variant)]
          )}
        >
          {children}
          {popoverFooter && (
            <div className="border-secondary bg-secondary flex flex-wrap items-center justify-end gap-2 rounded-b-lg border-t py-2 pr-2 pl-3">
              {popoverFooter}
            </div>
          )}
        </Popover>
      </RASelect>
    </Context.Provider>
  );
}

function SelectItem(props: SelectOption) {
  const { isDisabled, variant } = useContext(Context);

  return (
    <SelectListBoxItem
      {...props}
      isDisabled={isDisabled}
      variant={listVariantOf(variant)}
    />
  );
}

type SelectSearchableListBoxProps = AutocompleteProps;

function SelectSearchableListBox({
  children,
  ...props
}: SelectSearchableListBoxProps) {
  return (
    <Autocomplete {...props}>
      <FieldSearch />
      <SelectListBox
        renderEmptyState={() => (
          <Typography
            as="div"
            variant="body"
            size="sm"
            className="text-tertiary p-2 italic"
          >
            No results found
          </Typography>
        )}
      >
        {children}
      </SelectListBox>
    </Autocomplete>
  );
}

export const Select = Object.assign(SelectRoot, {
  Item: SelectItem,
  ListBox: SelectListBox,
  SearchableListBox: SelectSearchableListBox,
  Loading: SelectLoading,
  Section: SelectSection,
});
