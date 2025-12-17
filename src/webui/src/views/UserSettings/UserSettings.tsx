import type { Component } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Icon } from "../../components/Icon";
import {
  Summary,
  SummaryNav,
  SummaryCategory,
  SummaryNavItem,
  SummaryContent,
  SummarySection,
  SummaryItem,
  SummaryToggle,
  SummarySelect,
  SummaryButton,
} from "../../components/Summary";
import { ServerSettingsPanel } from "../../components/ServerSettingsPanel";
import { themeService } from "../../services/themeService";
import { getServerUrl } from "../../services/storageService";
import {
  getServerSettings,
  updateServerSetting,
  isRootUser,
  type ServerSetting,
} from "../../services/settingsService";

type ThemePreference = "system" | "light" | "dark";

interface UserSettingsProps {
  onBack?: () => void;
}

export const UserSettings: Component<UserSettingsProps> = (props) => {
  const [themePreference, setThemePreference] =
    createSignal<ThemePreference>("system");
  const [serverUrl, setServerUrl] = createSignal("");
  const [useSystemPrompts, setUseSystemPrompts] = createSignal(true);
  const [redactPrivateValues, setRedactPrivateValues] = createSignal(false);

  // Server settings state (for root users)
  const [serverSettings, setServerSettings] = createSignal<ServerSetting[]>([]);
  const [settingsLoading, setSettingsLoading] = createSignal(false);
  const [settingsError, setSettingsError] = createSignal<string | null>(null);
  const [updatingSettings, setUpdatingSettings] = createSignal<Set<string>>(
    new Set(),
  );

  const loadServerSettings = async () => {
    if (!isRootUser()) return;

    setSettingsLoading(true);
    setSettingsError(null);

    const result = await getServerSettings();
    if (result.success) {
      setServerSettings(result.settings);
    } else {
      setSettingsError(result.message);
    }

    setSettingsLoading(false);
  };

  const handleSettingUpdate = async (
    key: string,
    value: string | number | boolean,
  ) => {
    setUpdatingSettings((prev) => new Set(prev).add(key));

    const result = await updateServerSetting(key, value);
    if (result.success) {
      // Update local state
      setServerSettings((prev) =>
        prev.map((s) => (s.key === key ? { ...s, value: result.value } : s)),
      );
      // Could show a toast notification here if rebootRequired
    } else {
      setSettingsError(result.message);
    }

    setUpdatingSettings((prev) => {
      const next = new Set(prev);
      next.delete(key);
      return next;
    });
  };

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

    // Load server settings if user is root
    loadServerSettings();
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
    <section class="h-full flex flex-col">
      {/* Header */}
      <header class="shrink-0 px-6 py-4 border-b border-border">
        <button
          onClick={props.onBack}
          class="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors mb-2"
        >
          <Icon name="arrow-left" size="sm" />
          <span>Back</span>
        </button>
        <h1 class="text-2xl font-bold">Settings</h1>
      </header>

      {/* Summary Layout */}
      <Summary
        defaultSection="general"
        defaultExpanded={["general", "appearance"]}
        class="flex-1"
      >
        <SummaryNav>
          <SummaryCategory id="general" label="General" icon="gear">
            <SummaryNavItem id="general" label="General" />
            <SummaryNavItem id="privacy" label="Privacy" />
          </SummaryCategory>
          <SummaryCategory id="appearance" label="Appearance" icon="palette">
            <SummaryNavItem id="theme" label="Theme" />
          </SummaryCategory>
          <SummaryCategory id="server" label="Server" icon="plugs-connected">
            <SummaryNavItem id="server-connection" label="Connection" />
            <Show when={isRootUser()}>
              <SummaryNavItem id="server-settings" label="Settings" />
            </Show>
          </SummaryCategory>
          <SummaryCategory id="account" label="Account" icon="user">
            <SummaryNavItem id="profile" label="Profile" />
            <SummaryNavItem id="security" label="Security" />
          </SummaryCategory>
        </SummaryNav>

        <SummaryContent>
          {/* General Section */}
          <SummarySection
            id="general"
            title="General"
            description="General application settings"
          >
            <SummaryItem
              title="Use System Prompts"
              description="Use native OS dialogs for confirmations."
            >
              <SummaryToggle
                checked={useSystemPrompts()}
                onChange={setUseSystemPrompts}
              />
            </SummaryItem>
            <SummaryItem
              title="Language"
              description="Select your preferred language."
            >
              <SummarySelect
                value="en"
                options={[
                  { value: "en", label: "English" },
                  { value: "fr", label: "Français" },
                  { value: "de", label: "Deutsch" },
                ]}
                onChange={() => {}}
              />
            </SummaryItem>
          </SummarySection>

          {/* Privacy Section */}
          <SummarySection
            id="privacy"
            title="Privacy"
            description="Privacy and data settings"
          >
            <SummaryItem
              title="Redact Private Values"
              description="Hide the values of sensitive variables."
            >
              <SummaryToggle
                checked={redactPrivateValues()}
                onChange={setRedactPrivateValues}
              />
            </SummaryItem>
            <SummaryItem
              title="Private Files"
              description="Configure which files are considered private."
            >
              <SummaryButton onClick={() => {}}>Edit settings</SummaryButton>
            </SummaryItem>
          </SummarySection>

          {/* Theme Section */}
          <SummarySection
            id="theme"
            title="Theme"
            description="Customize the application appearance"
          >
            <SummaryItem
              title="Color Scheme"
              description="Choose how the application looks."
            >
              <SummarySelect
                value={themePreference()}
                options={[
                  { value: "system", label: "System" },
                  { value: "light", label: "Light" },
                  { value: "dark", label: "Dark" },
                ]}
                onChange={(value) =>
                  handleThemeChange(value as ThemePreference)
                }
              />
            </SummaryItem>
          </SummarySection>

          {/* Server Connection Section */}
          <SummarySection
            id="server-connection"
            title="Connection"
            description="Server connection settings"
          >
            <SummaryItem
              title="Connected Server"
              description={serverUrl() || "No server connected"}
            >
              <SummaryButton onClick={() => {}} disabled>
                Change server
              </SummaryButton>
            </SummaryItem>
            <SummaryItem
              title="Connection Status"
              description="Check the current connection status."
            >
              <SummaryButton onClick={() => {}}>Test connection</SummaryButton>
            </SummaryItem>
          </SummarySection>

          {/* Server Settings Section (Root users only) */}
          <SummarySection
            id="server-settings"
            title="Server Settings"
            description="Configure server runtime settings (requires root access)"
          >
            <ServerSettingsPanel
              settings={serverSettings()}
              loading={settingsLoading()}
              error={settingsError()}
              updatingKeys={updatingSettings()}
              onUpdate={handleSettingUpdate}
              onRetry={loadServerSettings}
            />
          </SummarySection>

          {/* Profile Section */}
          <SummarySection
            id="profile"
            title="Profile"
            description="Manage your profile information"
          >
            <SummaryItem
              title="Display Name"
              description="Your public display name."
            >
              <SummaryButton onClick={() => {}}>Edit profile</SummaryButton>
            </SummaryItem>
            <SummaryItem title="Avatar" description="Your profile picture.">
              <SummaryButton onClick={() => {}}>Change avatar</SummaryButton>
            </SummaryItem>
          </SummarySection>

          {/* Security Section */}
          <SummarySection
            id="security"
            title="Security"
            description="Account security settings"
          >
            <SummaryItem
              title="Change Password"
              description="Update your account password."
            >
              <SummaryButton onClick={() => {}}>Change password</SummaryButton>
            </SummaryItem>
            <SummaryItem
              title="Active Sessions"
              description="Manage your active login sessions."
            >
              <SummaryButton onClick={() => {}}>View sessions</SummaryButton>
            </SummaryItem>
            <SummaryItem
              title="Delete Account"
              description="Permanently delete your account and all data."
            >
              <SummaryButton
                onClick={() => {}}
                class="text-destructive border-destructive hover:bg-destructive/10"
              >
                Delete account
              </SummaryButton>
            </SummaryItem>
          </SummarySection>
        </SummaryContent>
      </Summary>
    </section>
  );
};
