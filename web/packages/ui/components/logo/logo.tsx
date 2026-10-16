import { LogoVariant } from "./types";
import { useTheme } from "../../../common/hooks/use-theme";
import LogoFullControlPlaneDark from "../../assets/dark/logo-full-control-plane.svg";
import LogoFullDark from "../../assets/dark/logo-full.svg";
import LogoIconControlPlaneDark from "../../assets/dark/logo-icon-control-plane.svg";
import LogoIconDark from "../../assets/dark/logo-icon.svg";
import LogoFullControlPlaneLight from "../../assets/light/logo-full-control-plane.svg";
import LogoFullLight from "../../assets/light/logo-full.svg";
import LogoIconControlPlaneLight from "../../assets/light/logo-icon-control-plane.svg";
import LogoIconLight from "../../assets/light/logo-icon.svg";

type LogoProps = {
  variant?: LogoVariant;
  className?: string;
};

export function Logo({ variant = "full", className }: LogoProps) {
  const theme = useTheme();

  const isDark = theme.value === "dark";

  return (
    <img
      alt="plakar.io"
      src={
        {
          full: isDark ? LogoFullDark : LogoFullLight,
          icon: isDark ? LogoIconDark : LogoIconLight,
          "full-control-plane": isDark
            ? LogoFullControlPlaneDark
            : LogoFullControlPlaneLight,
          "icon-control-plane": isDark
            ? LogoIconControlPlaneDark
            : LogoIconControlPlaneLight,
        }[variant]
      }
      className={className}
    />
  );
}
