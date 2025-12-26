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
import { i18nService, t, type LocalePreference } from "../../services/i18n";
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
  onViewSource?: (sourceId: string, sourceType: "default" | "user") => void;
}

export const Settings: Component<SettingsProps> = (props) => {
  const [themePreference, setThemePreference] =
    createSignal<ThemePreference>("system");
  const [languagePreference, setLanguagePreference] =
    createSignal<LocalePreference>("system");
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

  // Language packs state (for root users)
  const [languagePacks, setLanguagePacks] = createSignal<
    Array<{ locale: string; name: string; version: string; author?: string }>
  >([]);
  const [langPacksLoading, setLangPacksLoading] = createSignal(false);
  const [langPacksError, setLangPacksError] = createSignal<string | null>(null);
  const [uploadingLangPack, setUploadingLangPack] = createSignal(false);
  const [deleteLangPackModalOpen, setDeleteLangPackModalOpen] =
    createSignal(false);
  const [langPackToDelete, setLangPackToDelete] = createSignal<{
    locale: string;
    name: string;
  } | null>(null);
  const [deletingLangPack, setDeletingLangPack] = createSignal(false);

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

  const loadLanguagePacks = async () => {
    if (!isRootUser()) return;

    setLangPacksLoading(true);
    setLangPacksError(null);

    const result = await i18nService.listLanguagePacksFromServer();
    if (result.success && result.packs) {
      setLanguagePacks(result.packs);
    } else {
      setLangPacksError(result.error || "Failed to load language packs");
    }

    setLangPacksLoading(false);
  };

  const handleLangPackUpload = async (
    event: Event & { currentTarget: HTMLInputElement },
  ) => {
    const file = event.currentTarget.files?.[0];
    if (!file) return;

    setUploadingLangPack(true);
    setLangPacksError(null);

    const result = await i18nService.installLanguagePack(file);

    setUploadingLangPack(false);

    if (result.success) {
      loadLanguagePacks();
    } else {
      setLangPacksError(
        result.error || t("settings.general.languagePacks.uploadError"),
      );
    }

    // Reset the input
    event.currentTarget.value = "";
  };

  const openDeleteLangPackModal = (pack: { locale: string; name: string }) => {
    setLangPackToDelete(pack);
    setDeleteLangPackModalOpen(true);
  };

  const confirmDeleteLangPack = async () => {
    const pack = langPackToDelete();
    if (!pack) return;

    setDeletingLangPack(true);
    setLangPacksError(null);

    const result = await i18nService.deleteLanguagePack(pack.locale);

    setDeletingLangPack(false);

    if (result.success) {
      setDeleteLangPackModalOpen(false);
      setLangPackToDelete(null);
      loadLanguagePacks();
    } else {
      setLangPacksError(
        result.error || t("settings.general.languagePacks.deleteError"),
      );
    }
  };

  const cancelDeleteLangPack = () => {
    setDeleteLangPackModalOpen(false);
    setLangPackToDelete(null);
  };

  const handleAddSource = () => {
    setEditingSource(null);
    setSourceModalOpen(true);
  };

  const handleEditSource = (source: SourceDefault) => {
    // Navigate to source details view instead of opening modal
    props.onViewSource?.(source.id, "default");
  };

  const handleSourceFormSubmit = async (formData: CreateSourceRequest) => {
    setSourceSubmitting(true);
    setSourcesError(null);

    const editing = editingSource();

    if (editing) {
      const updateReq: UpdateSourceRequest = {
        name: formData.name,
        url: formData.url,
        component_id: formData.component_id,
        retrieval_method: formData.retrieval_method,
        url_template: formData.url_template,
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

    // Load current language preference from i18n service
    setLanguagePreference(i18nService.getPreference());

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

    // Load language packs if user is root
    loadLanguagePacks();
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

  const handleLanguageChange = async (preference: LocalePreference) => {
    setLanguagePreference(preference);
    await i18nService.setPreference(preference);
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
          <span>{t("common.actions.back")}</span>
        </button>
        <h1 class="text-2xl font-bold">{t("settings.title")}</h1>
      </header>

      {/* Summary Layout */}
      <Summary
        defaultSection="general"
        defaultExpanded={["general", "appearance"]}
        class="flex-1"
      >
        <SummaryNav>
          <SummaryCategory
            id="general"
            label={t("settings.categories.general")}
            icon="gear"
          >
            <SummaryNavItem id="general" label={t("settings.general.title")} />
            <SummaryNavItem id="privacy" label={t("settings.privacy.title")} />
            <Show when={isRootUser()}>
              <SummaryNavItem
                id="language-packs"
                label={t("settings.general.languagePacks.title")}
              />
            </Show>
          </SummaryCategory>
          <SummaryCategory
            id="appearance"
            label={t("settings.categories.appearance")}
            icon="palette"
          >
            <SummaryNavItem id="theme" label={t("settings.theme.title")} />
          </SummaryCategory>
          <SummaryCategory
            id="server"
            label={t("settings.categories.server")}
            icon="plugs-connected"
          >
            <SummaryNavItem
              id="server-connection"
              label={t("settings.server.connection.title")}
            />
            <Show when={isRootUser()}>
              <SummaryNavItem
                id="server-settings"
                label={t("settings.server.settings.title")}
              />
              <SummaryNavItem
                id="default-sources"
                label={t("settings.server.defaultSources.title")}
              />
            </Show>
          </SummaryCategory>
          <SummaryCategory
            id="account"
            label={t("settings.categories.account")}
            icon="user"
          >
            <SummaryNavItem
              id="profile"
              label={t("settings.account.profile.title")}
            />
            <SummaryNavItem
              id="security"
              label={t("settings.account.security.title")}
            />
          </SummaryCategory>
        </SummaryNav>

        <SummaryContent>
          {/* General Section */}
          <SummarySection
            id="general"
            title={t("settings.general.title")}
            description={t("settings.general.description")}
          >
            <SummaryItem
              title={t("settings.general.systemPrompts.title")}
              description={t("settings.general.systemPrompts.description")}
            >
              <SummaryToggle
                checked={useSystemPrompts()}
                onChange={setUseSystemPrompts}
              />
            </SummaryItem>
            <SummaryItem
              title={t("settings.general.language.title")}
              description={t("settings.general.language.description")}
            >
              <SummarySelect
                value={languagePreference()}
                options={[
                  {
                    value: "system",
                    label: t("settings.general.language.options.system"),
                  },
                  {
                    value: "en",
                    label: t("settings.general.language.options.en"),
                  },
                  {
                    value: "fr",
                    label: t("settings.general.language.options.fr"),
                  },
                  {
                    value: "de",
                    label: t("settings.general.language.options.de"),
                  },
                ]}
                onChange={(value) =>
                  handleLanguageChange(value as LocalePreference)
                }
              />
            </SummaryItem>
            <Show when={isRootUser()}>
              <SummaryItem
                title={t("settings.general.devMode.title")}
                description={t("settings.general.devMode.description")}
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
            title={t("settings.privacy.title")}
            description={t("settings.privacy.description")}
          >
            <SummaryItem
              title={t("settings.privacy.redactValues.title")}
              description={t("settings.privacy.redactValues.description")}
            >
              <SummaryToggle
                checked={redactPrivateValues()}
                onChange={setRedactPrivateValues}
              />
            </SummaryItem>
            <SummaryItem
              title={t("settings.privacy.privateFiles.title")}
              description={t("settings.privacy.privateFiles.description")}
            >
              <SummaryButton onClick={() => {}}>
                {t("settings.privacy.privateFiles.editButton")}
              </SummaryButton>
            </SummaryItem>
          </SummarySection>

          {/* Language Packs Section (Root users only) */}
          <SummarySection
            id="language-packs"
            title={t("settings.general.languagePacks.title")}
            description={t("settings.general.languagePacks.description")}
          >
            <div class="space-y-4">
              <Show when={langPacksError()}>
                <div class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
                  {langPacksError()}
                </div>
              </Show>

              <Show
                when={!langPacksLoading()}
                fallback={
                  <div class="flex items-center justify-center py-8">
                    <Spinner size="lg" />
                  </div>
                }
              >
                {/* Language packs table */}
                <Show
                  when={languagePacks().length > 0}
                  fallback={
                    <div class="text-center py-8 text-muted-foreground">
                      {t("settings.general.languagePacks.noPacksInstalled")}
                    </div>
                  }
                >
                  <div class="border border-border rounded-md overflow-hidden">
                    <table class="w-full text-sm">
                      <thead class="bg-muted/50">
                        <tr>
                          <th class="px-4 py-2 text-left font-medium">
                            {t("settings.general.languagePacks.table.locale")}
                          </th>
                          <th class="px-4 py-2 text-left font-medium">
                            {t("settings.general.languagePacks.table.name")}
                          </th>
                          <th class="px-4 py-2 text-left font-medium">
                            {t("settings.general.languagePacks.table.version")}
                          </th>
                          <th class="px-4 py-2 text-left font-medium">
                            {t("settings.general.languagePacks.table.author")}
                          </th>
                          <th class="px-4 py-2 text-right font-medium">
                            {t("settings.general.languagePacks.table.actions")}
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        <For each={languagePacks()}>
                          {(pack) => (
                            <tr class="border-t border-border">
                              <td class="px-4 py-2 font-mono">{pack.locale}</td>
                              <td class="px-4 py-2">{pack.name}</td>
                              <td class="px-4 py-2 font-mono text-muted-foreground">
                                {pack.version}
                              </td>
                              <td class="px-4 py-2 text-muted-foreground">
                                {pack.author || "—"}
                              </td>
                              <td class="px-4 py-2 text-right">
                                <button
                                  onClick={() => openDeleteLangPackModal(pack)}
                                  class="p-1.5 rounded-md hover:bg-destructive/10 transition-colors"
                                  title={t("common.actions.delete")}
                                >
                                  <Icon
                                    name="trash"
                                    size="sm"
                                    class="text-muted-foreground hover:text-destructive"
                                  />
                                </button>
                              </td>
                            </tr>
                          )}
                        </For>
                      </tbody>
                    </table>
                  </div>
                </Show>

                {/* Upload section */}
                <div class="pt-4 border-t border-border mt-4">
                  <label class="block">
                    <span class="text-sm font-medium">
                      {t("settings.general.languagePacks.upload")}
                    </span>
                    <p class="text-xs text-muted-foreground mb-2">
                      {t("settings.general.languagePacks.supportedFormats")}
                    </p>
                    <div class="relative">
                      <input
                        type="file"
                        accept=".tar.xz,.tar.gz,.tgz,.xz"
                        onChange={handleLangPackUpload}
                        disabled={uploadingLangPack()}
                        class="block w-full text-sm text-muted-foreground
                          file:mr-4 file:py-2 file:px-4
                          file:rounded-md file:border file:border-border
                          file:text-sm file:font-medium
                          file:bg-muted file:text-foreground
                          hover:file:bg-muted/80
                          file:cursor-pointer cursor-pointer
                          disabled:opacity-50 disabled:cursor-not-allowed"
                      />
                      <Show when={uploadingLangPack()}>
                        <div class="absolute right-3 top-1/2 -translate-y-1/2">
                          <Spinner size="sm" />
                        </div>
                      </Show>
                    </div>
                  </label>
                </div>
              </Show>
            </div>
          </SummarySection>

          {/* Theme Section */}
          <SummarySection
            id="theme"
            title={t("settings.theme.title")}
            description={t("settings.theme.description")}
          >
            <SummaryItem
              title={t("settings.theme.colorScheme.title")}
              description={t("settings.theme.colorScheme.description")}
            >
              <SummarySelect
                value={themePreference()}
                options={[
                  {
                    value: "system",
                    label: t("settings.theme.colorScheme.options.system"),
                  },
                  {
                    value: "light",
                    label: t("settings.theme.colorScheme.options.light"),
                  },
                  {
                    value: "dark",
                    label: t("settings.theme.colorScheme.options.dark"),
                  },
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
            title={t("settings.server.connection.title")}
            description={t("settings.server.connection.description")}
          >
            <SummaryItem
              title={t("settings.server.connection.connectedServer.title")}
              description={
                serverUrl() ||
                t("settings.server.connection.connectedServer.noServer")
              }
            >
              <SummaryButton onClick={() => {}} disabled>
                {t("settings.server.connection.connectedServer.changeButton")}
              </SummaryButton>
            </SummaryItem>
            <SummaryItem
              title={t("settings.server.connection.status.title")}
              description={t("settings.server.connection.status.description")}
            >
              <SummaryButton onClick={() => {}}>
                {t("settings.server.connection.status.testButton")}
              </SummaryButton>
            </SummaryItem>
          </SummarySection>

          {/* Server Settings Section (Root users only) */}
          <SummarySection
            id="server-settings"
            title={t("settings.server.settings.title")}
            description={t("settings.server.settings.description")}
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
            title={t("settings.server.defaultSources.title")}
            description={t("settings.server.defaultSources.description")}
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
                            <button
                              onClick={() =>
                                props.onViewSource?.(source.id, "default")
                              }
                              class="font-medium truncate hover:text-primary hover:underline transition-colors text-left"
                            >
                              {source.name}
                            </button>
                            <span
                              class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                                source.enabled
                                  ? "bg-primary/10 text-primary"
                                  : "bg-muted text-muted-foreground"
                              }`}
                            >
                              {source.enabled
                                ? t("common.status.enabled")
                                : t("common.status.disabled")}
                            </span>
                            <span class="text-xs text-muted-foreground">
                              {t("settings.server.defaultSources.priority")}:{" "}
                              {source.priority}
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
                            title={t("common.actions.edit")}
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
                            title={t("common.actions.delete")}
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
                      {t("settings.server.defaultSources.noSources")}
                    </div>
                  </Show>
                </div>

                <div class="pt-4">
                  <SummaryButton onClick={handleAddSource}>
                    {t("settings.server.defaultSources.addButton")}
                  </SummaryButton>
                </div>
              </Show>
            </div>
          </SummarySection>

          {/* Profile Section */}
          <SummarySection
            id="profile"
            title={t("settings.account.profile.title")}
            description={t("settings.account.profile.description")}
          >
            <SummaryItem
              title={t("settings.account.profile.displayName.title")}
              description={t(
                "settings.account.profile.displayName.description",
              )}
            >
              <SummaryButton onClick={() => {}}>
                {t("settings.account.profile.displayName.editButton")}
              </SummaryButton>
            </SummaryItem>
            <SummaryItem
              title={t("settings.account.profile.avatar.title")}
              description={t("settings.account.profile.avatar.description")}
            >
              <SummaryButton onClick={() => {}}>
                {t("settings.account.profile.avatar.changeButton")}
              </SummaryButton>
            </SummaryItem>
          </SummarySection>

          {/* Security Section */}
          <SummarySection
            id="security"
            title={t("settings.account.security.title")}
            description={t("settings.account.security.description")}
          >
            <SummaryItem
              title={t("settings.account.security.changePassword.title")}
              description={t(
                "settings.account.security.changePassword.description",
              )}
            >
              <SummaryButton onClick={() => {}}>
                {t("settings.account.security.changePassword.button")}
              </SummaryButton>
            </SummaryItem>
            <SummaryItem
              title={t("settings.account.security.sessions.title")}
              description={t("settings.account.security.sessions.description")}
            >
              <SummaryButton onClick={() => {}}>
                {t("settings.account.security.sessions.viewButton")}
              </SummaryButton>
            </SummaryItem>
            <SummaryItem
              title={t("settings.account.security.deleteAccount.title")}
              description={t(
                "settings.account.security.deleteAccount.description",
              )}
            >
              <SummaryButton
                onClick={() => {}}
                class="text-destructive border-destructive hover:bg-destructive/10"
              >
                {t("settings.account.security.deleteAccount.button")}
              </SummaryButton>
            </SummaryItem>
          </SummarySection>
        </SummaryContent>
      </Summary>

      {/* Source Form Modal */}
      <Modal
        isOpen={sourceModalOpen()}
        onClose={handleSourceFormCancel}
        title={
          editingSource()
            ? t("settings.server.defaultSources.editSource")
            : t("settings.server.defaultSources.addSource")
        }
      >
        <SourceForm
          key={editingSource()?.id || "new"}
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
        title={t("settings.server.defaultSources.deleteSource.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("settings.server.defaultSources.deleteSource.confirm", {
              name: sourceToDelete()?.name || "",
            })}
          </p>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelDeleteSource}
              disabled={deletingSource()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
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
              <span>
                {deletingSource()
                  ? t("common.status.deleting")
                  : t("common.actions.delete")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>

      {/* Delete Language Pack Confirmation Modal */}
      <Modal
        isOpen={deleteLangPackModalOpen()}
        onClose={cancelDeleteLangPack}
        title={t("settings.general.languagePacks.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("settings.general.languagePacks.delete.confirm", {
              name: langPackToDelete()?.name || "",
              locale: langPackToDelete()?.locale || "",
            })}
          </p>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelDeleteLangPack}
              disabled={deletingLangPack()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={confirmDeleteLangPack}
              disabled={deletingLangPack()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={deletingLangPack()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {deletingLangPack()
                  ? t("common.status.deleting")
                  : t("common.actions.delete")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
