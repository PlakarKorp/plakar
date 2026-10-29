import { IconButton, Menu } from "@plakar-ui/ui/components";
import { ComponentProps, useState } from "react";
import { LinkProps, MenuTrigger, Popover } from "react-aria-components";

import { RemoveIntegrationModal } from "./remove-integration-modal";

type IntegrationActionsMenuProps = {
  displayName: string;
  version: string;
  detailHref?: LinkProps["href"];
  latestVersion?: string;
  onUpgrade?: () => void;
  onRemove?: () => void;
  pending?: boolean;
  isDisabled?: boolean;
  variant?: ComponentProps<typeof IconButton>["variant"];
  size?: ComponentProps<typeof IconButton>["size"];
};

export function IntegrationActionsMenu({
  displayName,
  version,
  detailHref,
  latestVersion,
  onUpgrade,
  onRemove,
  pending = false,
  isDisabled = false,
  variant = "tertiary",
  size,
}: IntegrationActionsMenuProps) {
  const [isConfirmingRemove, setIsConfirmingRemove] = useState(false);

  if (!latestVersion && !detailHref && !onRemove) {
    return null;
  }

  return (
    <>
      <MenuTrigger>
        <IconButton
          icon="ellipsis-vertical"
          aria-label={`Actions for ${displayName}`}
          variant={variant}
          size={size}
          className="shrink-0"
        />
        <Popover placement="bottom end" offset={4}>
          <Menu>
            {latestVersion && (
              <Menu.Item
                label={`Update to ${latestVersion}`}
                leftIcon="arrow-up"
                onAction={onUpgrade}
                isDisabled={!onUpgrade || pending || isDisabled}
              />
            )}
            {detailHref && (
              <Menu.Item
                label="Read documentation"
                leftIcon="file-lines"
                href={detailHref}
              />
            )}
            {onRemove && (
              <Menu.Item
                label="Remove"
                leftIcon="trash"
                error
                onAction={() => setIsConfirmingRemove(true)}
                isDisabled={pending || isDisabled}
              />
            )}
          </Menu>
        </Popover>
      </MenuTrigger>
      {onRemove && (
        <RemoveIntegrationModal
          displayName={displayName}
          version={version}
          isOpen={isConfirmingRemove}
          onOpenChange={setIsConfirmingRemove}
          onConfirm={onRemove}
        />
      )}
    </>
  );
}
