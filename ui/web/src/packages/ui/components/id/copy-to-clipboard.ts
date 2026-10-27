export async function copyToClipboard(
  text: string,
  anchor?: Element
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

  // Preferring parentElement because modals block any focus outside. Without
  // an anchor still in the page, the focused element stands in for it.
  const near = anchor?.isConnected ? anchor : document.activeElement;
  const container =
    (near !== document.body && near?.parentElement) || document.body;

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
