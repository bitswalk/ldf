import type { Component } from "solid-js";
import { createSignal, onMount } from "solid-js";
import { Icon } from "../../components/Icon";
import { themeService } from "../../services/themeService";
import { getServerUrl } from "../../services/storageService";

type ThemePreference = "system" | "light" | "dark";

interface UserSettingsProps {
  onBack?: () => void;
}

export const UserSettings: Component<UserSettingsProps> = (props) => {
  const [themePreference, setThemePreference] =
    createSignal<ThemePreference>("system");
  const [serverUrl, setServerUrl] = createSignal("");

  onMount(() => {
    // Load current theme preference
    const autoMode = localStorage.getItem("theme-auto");
    if (autoMode === "true" || autoMode === null) {
      setThemePreference("system");
    } else {
      const savedMode = localStorage.getItem("theme-mode");
      setThemePreference(savedMode === "1" ? "light" : "dark");
    }

    // Load server URL
    const url = getServerUrl();
    if (url) {
      setServerUrl(url);
    }
  });

  const handleThemeChange = (preference: ThemePreference) => {
    setThemePreference(preference);

    switch (preference) {
      case "system":
        themeService.enableAutoMode();
        break;
      case "light":
        themeService.disableAutoMode(true);
        break;
      case "dark":
        themeService.disableAutoMode(false);
        break;
    }
  };

  return (
    <section class="h-full flex flex-col items-center justify-start p-8 overflow-y-auto">
      <header class="w-full max-w-2xl mb-8">
        <button
          onClick={props.onBack}
          class="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors mb-4"
        >
          <Icon name="arrow-left" size="sm" />
          <span>Back</span>
        </button>
        <h1 class="text-4xl font-bold">Settings</h1>
        <p class="text-muted-foreground mt-2">
          Manage your preferences and application settings
        </p>
      </header>

      <section class="w-full max-w-2xl flex flex-col gap-8">
        {/* Theme Settings */}
        <article class="border border-border rounded-lg p-6">
          <header class="flex items-center gap-3 mb-4">
            <Icon name="palette" size="lg" class="text-primary" />
            <h2 class="text-xl font-semibold">Appearance</h2>
          </header>

          <fieldset class="flex flex-col gap-2">
            <legend class="text-sm text-muted-foreground mb-3">
              Choose how the application looks
            </legend>

            <label class="flex items-center gap-3 p-3 border border-border rounded-md cursor-pointer hover:bg-muted/50 transition-colors has-[:checked]:border-primary has-[:checked]:bg-primary/5">
              <input
                type="radio"
                name="theme"
                value="system"
                checked={themePreference() === "system"}
                onChange={() => handleThemeChange("system")}
                class="w-4 h-4 accent-primary"
              />
              <Icon name="desktop" size="md" class="text-muted-foreground" />
              <section class="flex flex-col">
                <span class="font-medium">System</span>
                <span class="text-sm text-muted-foreground">
                  Automatically switch based on time of day
                </span>
              </section>
            </label>

            <label class="flex items-center gap-3 p-3 border border-border rounded-md cursor-pointer hover:bg-muted/50 transition-colors has-[:checked]:border-primary has-[:checked]:bg-primary/5">
              <input
                type="radio"
                name="theme"
                value="light"
                checked={themePreference() === "light"}
                onChange={() => handleThemeChange("light")}
                class="w-4 h-4 accent-primary"
              />
              <Icon name="sun" size="md" class="text-muted-foreground" />
              <section class="flex flex-col">
                <span class="font-medium">Light</span>
                <span class="text-sm text-muted-foreground">
                  Always use light theme
                </span>
              </section>
            </label>

            <label class="flex items-center gap-3 p-3 border border-border rounded-md cursor-pointer hover:bg-muted/50 transition-colors has-[:checked]:border-primary has-[:checked]:bg-primary/5">
              <input
                type="radio"
                name="theme"
                value="dark"
                checked={themePreference() === "dark"}
                onChange={() => handleThemeChange("dark")}
                class="w-4 h-4 accent-primary"
              />
              <Icon name="moon" size="md" class="text-muted-foreground" />
              <section class="flex flex-col">
                <span class="font-medium">Dark</span>
                <span class="text-sm text-muted-foreground">
                  Always use dark theme
                </span>
              </section>
            </label>
          </fieldset>
        </article>

        {/* Server Connection */}
        <article class="border border-border rounded-lg p-6">
          <header class="flex items-center gap-3 mb-4">
            <Icon name="plugs-connected" size="lg" class="text-primary" />
            <h2 class="text-xl font-semibold">Server Connection</h2>
          </header>

          <section class="flex flex-col gap-3">
            <label class="flex flex-col gap-1">
              <span class="text-sm text-muted-foreground">
                Connected Server URL
              </span>
              <input
                type="url"
                value={serverUrl()}
                disabled
                class="px-4 py-2 border border-border rounded-md bg-muted text-muted-foreground cursor-not-allowed"
              />
            </label>
            <p class="text-sm text-muted-foreground">
              To change the server, you need to logout and reconnect.
            </p>
          </section>
        </article>
      </section>
    </section>
  );
};
