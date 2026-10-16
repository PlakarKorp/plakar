import { LogoVariant } from "./types";
import { useTheme } from "../../../common/hooks/use-theme";
import LogoFullDark from "../../assets/dark/logo-full.svg";
import LogoIconDark from "../../assets/dark/logo-icon.svg";
import LogoFullLight from "../../assets/light/logo-full.svg";
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
        }[variant]
      }
      className={className}
    />
  );
}
