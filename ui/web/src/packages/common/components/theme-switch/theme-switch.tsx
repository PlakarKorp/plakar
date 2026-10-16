import {
  Icon,
  SegmentedControl,
  SegmentedControlItem,
  Typography,
} from "@plakar-ui/ui/components";

import { useTheme } from "../../hooks";
import { Mode } from "../../theme";

const THEME_ITEMS: SegmentedControlItem[] = [
  { id: "light", label: "Light theme", icon: "sun" },
  { id: "system", label: "System theme", icon: "circle-half-stroke" },
  { id: "dark", label: "Dark theme", icon: "moon" },
];

type ThemeSwitchProps = {
  isCompact?: boolean;
};

export function ThemeSwitch({ isCompact = false }: ThemeSwitchProps) {
  const { mode, setMode } = useTheme();

  if (isCompact) {
    return (
      <div className="text-secondary flex h-9 items-center justify-center px-3">
        <Icon type="palette" size="md" aria-label="Theme" />
      </div>
    );
  }

  return (
    <div className="text-secondary flex h-9 items-center justify-between gap-3 px-3">
      <div className="flex items-center gap-3">
        <Icon type="palette" size="md" />
        <Typography as="span" size="sm" weight="medium">
          Theme
        </Typography>
      </div>
      <SegmentedControl
        ariaLabel="Theme"
        items={THEME_ITEMS}
        selectedId={mode}
        onSelect={(id) => setMode(id as Mode)}
        hideLabels
      />
    </div>
  );
}
