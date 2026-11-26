export function isApplePlatform() {
  if (typeof navigator === "undefined") return false;
  return /mac|iphone|ipad|ipod/i.test(navigator.userAgent);
}

export function modifierKeyLabel() {
  return isApplePlatform() ? "⌘" : "Ctrl";
}
