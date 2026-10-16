import { IconSize } from "./icon.constants";

const ICON_SIZE_CLASS_NAMES: Record<IconSize, string> = {
  xs: "size-2.5",
  sm: "size-3",
  md: "size-4",
  lg: "size-5",
  xl: "size-6",
  "2xl": "size-7",
  "3xl": "size-8",
  "4xl": "size-9",
  "5xl": "size-10",
  "6xl": "size-12",
};

export function getIconSizeClassName(size: IconSize) {
  return ICON_SIZE_CLASS_NAMES[size];
}
