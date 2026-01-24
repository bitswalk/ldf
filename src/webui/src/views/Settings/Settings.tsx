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
import { Spinner } from "../../components/Spinner";
import { themeService } from "../../services/theme";
import { i18nService, t, type LocalePreference } from "../../services/i18n";
import { getServerUrl } from "../../services/storage";
import {
  getServerSettings,
  updateServerSetting,
  isRootUser,
  setDevMode,
  resetDatabase,
  type ServerSetting,
} from "../../services/settings";
import { isDevMode } from "../../lib/utils";
import {
  getBrandingAssetInfo,
  getBrandingAssetURL,
  uploadBrandingAsset,
  deleteBrandingAsset,
  getAppName,
  setAppName,
  updateDocumentTitle,
  DEFAULT_APP_NAME,
  APP_NAME_MAX_LENGTH,
  type BrandingAssetInfo,
} from "../../services/branding";

type ThemePreference = "system" | "light" | "dark";

interface SettingsProps {
  onBack?: () => void;
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

  // Database reset state
  const [resetDbModalOpen, setResetDbModalOpen] = createSignal(false);
  const [resetDbConfirmation, setResetDbConfirmation] = createSignal("");
  const [resettingDb, setResettingDb] = createSignal(false);
  const [resetDbError, setResetDbError] = createSignal<string | null>(null);

  // Branding state (for root users)
  const [logoInfo, setLogoInfo] = createSignal<BrandingAssetInfo | null>(null);
  const [faviconInfo, setFaviconInfo] = createSignal<BrandingAssetInfo | null>(
    null,
  );
  const [brandingLoading, setBrandingLoading] = createSignal(false);
  const [uploadingLogo, setUploadingLogo] = createSignal(false);
  const [uploadingFavicon, setUploadingFavicon] = createSignal(false);
  const [logoUploadProgress, setLogoUploadProgress] = createSignal(0);
  const [faviconUploadProgress, setFaviconUploadProgress] = createSignal(0);
  const [brandingError, setBrandingError] = createSignal<string | null>(null);
  const [deleteLogoModalOpen, setDeleteLogoModalOpen] = createSignal(false);
  const [deleteFaviconModalOpen, setDeleteFaviconModalOpen] =
    createSignal(false);
  const [deletingLogo, setDeletingLogo] = createSignal(false);
  const [deletingFavicon, setDeletingFavicon] = createSignal(false);

  // App name state (for root users)
  const [appName, setAppNameValue] = createSignal("");
  const [appNameSaving, setAppNameSaving] = createSignal(false);

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

  const openResetDbModal = () => {
    setResetDbConfirmation("");
    setResetDbError(null);
    setResetDbModalOpen(true);
  };

  const cancelResetDb = () => {
    setResetDbModalOpen(false);
    setResetDbConfirmation("");
    setResetDbError(null);
  };

  const confirmResetDb = async () => {
    if (resetDbConfirmation() !== "RESET_DATABASE") {
      setResetDbError(t("settings.server.database.reset.invalidConfirmation"));
      return;
    }

    setResettingDb(true);
    setResetDbError(null);

    const result = await resetDatabase("RESET_DATABASE");

    setResettingDb(false);

    if (result.success) {
      setResetDbModalOpen(false);
      // Reload the page to reflect the reset state
      window.location.reload();
    } else {
      setResetDbError(result.message);
    }
  };

  const loadBrandingAssets = async () => {
    if (!isRootUser()) return;

    setBrandingLoading(true);
    setBrandingError(null);

    const [logoResult, faviconResult, appNameResult] = await Promise.all([
      getBrandingAssetInfo("logo"),
      getBrandingAssetInfo("favicon"),
      getAppName(),
    ]);

    if (logoResult.success) {
      setLogoInfo(logoResult.info);
    }
    if (faviconResult.success) {
      setFaviconInfo(faviconResult.info);
    }
    if (appNameResult.success) {
      // If it's the default name, show empty so the placeholder shows
      setAppNameValue(
        appNameResult.appName === DEFAULT_APP_NAME ? "" : appNameResult.appName,
      );
    }

    setBrandingLoading(false);
  };

  const handleAppNameSave = async () => {
    setAppNameSaving(true);
    setBrandingError(null);

    const result = await setAppName(appName());

    setAppNameSaving(false);

    if (result.success) {
      // Update the document title
      updateDocumentTitle(result.appName);
      // Trigger a custom event so the Header can update
      window.dispatchEvent(
        new CustomEvent("appNameChanged", { detail: result.appName }),
      );
    } else {
      setBrandingError(result.message);
    }
  };

  const handleLogoUpload = async (
    event: Event & { currentTarget: HTMLInputElement },
  ) => {
    const file = event.currentTarget.files?.[0];
    if (!file) return;

    setUploadingLogo(true);
    setLogoUploadProgress(0);
    setBrandingError(null);

    const result = await uploadBrandingAsset("logo", file, (progress) => {
      setLogoUploadProgress(progress);
    });

    setUploadingLogo(false);

    if (result.success) {
      loadBrandingAssets();
    } else {
      setBrandingError(result.message);
    }

    event.currentTarget.value = "";
  };

  const handleFaviconUpload = async (
    event: Event & { currentTarget: HTMLInputElement },
  ) => {
    const file = event.currentTarget.files?.[0];
    if (!file) return;

    setUploadingFavicon(true);
    setFaviconUploadProgress(0);
    setBrandingError(null);

    const result = await uploadBrandingAsset("favicon", file, (progress) => {
      setFaviconUploadProgress(progress);
    });

    setUploadingFavicon(false);

    if (result.success) {
      loadBrandingAssets();
      // Update the favicon in the document head
      const faviconLink = document.querySelector(
        "link[rel='icon']",
      ) as HTMLLinkElement;
      if (faviconLink) {
        const url = getBrandingAssetURL("favicon");
        if (url) {
          faviconLink.href = url + "?t=" + Date.now();
        }
      }
    } else {
      setBrandingError(result.message);
    }

    event.currentTarget.value = "";
  };

  const confirmDeleteLogo = async () => {
    setDeletingLogo(true);
    setBrandingError(null);

    const result = await deleteBrandingAsset("logo");

    setDeletingLogo(false);

    if (result.success) {
      setDeleteLogoModalOpen(false);
      setLogoInfo(null);
    } else {
      setBrandingError(result.message);
    }
  };

  const confirmDeleteFavicon = async () => {
    setDeletingFavicon(true);
    setBrandingError(null);

    const result = await deleteBrandingAsset("favicon");

    setDeletingFavicon(false);

    if (result.success) {
      setDeleteFaviconModalOpen(false);
      setFaviconInfo(null);
    } else {
      setBrandingError(result.message);
    }
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

    // Load language packs if user is root
    loadLanguagePacks();

    // Load branding assets if user is root
    loadBrandingAssets();
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

            {/* Branding section (root users only) */}
            <Show when={isRootUser()}>
              <div class="pt-6">
                <h3 class="text-lg font-semibold mb-4">
                  {t("settings.theme.branding.title")}
                </h3>
                <p class="text-sm text-muted-foreground mb-6">
                  {t("settings.theme.branding.description")}
                </p>

                <Show when={brandingError()}>
                  <div class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm mb-4">
                    {brandingError()}
                  </div>
                </Show>

                <Show
                  when={!brandingLoading()}
                  fallback={
                    <div class="flex items-center justify-center py-8">
                      <Spinner size="lg" />
                    </div>
                  }
                >
                  <div class="space-y-6">
                    {/* Application Name */}
                    <div class="space-y-3">
                      <div>
                        <h4 class="text-sm font-medium">
                          {t("settings.theme.branding.appName.title")}
                        </h4>
                        <p class="text-xs text-muted-foreground">
                          {t("settings.theme.branding.appName.description")}
                        </p>
                      </div>

                      <div class="flex items-center gap-3">
                        <div class="flex-1">
                          <input
                            type="text"
                            value={appName()}
                            onInput={(e) =>
                              setAppNameValue(e.currentTarget.value)
                            }
                            placeholder={t(
                              "settings.theme.branding.appName.placeholder",
                            )}
                            maxLength={APP_NAME_MAX_LENGTH}
                            class="w-full px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                          />
                          <div class="flex justify-between mt-1">
                            <p class="text-xs text-muted-foreground">
                              {t("settings.theme.branding.appName.hint")}
                            </p>
                            <p class="text-xs text-muted-foreground">
                              {t("settings.theme.branding.appName.charCount", {
                                count: appName().length,
                              })}
                            </p>
                          </div>
                        </div>
                        <button
                          onClick={handleAppNameSave}
                          disabled={appNameSaving()}
                          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
                        >
                          <Show when={appNameSaving()}>
                            <Spinner size="sm" />
                          </Show>
                          <span>{t("common.actions.save")}</span>
                        </button>
                      </div>
                    </div>

                    {/* Logo Upload */}
                    <div class="space-y-3">
                      <div class="flex items-center justify-between">
                        <div>
                          <h4 class="text-sm font-medium">
                            {t("settings.theme.branding.logo.title")}
                          </h4>
                          <p class="text-xs text-muted-foreground">
                            {t("settings.theme.branding.logo.description")}
                          </p>
                        </div>
                        <Show when={logoInfo()?.exists}>
                          <button
                            onClick={() => setDeleteLogoModalOpen(true)}
                            class="p-2 rounded-md hover:bg-destructive/10 transition-colors"
                            title={t("common.actions.delete")}
                          >
                            <Icon
                              name="trash"
                              size="sm"
                              class="text-muted-foreground hover:text-destructive"
                            />
                          </button>
                        </Show>
                      </div>

                      <div class="flex items-center gap-4">
                        <Show
                          when={logoInfo()?.exists}
                          fallback={
                            <div class="w-24 h-24 bg-muted rounded-md flex items-center justify-center border-2 border-dashed border-border">
                              <Icon
                                name="image"
                                size="lg"
                                class="text-muted-foreground"
                              />
                            </div>
                          }
                        >
                          <div class="w-24 h-24 bg-muted rounded-md flex items-center justify-center border border-border overflow-hidden">
                            <img
                              src={
                                getBrandingAssetURL("logo") + "?t=" + Date.now()
                              }
                              alt="Logo"
                              class="max-w-full max-h-full object-contain"
                            />
                          </div>
                        </Show>

                        <div class="flex-1">
                          <label class="block">
                            <div class="relative">
                              <input
                                type="file"
                                accept="image/png,image/jpeg,image/webp,image/svg+xml"
                                onChange={handleLogoUpload}
                                disabled={uploadingLogo()}
                                class="block w-full text-sm text-muted-foreground
                                  file:mr-4 file:py-2 file:px-4
                                  file:rounded-md file:border file:border-border
                                  file:text-sm file:font-medium
                                  file:bg-muted file:text-foreground
                                  hover:file:bg-muted/80
                                  file:cursor-pointer cursor-pointer
                                  disabled:opacity-50 disabled:cursor-not-allowed"
                              />
                              <Show when={uploadingLogo()}>
                                <div class="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-2">
                                  <span class="text-xs text-muted-foreground">
                                    {logoUploadProgress()}%
                                  </span>
                                  <Spinner size="sm" />
                                </div>
                              </Show>
                            </div>
                            <p class="text-xs text-muted-foreground mt-1">
                              {t("settings.theme.branding.logo.formats")}
                            </p>
                          </label>
                        </div>
                      </div>
                    </div>

                    {/* Favicon Upload */}
                    <div class="space-y-3">
                      <div class="flex items-center justify-between">
                        <div>
                          <h4 class="text-sm font-medium">
                            {t("settings.theme.branding.favicon.title")}
                          </h4>
                          <p class="text-xs text-muted-foreground">
                            {t("settings.theme.branding.favicon.description")}
                          </p>
                        </div>
                        <Show when={faviconInfo()?.exists}>
                          <button
                            onClick={() => setDeleteFaviconModalOpen(true)}
                            class="p-2 rounded-md hover:bg-destructive/10 transition-colors"
                            title={t("common.actions.delete")}
                          >
                            <Icon
                              name="trash"
                              size="sm"
                              class="text-muted-foreground hover:text-destructive"
                            />
                          </button>
                        </Show>
                      </div>

                      <div class="flex items-center gap-4">
                        <Show
                          when={faviconInfo()?.exists}
                          fallback={
                            <div class="w-16 h-16 bg-muted rounded-md flex items-center justify-center border-2 border-dashed border-border">
                              <Icon
                                name="image"
                                size="md"
                                class="text-muted-foreground"
                              />
                            </div>
                          }
                        >
                          <div class="w-16 h-16 bg-muted rounded-md flex items-center justify-center border border-border overflow-hidden">
                            <img
                              src={
                                getBrandingAssetURL("favicon") +
                                "?t=" +
                                Date.now()
                              }
                              alt="Favicon"
                              class="max-w-full max-h-full object-contain"
                            />
                          </div>
                        </Show>

                        <div class="flex-1">
                          <label class="block">
                            <div class="relative">
                              <input
                                type="file"
                                accept="image/png,image/x-icon,image/svg+xml,.ico"
                                onChange={handleFaviconUpload}
                                disabled={uploadingFavicon()}
                                class="block w-full text-sm text-muted-foreground
                                  file:mr-4 file:py-2 file:px-4
                                  file:rounded-md file:border file:border-border
                                  file:text-sm file:font-medium
                                  file:bg-muted file:text-foreground
                                  hover:file:bg-muted/80
                                  file:cursor-pointer cursor-pointer
                                  disabled:opacity-50 disabled:cursor-not-allowed"
                              />
                              <Show when={uploadingFavicon()}>
                                <div class="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-2">
                                  <span class="text-xs text-muted-foreground">
                                    {faviconUploadProgress()}%
                                  </span>
                                  <Spinner size="sm" />
                                </div>
                              </Show>
                            </div>
                            <p class="text-xs text-muted-foreground mt-1">
                              {t("settings.theme.branding.favicon.formats")}
                            </p>
                          </label>
                        </div>
                      </div>
                    </div>
                  </div>
                </Show>
              </div>
            </Show>
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
              onResetDatabase={openResetDbModal}
            />
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

      {/* Database Reset Confirmation Modal */}
      <Modal
        isOpen={resetDbModalOpen()}
        onClose={cancelResetDb}
        title={t("settings.server.database.reset.modal.title")}
      >
        <section class="flex flex-col gap-6">
          <div class="p-4 bg-destructive/10 border border-destructive/20 rounded-md">
            <p class="text-destructive font-medium mb-2">
              {t("settings.server.database.reset.modal.warning")}
            </p>
            <p class="text-sm text-muted-foreground">
              {t("settings.server.database.reset.modal.description")}
            </p>
          </div>

          <Show when={resetDbError()}>
            <div class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
              {resetDbError()}
            </div>
          </Show>

          <div class="space-y-2">
            <label class="text-sm font-medium" for="reset-confirmation">
              {t("settings.server.database.reset.modal.confirmLabel")}
            </label>
            <input
              id="reset-confirmation"
              type="text"
              class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-destructive transition-colors font-mono"
              placeholder="RESET_DATABASE"
              value={resetDbConfirmation()}
              onInput={(e) => setResetDbConfirmation(e.target.value)}
            />
            <p class="text-xs text-muted-foreground">
              {t("settings.server.database.reset.modal.confirmHint")}
            </p>
          </div>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelResetDb}
              disabled={resettingDb()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={confirmResetDb}
              disabled={
                resettingDb() || resetDbConfirmation() !== "RESET_DATABASE"
              }
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={resettingDb()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {resettingDb()
                  ? t("settings.server.database.reset.modal.resetting")
                  : t("settings.server.database.reset.modal.confirmButton")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>

      {/* Delete Logo Confirmation Modal */}
      <Modal
        isOpen={deleteLogoModalOpen()}
        onClose={() => setDeleteLogoModalOpen(false)}
        title={t("settings.theme.branding.logo.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("settings.theme.branding.logo.delete.confirm")}
          </p>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setDeleteLogoModalOpen(false)}
              disabled={deletingLogo()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={confirmDeleteLogo}
              disabled={deletingLogo()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={deletingLogo()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {deletingLogo()
                  ? t("common.status.deleting")
                  : t("common.actions.delete")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>

      {/* Delete Favicon Confirmation Modal */}
      <Modal
        isOpen={deleteFaviconModalOpen()}
        onClose={() => setDeleteFaviconModalOpen(false)}
        title={t("settings.theme.branding.favicon.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("settings.theme.branding.favicon.delete.confirm")}
          </p>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setDeleteFaviconModalOpen(false)}
              disabled={deletingFavicon()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={confirmDeleteFavicon}
              disabled={deletingFavicon()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={deletingFavicon()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {deletingFavicon()
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
