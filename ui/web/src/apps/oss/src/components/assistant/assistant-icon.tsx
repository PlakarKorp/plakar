import { IconButton } from "@plakar-ui/ui/components";

type AssistantIconProps = {
  onPress: () => void;
};

export function AssistantIcon({ onPress }: AssistantIconProps) {
  return (
    <IconButton
      aria-label="Open the assistant"
      variant="tertiary"
      icon="wand-magic-sparkles"
      onPress={onPress}
      size="sm"
      tooltip="Assistant"
    />
  );
}
