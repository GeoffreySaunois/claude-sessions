// Theme cycle: System -> Light -> Dark -> System. "system" = no data-theme
// attribute (CSS media query decides); "light"/"dark" force the palette.
export type ThemeMode = "system" | "light" | "dark";

const LS_THEME = "ccs-theme";
const CYCLE: ThemeMode[] = ["system", "light", "dark"];
export const THEME_META: Record<ThemeMode, { ic: string; label: string }> = {
  system: { ic: "🖥", label: "System" },
  light: { ic: "☀", label: "Light" },
  dark: { ic: "🌙", label: "Dark" },
};

function read(): ThemeMode {
  try {
    const m = localStorage.getItem(LS_THEME);
    if (m === "light" || m === "dark") return m;
  } catch (e) {
    /* ignore */
  }
  return "system";
}

class Theme {
  mode = $state<ThemeMode>(read());

  apply(mode: ThemeMode): void {
    this.mode = mode;
    if (mode === "light" || mode === "dark") {
      document.documentElement.setAttribute("data-theme", mode);
    } else {
      document.documentElement.removeAttribute("data-theme");
    }
    try {
      if (mode === "system") localStorage.removeItem(LS_THEME);
      else localStorage.setItem(LS_THEME, mode);
    } catch (e) {
      /* ignore */
    }
  }

  cycle(): void {
    const i = CYCLE.indexOf(this.mode);
    this.apply(CYCLE[(i + 1) % CYCLE.length]!);
  }

  get meta() {
    return THEME_META[this.mode];
  }
}

export const theme = new Theme();
