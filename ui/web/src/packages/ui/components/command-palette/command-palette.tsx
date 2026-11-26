import clsx from "clsx";
import { FocusEvent, KeyboardEvent, PropsWithChildren, ReactNode } from "react";
import {
  Autocomplete,
  Button as RAButton,
  Dialog,
  Header,
  Input,
  Key,
  ListBox,
  ListBoxItem,
  ListBoxItemProps,
  ListBoxSection,
  Modal,
  ModalOverlay,
  SearchField,
  ToggleButtonGroup,
} from "react-aria-components";

import {
  Icon,
  IconButton,
  IconType,
  Spinner,
  TagToggleGroup,
  TextHighlight,
  TextRange,
  Typography,
} from "..";
import { Kbd } from "../kbd/kbd";
import { modifierKeyLabel } from "../kbd/kbd.utils";

type CommandPaletteProps = PropsWithChildren<{
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
  "aria-label": string;
  inputValue: string;
  onInputChange: (value: string) => void;
}>;

function CommandPaletteRoot({
  isOpen,
  onOpenChange,
  "aria-label": ariaLabel,
  inputValue,
  onInputChange,
  children,
}: CommandPaletteProps) {
  return (
    <ModalOverlay
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      isDismissable
      className={clsx(
        "fixed inset-0 z-50 flex items-start justify-center bg-black/40 sm:px-4 sm:pt-[12vh]",
        "data-[entering]:animate-[modal-fade_150ms]",
        "data-[exiting]:animate-[modal-fade_100ms_reverse_ease-in]"
      )}
    >
      <Modal className="h-full w-full sm:h-auto sm:max-w-3xl">
        <Dialog
          aria-label={ariaLabel}
          className={clsx(
            "bg-overlay flex h-full flex-col overflow-hidden outline-none",
            "sm:border-primary sm:max-h-[min(38rem,76vh)] sm:rounded-xl sm:border sm:shadow-2xl"
          )}
        >
          <Autocomplete inputValue={inputValue} onInputChange={onInputChange}>
            {children}
          </Autocomplete>
        </Dialog>
      </Modal>
    </ModalOverlay>
  );
}

type CommandPaletteSearchProps = {
  placeholder: string;
  isLoading?: boolean;
  scope?: { label: string; icon?: IconType };
  onScopeRemove?: () => void;
  onKeyDown?: (event: KeyboardEvent<HTMLInputElement>) => void;
  onClose: () => void;
};

function CommandPaletteSearch({
  placeholder,
  isLoading = false,
  scope,
  onScopeRemove,
  onKeyDown,
  onClose,
}: CommandPaletteSearchProps) {
  return (
    <SearchField
      aria-label={placeholder}
      autoFocus
      className="border-secondary flex h-14 shrink-0 items-center gap-3 border-b px-4"
    >
      <span className="text-tertiary flex size-5 items-center justify-center">
        {isLoading ? (
          <Spinner size="md" />
        ) : (
          <Icon type="magnifying-glass" size="md" />
        )}
      </span>

      {scope && (
        <span className="bg-accent-tertiary text-accent flex shrink-0 items-center gap-1 rounded-md py-0.5 pr-1 pl-2">
          {scope.icon && <Icon type={scope.icon} size="sm" />}
          <Typography as="span" size="sm" weight="medium">
            {scope.label}
          </Typography>
          <RAButton
            aria-label={`Search everything instead of ${scope.label}`}
            onPress={onScopeRemove}
            className="data-hovered:bg-accent-quaternary data-focus-visible:outline-focus flex size-5 cursor-pointer items-center justify-center rounded-sm"
          >
            <Icon type="close" size="xs" />
          </RAButton>
        </span>
      )}

      <Input
        placeholder={placeholder}
        onKeyDown={onKeyDown}
        className="text-primary placeholder:text-placeholder min-w-0 flex-1 bg-transparent text-base outline-none [&::-webkit-search-cancel-button]:hidden"
      />

      <span className="hidden sm:inline-flex">
        <Kbd>Esc</Kbd>
      </span>
      <IconButton
        icon="close"
        variant="tertiary"
        size="sm"
        aria-label="Close search"
        onPress={onClose}
        className="sm:hidden"
      />
    </SearchField>
  );
}

type CommandPaletteScopesProps = PropsWithChildren<{
  "aria-label": string;
  selected: Key;
  onSelectedChange: (key: Key) => void;
}>;

function CommandPaletteScopes({
  "aria-label": ariaLabel,
  selected,
  onSelectedChange,
  children,
}: CommandPaletteScopesProps) {
  return (
    <ToggleButtonGroup
      aria-label={ariaLabel}
      selectionMode="single"
      disallowEmptySelection
      selectedKeys={[selected]}
      onSelectionChange={(keys) => {
        const [key] = keys;
        if (key !== undefined) onSelectedChange(key);
      }}
      className="border-secondary scrollbar-hide flex shrink-0 gap-1.5 overflow-x-auto border-b px-4 py-2"
    >
      {children}
    </ToggleButtonGroup>
  );
}

type CommandPaletteScopeProps = {
  id: Key;
  icon?: IconType;
  children: ReactNode;
};

function CommandPaletteScope({ id, icon, children }: CommandPaletteScopeProps) {
  return (
    <TagToggleGroup.Button
      id={id}
      leftIcon={icon}
      className="shrink-0 px-2.5 py-0.5 whitespace-nowrap"
    >
      {children}
    </TagToggleGroup.Button>
  );
}

function CommandPaletteBody({
  preview,
  children,
}: PropsWithChildren<{ preview?: ReactNode }>) {
  return (
    <div className="flex min-h-0 flex-1">
      {children}
      {preview && (
        <aside
          aria-label="Preview"
          className="border-secondary bg-secondary hidden w-80 shrink-0 overflow-y-auto border-l lg:block"
        >
          {preview}
        </aside>
      )}
    </div>
  );
}

type CommandPaletteListProps = PropsWithChildren<{
  "aria-label": string;
  onAction?: (key: Key) => void;
  onFocusKey?: (key: string) => void;
  renderEmptyState?: () => ReactNode;
}>;

const ITEM_KEY_ATTRIBUTE = "data-palette-key";

function CommandPaletteList({
  "aria-label": ariaLabel,
  onAction,
  onFocusKey,
  renderEmptyState,
  children,
}: CommandPaletteListProps) {
  // Items take a virtual focus, which the input keeps: the events still bubble.
  const handleFocus = (event: FocusEvent<HTMLDivElement>) => {
    if (!(event.target instanceof HTMLElement)) return;
    const key = event.target
      .closest(`[${ITEM_KEY_ATTRIBUTE}]`)
      ?.getAttribute(ITEM_KEY_ATTRIBUTE);
    if (key) onFocusKey?.(key);
  };

  return (
    <div
      className="min-w-0 flex-1 scroll-py-2 overflow-y-auto overscroll-contain p-2"
      onFocus={handleFocus}
    >
      <ListBox
        aria-label={ariaLabel}
        onAction={onAction}
        renderEmptyState={renderEmptyState}
        className="flex flex-col outline-none"
      >
        {children}
      </ListBox>
    </div>
  );
}

type CommandPaletteSectionProps = PropsWithChildren<{
  id: string;
  title: string;
  aside?: ReactNode;
}>;

function CommandPaletteSection({
  id,
  title,
  aside,
  children,
}: CommandPaletteSectionProps) {
  return (
    <ListBoxSection id={id} className="not-first:mt-2">
      <Header className="flex items-center justify-between gap-2 px-2 pt-1 pb-1">
        <Typography
          as="span"
          size="xs"
          weight="medium"
          className="text-tertiary tracking-wide uppercase"
        >
          {title}
        </Typography>
        {aside}
      </Header>
      {children}
    </ListBoxSection>
  );
}

type CommandPaletteItemProps = Omit<
  ListBoxItemProps,
  "id" | "children" | "textValue" | "className"
> & {
  id: string;
  label: string;
  labelRanges?: readonly TextRange[];
  description?: ReactNode;
  prefix?: ReactNode;
  icon?: IconType;
  meta?: ReactNode;
};

function CommandPaletteItem({
  id,
  label,
  labelRanges,
  description,
  prefix,
  icon = "arrow-right",
  meta,
  ...props
}: CommandPaletteItemProps) {
  return (
    <ListBoxItem
      {...props}
      {...{ [ITEM_KEY_ATTRIBUTE]: id }}
      id={id}
      textValue={label}
      className={clsx(
        "group flex cursor-pointer items-center gap-3 rounded-lg px-2 py-1.5 outline-none",
        "data-hovered:bg-secondary data-focused:bg-tertiary",
        "data-disabled:cursor-not-allowed data-disabled:opacity-60"
      )}
    >
      {({ isFocused }) => (
        <>
          <span className="flex shrink-0">
            {prefix ?? (
              <span className="border-secondary bg-primary text-tertiary flex size-8 items-center justify-center rounded-md border">
                <Icon type={icon} size="md" />
              </span>
            )}
          </span>

          <span className="flex min-w-0 flex-1 flex-col">
            <Typography
              as="span"
              size="sm"
              weight="medium"
              className="text-primary [&_mark]:text-accent truncate [&_mark]:bg-transparent [&_mark]:font-semibold"
            >
              <TextHighlight text={label} ranges={labelRanges} />
            </Typography>
            {description && (
              <Typography
                as="span"
                size="xs"
                className="text-tertiary truncate"
              >
                {description}
              </Typography>
            )}
          </span>

          {meta && (
            <span className="hidden shrink-0 items-center gap-2 sm:flex">
              {meta}
            </span>
          )}

          <span
            aria-hidden
            className={clsx(
              "hidden shrink-0 sm:inline-flex",
              !isFocused && "invisible"
            )}
          >
            <Kbd>↵</Kbd>
          </span>
        </>
      )}
    </ListBoxItem>
  );
}

type CommandPaletteEmptyProps = {
  icon?: IconType;
  title: ReactNode;
  description?: ReactNode;
};

function CommandPaletteEmpty({
  icon = "magnifying-glass",
  title,
  description,
}: CommandPaletteEmptyProps) {
  return (
    <div className="flex flex-col items-center gap-2 px-6 py-12 text-center">
      <span className="bg-secondary text-tertiary flex size-10 items-center justify-center rounded-full">
        <Icon type={icon} size="lg" />
      </span>
      <Typography size="sm" weight="medium" className="text-primary">
        {title}
      </Typography>
      {description && (
        <Typography size="xs" className="text-tertiary max-w-sm">
          {description}
        </Typography>
      )}
    </div>
  );
}

function CommandPaletteFooter({ children }: PropsWithChildren) {
  return (
    <div className="border-secondary bg-secondary hidden shrink-0 items-center gap-4 border-t px-4 py-2 sm:flex">
      {children}
    </div>
  );
}

type CommandPaletteHintProps = {
  keys: string[];
  label: string;
};

function CommandPaletteHint({ keys, label }: CommandPaletteHintProps) {
  return (
    <span className="flex items-center gap-1.5">
      <span className="flex items-center gap-0.5">
        {keys.map((key) => (
          <Kbd key={key}>{key}</Kbd>
        ))}
      </span>
      <Typography as="span" size="xs" className="text-tertiary">
        {label}
      </Typography>
    </span>
  );
}

type CommandPaletteTriggerProps = {
  onPress: () => void;
  isCompact?: boolean;
  label?: string;
};

function CommandPaletteTrigger({
  onPress,
  isCompact = false,
  label = "Search",
}: CommandPaletteTriggerProps) {
  const shortcut = `${modifierKeyLabel()} K`;

  if (isCompact) {
    return (
      <IconButton
        icon="magnifying-glass"
        variant="tertiary"
        aria-label={label}
        tooltip={`${label} (${shortcut})`}
        onPress={onPress}
      />
    );
  }

  return (
    <RAButton
      onPress={onPress}
      className={clsx(
        "border-primary bg-primary text-tertiary flex h-9 w-full cursor-pointer items-center gap-2 rounded-lg border pr-1.5 pl-3",
        "data-hovered:bg-primary-hover data-hovered:text-secondary data-focus-visible:outline-focus",
        "duration-150 motion-safe:transition-colors"
      )}
    >
      <Icon type="magnifying-glass" size="sm" />
      <Typography as="span" size="sm" className="flex-1 text-left">
        {label}…
      </Typography>
      <span className="flex items-center gap-0.5">
        <Kbd>{modifierKeyLabel()}</Kbd>
        <Kbd>K</Kbd>
      </span>
    </RAButton>
  );
}

export const CommandPalette = Object.assign(CommandPaletteRoot, {
  Search: CommandPaletteSearch,
  Scopes: CommandPaletteScopes,
  Scope: CommandPaletteScope,
  Body: CommandPaletteBody,
  List: CommandPaletteList,
  Section: CommandPaletteSection,
  Item: CommandPaletteItem,
  Empty: CommandPaletteEmpty,
  Footer: CommandPaletteFooter,
  Hint: CommandPaletteHint,
  Trigger: CommandPaletteTrigger,
});
