import { IconType, IconTypes } from "@plakar-ui/ui/components";

const iconTypeSet = new Set<string>(IconTypes);

/**
 * Whether the design system ships a dedicated icon for this integration.
 */
export function hasIntegrationIcon(name: string | undefined): name is IconType {
  return !!name && iconTypeSet.has(name);
}

export function getIntegrationIcon(name: string | undefined): IconType {
  return hasIntegrationIcon(name) ? name : "puzzle-piece";
}
