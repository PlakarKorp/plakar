import { useEffect, useState } from "react";
import type { PressEvent } from "react-aria-components";

import { IconButton } from "..";
import { copyToClipboard } from "./copy-to-clipboard";

interface CopyButtonProps {
  tooltip: string;
  content: string;
  className?: string;
}

export function CopyButton({ tooltip, content, className }: CopyButtonProps) {
  const [clicked, setClicked] = useState(false);

  const handlePress = async (e: PressEvent) => {
    const success = await copyToClipboard(content, e.target);

    if (success) {
      setClicked(true);
    }
  };

  useEffect(() => {
    if (!clicked) {
      return;
    }

    const timeout = setTimeout(() => {
      setClicked(false);
    }, 1500);

    return () => {
      clearTimeout(timeout);
    };
  }, [clicked]);

  return (
    <IconButton
      aria-label="Copy to clipboard"
      variant="tertiary"
      icon={clicked ? "check" : "copy"}
      tooltip={tooltip}
      onPress={handlePress}
      size="sm"
      className={className}
    />
  );
}
