import type { Component } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
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
import { Modal } from "../../components/Modal";
import { SourceForm } from "../../components/SourceForm";
import { Spinner } from "../../components/Spinner";
import { themeService } from "../../services/theme";
import { getServerUrl } from "../../services/storage";
import {
  getServerSettings,
  updateServerSetting,
  isRootUser,
  setDevMode,
  type ServerSetting,
} from "../../services/settings";
import {
  listDefaultSources,
  createDefaultSource,
  updateDefaultSource,
  deleteDefaultSource,
  type SourceDefault,
  type CreateSourceRequest,
  type UpdateSourceRequest,
} from "../../services/sources";
import { isDevMode } from "../../lib/utils";

type ThemePreference = "system" | "light" | "dark";

interface SettingsProps {
  onBack?: () => void;
}

export const Settings: Component<SettingsProps> = (props) => {
  const [themePreference, setThemePreference] =
    createSignal<ThemePreference>("system");
  const [serverUrl, setServerUrl] = createSignal("");
  const [useSystemPrompts, setUseSystemPrompts] = createSignal(true);
  const [redactPrivateValues, setRedactPrivateValues] = createSignal(false);
  const [devModeEnabled, setDevModeEnabled] = createSignal(false);
  const [devModeUpdating, setDevModeUpdating] = createSignal(false);

  // Server settings state (for root users)
  const [serverSettings, setServerSettings] = createSignal<ServerSetting[]>([]);
  const [settingsLoading, setSettingsLoading] = createSignal(false);
  const [settingsError, setSettingsError] = createSignal<string | null>(null);
  const [updatingSettings, setUpdatingSettings] = createSignal<Set<string>>(
    new Set(),
  );

  // Default sources state (for root users)
  const [defaultSources, setDefaultSources] = createSignal<SourceDefault[]>([]);
  const [sourcesLoading, setSourcesLoading] = createSignal(false);
  const [sourcesError, setSourcesError] = createSignal<string | null>(null);
  const [sourceModalOpen, setSourceModalOpen] = createSignal(false);
  const [editingSource, setEditingSource] = createSignal<SourceDefault | null>(
    null,
  );
  const [sourceSubmitting, setSourceSubmitting] = createSignal(false);
  const [deleteSourceModalOpen, setDeleteSourceModalOpen] = createSignal(false);
  const [sourceToDelete, setSourceToDelete] =
    createSignal<SourceDefault | null>(null);
  const [deletingSource, setDeletingSource] = createSignal(false);

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

  const handleDevModeToggle = async (enabled: boolean) => {
    setDevModeUpdating(true);
    const result = await setDevMode(enabled);
    if (result.success) {
      setDevModeEnabled(enabled);
    }
    setDevModeUpdating(false);
  };

  const loadDefaultSources = async () => {
    if (!isRootUser()) return;

    setSourcesLoading(true);
    setSourcesError(null);

    const result = await listDefaultSources();
    if (result.success) {
      setDefaultSources(result.sources);
    } else {
      setSourcesError(result.message);
    }

    setSourcesLoading(false);
  };

  const handleAddSource = () => {
    setEditingSource(null);
    setSourceModalOpen(true);
  };

  const handleEditSource = (source: SourceDefault) => {
    setEditingSource(source);
    setSourceModalOpen(true);
  };

  const handleSourceFormSubmit = async (formData: CreateSourceRequest) => {
    setSourceSubmitting(true);
    setSourcesError(null);

    const editing = editingSource();

    if (editing) {
      const updateReq: UpdateSourceRequest = {
        name: formData.name,
        url: formData.url,
        priority: formData.priority,
        enabled: formData.enabled,
      };
      const result = await updateDefaultSource(editing.id, updateReq);

      setSourceSubmitting(false);

      if (result.success) {
        setSourceModalOpen(false);
        setEditingSource(null);
        loadDefaultSources();
      } else {
        setSourcesError(result.message);
      }
    } else {
      const result = await createDefaultSource(formData);

      setSourceSubmitting(false);

      if (result.success) {
        setSourceModalOpen(false);
        loadDefaultSources();
      } else {
        setSourcesError(result.message);
      }
    }
  };

  const handleSourceFormCancel = () => {
    setSourceModalOpen(false);
    setEditingSource(null);
  };

  const openDeleteSourceModal = (source: SourceDefault) => {
    setSourceToDelete(source);
    setDeleteSourceModalOpen(true);
  };

  const confirmDeleteSource = async () => {
    const source = sourceToDelete();
    if (!source) return;

    setDeletingSource(true);
    setSourcesError(null);

    const result = await deleteDefaultSource(source.id);

    setDeletingSource(false);

    if (result.success) {
      setDeleteSourceModalOpen(false);
      setSourceToDelete(null);
      loadDefaultSources();
    } else {
      setSourcesError(result.message);
    }
  };

  const cancelDeleteSource = () => {
    setDeleteSourceModalOpen(false);
    setSourceToDelete(null);
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

    // Load devmode state from localStorage
    setDevModeEnabled(isDevMode());

    // Load server settings if user is root
    loadServerSettings();

    // Load default sources if user is root
    loadDefaultSources();
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
              <SummaryNavItem id="default-sources" label="Default Sources" />
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
            <Show when={isRootUser()}>
              <SummaryItem
                title="Developer Mode"
                description="Enable debug console and verbose logging."
              >
                <SummaryToggle
                  checked={devModeEnabled()}
                  onChange={handleDevModeToggle}
                  disabled={devModeUpdating()}
                />
              </SummaryItem>
            </Show>
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

          {/* Default Sources Section (Root users only) */}
          <SummarySection
            id="default-sources"
            title="Default Sources"
            description="Manage system-wide default upstream sources available to all users"
          >
            <div class="space-y-4">
              <Show when={sourcesError()}>
                <div class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
                  {sourcesError()}
                </div>
              </Show>

              <Show
                when={!sourcesLoading()}
                fallback={
                  <div class="flex items-center justify-center py-8">
                    <Spinner size="lg" />
                  </div>
                }
              >
                <div class="space-y-2">
                  <For each={defaultSources()}>
                    {(source) => (
                      <div class="flex items-center justify-between p-3 bg-muted/50 rounded-md border border-border">
                        <div class="flex-1 min-w-0">
                          <div class="flex items-center gap-2">
                            <span class="font-medium truncate">
                              {source.name}
                            </span>
                            <span
                              class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                                source.enabled
                                  ? "bg-primary/10 text-primary"
                                  : "bg-muted text-muted-foreground"
                              }`}
                            >
                              {source.enabled ? "Enabled" : "Disabled"}
                            </span>
                            <span class="text-xs text-muted-foreground">
                              Priority: {source.priority}
                            </span>
                          </div>
                          <div class="text-sm text-muted-foreground font-mono truncate mt-1">
                            {source.url}
                          </div>
                        </div>
                        <div class="flex items-center gap-2 ml-4">
                          <button
                            onClick={() => handleEditSource(source)}
                            class="p-2 rounded-md hover:bg-muted transition-colors"
                            title="Edit source"
                          >
                            <Icon
                              name="pencil"
                              size="sm"
                              class="text-muted-foreground hover:text-foreground"
                            />
                          </button>
                          <button
                            onClick={() => openDeleteSourceModal(source)}
                            class="p-2 rounded-md hover:bg-destructive/10 transition-colors"
                            title="Delete source"
                          >
                            <Icon
                              name="trash"
                              size="sm"
                              class="text-muted-foreground hover:text-destructive"
                            />
                          </button>
                        </div>
                      </div>
                    )}
                  </For>

                  <Show when={defaultSources().length === 0}>
                    <div class="text-center py-8 text-muted-foreground">
                      No default sources configured yet.
                    </div>
                  </Show>
                </div>

                <div class="pt-4">
                  <SummaryButton onClick={handleAddSource}>
                    Add Default Source
                  </SummaryButton>
                </div>
              </Show>
            </div>
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

      {/* Source Form Modal */}
      <Modal
        isOpen={sourceModalOpen()}
        onClose={handleSourceFormCancel}
        title={editingSource() ? "Edit Default Source" : "Add Default Source"}
      >
        <SourceForm
          onSubmit={handleSourceFormSubmit}
          onCancel={handleSourceFormCancel}
          initialData={
            editingSource()
              ? {
                  ...editingSource()!,
                  is_system: true,
                }
              : undefined
          }
          isSubmitting={sourceSubmitting()}
        />
      </Modal>

      {/* Delete Source Confirmation Modal */}
      <Modal
        isOpen={deleteSourceModalOpen()}
        onClose={cancelDeleteSource}
        title="Delete Default Source"
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            Are you sure you want to delete the default source{" "}
            <span class="text-foreground font-medium">
              "{sourceToDelete()?.name}"
            </span>
            ? This action cannot be undone and will affect all users.
          </p>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelDeleteSource}
              disabled={deletingSource()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={confirmDeleteSource}
              disabled={deletingSource()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={deletingSource()}>
                <Spinner size="sm" />
              </Show>
              <span>{deletingSource() ? "Deleting..." : "Delete"}</span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
