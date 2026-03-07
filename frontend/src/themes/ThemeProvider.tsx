import { createContext, useContext, useMemo, type ReactNode } from "react";
import type { Theme } from "../types/theme";
import { bruinLight } from "./bruin";

const ThemeContext = createContext<Theme>(bruinLight);

export function useTheme() {
  return useContext(ThemeContext);
}

export function ThemeProvider({
  theme,
  children,
}: {
  theme: Theme;
  children: ReactNode;
}) {
  const style = useMemo(() => {
    const vars: Record<string, string> = {
      fontFamily: '"Geist", system-ui, -apple-system, sans-serif',
    };
    for (const [key, value] of Object.entries(theme.tokens)) {
      vars[`--dac-${key}`] = value;
    }
    return vars;
  }, [theme]);

  return (
    <ThemeContext.Provider value={theme}>
      <div
        style={style}
        className="dac-root min-h-screen bg-[var(--dac-background)] text-[var(--dac-text-primary)] antialiased"
      >
        {children}
      </div>
    </ThemeContext.Provider>
  );
}
