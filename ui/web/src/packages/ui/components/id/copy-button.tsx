import { useEffect, useState } from "react";
import type { PressEvent } from "react-aria-components";

import { IconButton } from "..";

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

async function copyToClipboard(
  text: string,
  anchor: Element
): Promise<boolean> {
  if (window.isSecureContext && !!navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fall back to legacy copy method.
      // This can fail if the page is not served over HTTPS,
      // or if the user denies clipboard permissions.
    }
  }

  // Preferring parentElement because modals block any focus outside
  const container = anchor.parentElement ?? document.body;

  // Saving focus so that keyboard users don't lose it
  const previouslyFocused =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "");

  textarea.style.position = "fixed";
  textarea.style.top = "0";
  textarea.style.left = "0";
  textarea.style.width = "2em";
  textarea.style.height = "2em";
  textarea.style.padding = "0";
  textarea.style.border = "0";
  textarea.style.outline = "0";
  textarea.style.boxShadow = "none";
  textarea.style.opacity = "0";

  container.appendChild(textarea);

  textarea.focus();
  textarea.select();

  try {
    return document.execCommand("copy");
  } catch {
    return false;
  } finally {
    container.removeChild(textarea);

    if (previouslyFocused?.isConnected) {
      previouslyFocused.focus();
    }
  }
}
