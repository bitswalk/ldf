import { createSignal, createEffect, createRoot } from "solid-js";
import type { Theme } from "../themes/default";
import { timeService } from "./time";

// ThemeMode: false = dark/night (0), true = light/day (1)
export type ThemeMode = boolean;

class ThemeService {
  private currentTheme: ReturnType<typeof createSignal<Theme | null>>;
  private mode: ReturnType<typeof createSignal<ThemeMode>>;
  private disposeRoot?: () => void;

  constructor() {
    // Wrap signal creation in createRoot to avoid disposal warnings
    this.disposeRoot = createRoot((dispose) => {
      this.currentTheme = createSignal<Theme | null>(null);
      this.mode = createSignal<ThemeMode>(true); // Default to light (1)
      return dispose;
    });
  }

  getCurrentTheme = () => this.currentTheme[0]();
  getMode = () => this.mode[0]();

  // Helper to check if dark mode
  isDark = () => !this.mode[0]();
  isLight = () => this.mode[0]();

  async loadTheme(theme: Theme): Promise<void> {
    this.currentTheme[1](theme);
  }

  setThemeMode(mode: ThemeMode): void {
    this.mode[1](mode);
    localStorage.setItem("theme-mode", mode ? "1" : "0");

    if (!mode) {
      // Dark mode (false/0)
      document.documentElement.classList.add("dark");
    } else {
      // Light mode (true/1)
      document.documentElement.classList.remove("dark");
    }
  }

  toggleThemeMode(): void {
    const newMode = !this.mode[0]();
    this.setThemeMode(newMode);
  }

  initialize(): void {
    const savedModeStr = localStorage.getItem("theme-mode");
    const autoModeStr = localStorage.getItem("theme-auto");

    // Default to auto mode if not explicitly set
    const autoMode = autoModeStr === null ? true : autoModeStr === "true";

    if (savedModeStr !== null && !autoMode) {
      // User has manually set a preference
      const savedMode = savedModeStr === "1"; // "1" = light (true), "0" = dark (false)
      this.setThemeMode(savedMode);
    } else {
      // Use automatic time-based switching
      if (autoModeStr === null) {
        // First time, set the flag
        localStorage.setItem("theme-auto", "true");
      }

      timeService.initialize();

      // Set initial theme based on time of day
      const initialMode = timeService.getTimeOfDay() === "day"; // day = true (light), night = false (dark)
      this.setThemeMode(initialMode);

      // Subscribe to time of day changes
      createRoot(() => {
        timeService.onTimeOfDayChange((timeOfDay) => {
          const newMode = timeOfDay === "day"; // day = true (light), night = false (dark)
          this.setThemeMode(newMode);
        });
      });
    }
  }

  enableAutoMode(): void {
    localStorage.setItem("theme-auto", "true");
    localStorage.removeItem("theme-mode");
    this.initialize();
  }

  disableAutoMode(preferredMode: ThemeMode): void {
    localStorage.setItem("theme-auto", "false");
    localStorage.setItem("theme-mode", preferredMode ? "1" : "0");
    this.setThemeMode(preferredMode);
  }
}

export const themeService = new ThemeService();
