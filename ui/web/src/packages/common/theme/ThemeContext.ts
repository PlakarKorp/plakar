import { createContext } from "react";

export type Scheme = "light" | "dark";
export type Mode = Scheme | "system";

type ThemeContextType = {
  // The effective theme (light or dark)
  value: Scheme;
  // The current selected mode (light, dark, or system)
  mode: Mode;
  setMode: (mode: Mode) => void;
};

export const ThemeContext = createContext<ThemeContextType | undefined>(
  undefined
);
