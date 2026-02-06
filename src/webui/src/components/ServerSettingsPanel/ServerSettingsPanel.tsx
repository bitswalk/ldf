import type { Component } from "solid-js";
import { createSignal, createMemo, Show, For } from "solid-js";
import { Icon } from "../Icon";
import { t } from "../../services/i18n";
import { getServerSetting, type ServerSetting } from "../../services/settings";

interface ServerSettingsPanelProps {
  settings: ServerSetting[];
  loading: boolean;
  error: string | null;
  updatingKeys: Set<string>;
  onUpdate: (key: string, value: string | number | boolean) => void;
  onRetry: () => void;
  onResetDatabase?: () => void;
}

// Setting group configuration - defines the hierarchy and UI behavior
interface SettingGroupConfig {
  key: string;
  label: string;
  description: string;
  icon: string;
  required: boolean; // If true, no enable/disable toggle
  children?: SettingConfig[];
  variants?: SettingVariantConfig[]; // For mutually exclusive options like storage.type
}

interface SettingVariantConfig {
  key: string; // The value of the parent setting that enables this variant
  label: string;
  children: SettingConfig[];
}

interface SettingConfig {
  key: string;
  label: string;
  description?: string;
  type: "string" | "int" | "bool";
  options?: { value: string; label: string }[];
}

// Define the settings structure (function to resolve i18n at call time)
const getSettingsStructure = (): SettingGroupConfig[] => [
  {
    key: "server",
    label: t("settings.serverPanel.server.label"),
    description: t("settings.serverPanel.server.description"),
    icon: "desktop-tower",
    required: true,
    children: [
      {
        key: "server.bind",
        label: t("settings.serverPanel.server.bind.label"),
        description: t("settings.serverPanel.server.bind.description"),
        type: "string",
      },
      {
        key: "server.port",
        label: t("settings.serverPanel.server.port.label"),
        description: t("settings.serverPanel.server.port.description"),
        type: "int",
      },
      {
        key: "server.tls.enabled",
        label: t("settings.serverPanel.server.tlsEnabled.label"),
        description: t("settings.serverPanel.server.tlsEnabled.description"),
        type: "bool",
      },
      {
        key: "server.tls.cert_path",
        label: t("settings.serverPanel.server.tlsCertPath.label"),
        description: t("settings.serverPanel.server.tlsCertPath.description"),
        type: "string",
      },
      {
        key: "server.tls.key_path",
        label: t("settings.serverPanel.server.tlsKeyPath.label"),
        description: t("settings.serverPanel.server.tlsKeyPath.description"),
        type: "string",
      },
    ],
  },
  {
    key: "log",
    label: t("settings.serverPanel.log.label"),
    description: t("settings.serverPanel.log.description"),
    icon: "note",
    required: false,
    children: [
      {
        key: "log.output",
        label: t("settings.serverPanel.log.output.label"),
        description: t("settings.serverPanel.log.output.description"),
        type: "string",
        options: [
          { value: "auto", label: t("settings.serverPanel.log.output.auto") },
          {
            value: "stdout",
            label: t("settings.serverPanel.log.output.stdout"),
          },
          {
            value: "journald",
            label: t("settings.serverPanel.log.output.journald"),
          },
        ],
      },
      {
        key: "log.level",
        label: t("settings.serverPanel.log.level.label"),
        description: t("settings.serverPanel.log.level.description"),
        type: "string",
        options: [
          { value: "debug", label: t("settings.serverPanel.log.level.debug") },
          { value: "info", label: t("settings.serverPanel.log.level.info") },
          { value: "warn", label: t("settings.serverPanel.log.level.warn") },
          { value: "error", label: t("settings.serverPanel.log.level.error") },
        ],
      },
    ],
  },
  {
    key: "database",
    label: t("settings.serverPanel.database.label"),
    description: t("settings.serverPanel.database.description"),
    icon: "database",
    required: true,
    children: [
      {
        key: "database.path",
        label: t("settings.serverPanel.database.path.label"),
        description: t("settings.serverPanel.database.path.description"),
        type: "string",
      },
    ],
  },
  {
    key: "storage",
    label: t("settings.serverPanel.storage.label"),
    description: t("settings.serverPanel.storage.description"),
    icon: "hard-drives",
    required: true,
    variants: [
      {
        key: "local",
        label: t("settings.serverPanel.storage.local.label"),
        children: [
          {
            key: "storage.local.path",
            label: t("settings.serverPanel.storage.local.path.label"),
            description: t(
              "settings.serverPanel.storage.local.path.description",
            ),
            type: "string",
          },
        ],
      },
      {
        key: "s3",
        label: t("settings.serverPanel.storage.s3.label"),
        children: [
          {
            key: "storage.s3.provider",
            label: t("settings.serverPanel.storage.s3.provider.label"),
            description: t(
              "settings.serverPanel.storage.s3.provider.description",
            ),
            type: "string",
            options: [
              {
                value: "garage",
                label: t("settings.serverPanel.storage.s3.provider.garage"),
              },
              {
                value: "minio",
                label: t("settings.serverPanel.storage.s3.provider.minio"),
              },
              {
                value: "aws",
                label: t("settings.serverPanel.storage.s3.provider.aws"),
              },
              {
                value: "other",
                label: t("settings.serverPanel.storage.s3.provider.other"),
              },
            ],
          },
          {
            key: "storage.s3.endpoint",
            label: t("settings.serverPanel.storage.s3.endpoint.label"),
            description: t(
              "settings.serverPanel.storage.s3.endpoint.description",
            ),
            type: "string",
          },
          {
            key: "storage.s3.region",
            label: t("settings.serverPanel.storage.s3.region.label"),
            description: t(
              "settings.serverPanel.storage.s3.region.description",
            ),
            type: "string",
          },
          {
            key: "storage.s3.bucket",
            label: t("settings.serverPanel.storage.s3.bucket.label"),
            description: t(
              "settings.serverPanel.storage.s3.bucket.description",
            ),
            type: "string",
          },
          {
            key: "storage.s3.access_key",
            label: t("settings.serverPanel.storage.s3.accessKey.label"),
            description: t(
              "settings.serverPanel.storage.s3.accessKey.description",
            ),
            type: "string",
          },
          {
            key: "storage.s3.secret_key",
            label: t("settings.serverPanel.storage.s3.secretKey.label"),
            description: t(
              "settings.serverPanel.storage.s3.secretKey.description",
            ),
            type: "string",
          },
        ],
      },
    ],
  },
  {
    key: "sync",
    label: t("settings.serverPanel.sync.label"),
    description: t("settings.serverPanel.sync.description"),
    icon: "arrows-clockwise",
    required: false,
    children: [
      {
        key: "sync.cache_duration",
        label: t("settings.serverPanel.sync.cacheDuration.label"),
        description: t("settings.serverPanel.sync.cacheDuration.description"),
        type: "int",
      },
    ],
  },
  {
    key: "build",
    label: t("settings.serverPanel.build.label"),
    description: t("settings.serverPanel.build.description"),
    icon: "hammer",
    required: false,
    children: [
      {
        key: "build.workspace",
        label: t("settings.serverPanel.build.workspace.label"),
        description: t("settings.serverPanel.build.workspace.description"),
        type: "string",
      },
      {
        key: "build.workers",
        label: t("settings.serverPanel.build.workers.label"),
        description: t("settings.serverPanel.build.workers.description"),
        type: "int",
      },
    ],
  },
  {
    key: "download",
    label: t("settings.serverPanel.download.label"),
    description: t("settings.serverPanel.download.description"),
    icon: "download-simple",
    required: false,
    children: [
      {
        key: "download.cache.enabled",
        label: t("settings.serverPanel.download.cacheEnabled.label"),
        description: t(
          "settings.serverPanel.download.cacheEnabled.description",
        ),
        type: "bool",
      },
      {
        key: "download.cache.max_size_gb",
        label: t("settings.serverPanel.download.cacheMaxSize.label"),
        description: t(
          "settings.serverPanel.download.cacheMaxSize.description",
        ),
        type: "int",
      },
      {
        key: "download.proxy_url",
        label: t("settings.serverPanel.download.proxyUrl.label"),
        description: t("settings.serverPanel.download.proxyUrl.description"),
        type: "string",
      },
      {
        key: "download.local_mirror_path",
        label: t("settings.serverPanel.download.localMirrorPath.label"),
        description: t(
          "settings.serverPanel.download.localMirrorPath.description",
        ),
        type: "string",
      },
      {
        key: "download.throttle.per_worker_mbps",
        label: t("settings.serverPanel.download.throttlePerWorker.label"),
        description: t(
          "settings.serverPanel.download.throttlePerWorker.description",
        ),
        type: "int",
      },
      {
        key: "download.throttle.global_mbps",
        label: t("settings.serverPanel.download.throttleGlobal.label"),
        description: t(
          "settings.serverPanel.download.throttleGlobal.description",
        ),
        type: "int",
      },
    ],
  },
  {
    key: "security",
    label: t("settings.serverPanel.security.label"),
    description: t("settings.serverPanel.security.description"),
    icon: "shield-check",
    required: false,
    children: [
      {
        key: "security.rate_limit.enabled",
        label: t("settings.serverPanel.security.rateLimitEnabled.label"),
        description: t(
          "settings.serverPanel.security.rateLimitEnabled.description",
        ),
        type: "bool",
      },
      {
        key: "security.rate_limit.auth_per_min",
        label: t("settings.serverPanel.security.rateLimitAuthPerMin.label"),
        description: t(
          "settings.serverPanel.security.rateLimitAuthPerMin.description",
        ),
        type: "int",
      },
      {
        key: "security.rate_limit.api_per_min",
        label: t("settings.serverPanel.security.rateLimitApiPerMin.label"),
        description: t(
          "settings.serverPanel.security.rateLimitApiPerMin.description",
        ),
        type: "int",
      },
      {
        key: "security.rate_limit.trust_proxy",
        label: t("settings.serverPanel.security.rateLimitTrustProxy.label"),
        description: t(
          "settings.serverPanel.security.rateLimitTrustProxy.description",
        ),
        type: "bool",
      },
      {
        key: "security.max_refresh_tokens_per_user",
        label: t("settings.serverPanel.security.maxRefreshTokens.label"),
        description: t(
          "settings.serverPanel.security.maxRefreshTokens.description",
        ),
        type: "int",
      },
      {
        key: "security.master_key_path",
        label: t("settings.serverPanel.security.masterKeyPath.label"),
        description: t(
          "settings.serverPanel.security.masterKeyPath.description",
        ),
        type: "string",
      },
    ],
  },
];

// Component for rendering a single setting input
const SettingInput: Component<{
  setting: ServerSetting | undefined;
  config: SettingConfig;
  disabled: boolean;
  updating: boolean;
  onUpdate: (key: string, value: string | number | boolean) => void;
}> = (props) => {
  const value = () => props.setting?.value ?? "";
  const [editValue, setEditValue] = createSignal<string>("");
  const [isEditing, setIsEditing] = createSignal(false);
  const [hasChanges, setHasChanges] = createSignal(false);
  const [showSecret, setShowSecret] = createSignal(false);
  const [revealedValue, setRevealedValue] = createSignal<string | null>(null);
  const [loadingReveal, setLoadingReveal] = createSignal(false);

  // Check if value is sensitive (masked) - detected from server response
  const isSensitive = () => String(value()) === "********";

  // Toggle secret visibility - fetch revealed value from server
  const toggleSecretVisibility = async () => {
    if (showSecret()) {
      // Hide the secret
      setShowSecret(false);
      setRevealedValue(null);
    } else {
      // Fetch and show the secret
      setLoadingReveal(true);
      const result = await getServerSetting(props.config.key, true);
      setLoadingReveal(false);
      if (result.success) {
        setRevealedValue(String(result.setting.value));
        setShowSecret(true);
      }
    }
  };

  // Initialize edit value when entering edit mode
  const startEditing = () => {
    if (props.disabled || props.updating) return;
    // For sensitive fields, start with empty value since we can't show the real one
    const initialValue = isSensitive() ? "" : String(value());
    setEditValue(initialValue);
    setIsEditing(true);
    setHasChanges(false);
  };

  // Handle input change
  const handleInputChange = (newValue: string) => {
    setEditValue(newValue);
    const originalValue = isSensitive() ? "" : String(value());
    setHasChanges(newValue !== originalValue);
  };

  // Submit the value
  const submitValue = () => {
    if (!hasChanges() && !isSensitive()) {
      setIsEditing(false);
      return;
    }

    // For sensitive fields, only submit if user entered something
    if (isSensitive() && editValue() === "") {
      setIsEditing(false);
      return;
    }

    const finalValue =
      props.config.type === "int" ? parseInt(editValue(), 10) : editValue();

    // Validate int type
    if (props.config.type === "int" && isNaN(finalValue as number)) {
      setIsEditing(false);
      return;
    }

    props.onUpdate(props.config.key, finalValue);
    setIsEditing(false);
    setHasChanges(false);
  };

  // Cancel editing
  const cancelEditing = () => {
    setIsEditing(false);
    setHasChanges(false);
  };

  // Handle keyboard events
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      submitValue();
    } else if (e.key === "Escape") {
      e.preventDefault();
      cancelEditing();
    }
  };

  return (
    <article class="flex items-center justify-between py-3 pl-8 pr-4 gap-4">
      <section class="flex flex-col min-w-0 flex-1">
        <span class="text-sm font-medium">{props.config.label}</span>
        <Show when={props.config.description}>
          <span class="text-xs text-muted-foreground">
            {props.config.description}
          </span>
        </Show>
      </section>
      <section class="shrink-0 flex items-center gap-2">
        {/* Boolean toggle */}
        <Show when={props.config.type === "bool"}>
          <button
            type="button"
            role="switch"
            aria-checked={value() === true}
            disabled={props.disabled || props.updating}
            onClick={() => props.onUpdate(props.config.key, value() !== true)}
            class={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 ${
              value() === true ? "bg-primary" : "bg-muted"
            }`}
          >
            <span
              class={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-background shadow-lg ring-0 transition-transform ${
                value() === true ? "translate-x-4" : "translate-x-0"
              }`}
            />
          </button>
        </Show>

        {/* Select dropdown */}
        <Show when={props.config.type === "string" && props.config.options}>
          <select
            value={String(value())}
            onChange={(e) => props.onUpdate(props.config.key, e.target.value)}
            disabled={props.disabled || props.updating}
            class="px-2 py-1 text-sm border border-border rounded bg-background text-foreground cursor-pointer hover:bg-muted/50 focus:outline-none focus:ring-2 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
          >
            <For each={props.config.options}>
              {(option) => <option value={option.value}>{option.label}</option>}
            </For>
          </select>
        </Show>

        {/* Editable string input */}
        <Show when={props.config.type === "string" && !props.config.options}>
          <Show
            when={isEditing()}
            fallback={
              <section class="flex items-center gap-1">
                <button
                  type="button"
                  onClick={startEditing}
                  disabled={props.disabled || props.updating}
                  class="text-sm font-mono max-w-56 truncate block px-2 py-1 border border-transparent rounded hover:border-border hover:bg-muted/50 transition-colors text-left disabled:cursor-not-allowed disabled:opacity-50"
                  title={isSensitive() ? "Click to edit" : String(value())}
                >
                  {props.updating ? (
                    <Icon name="spinner" size="xs" class="animate-spin" />
                  ) : isSensitive() ? (
                    showSecret() && revealedValue() ? (
                      <span class="font-mono">{revealedValue()}</span>
                    ) : (
                      <span class="text-muted-foreground">••••••••</span>
                    )
                  ) : (
                    String(value()) || (
                      <span class="text-muted-foreground">—</span>
                    )
                  )}
                </button>
                <Show
                  when={
                    isSensitive() && String(value()) && String(value()) !== ""
                  }
                >
                  <button
                    type="button"
                    onClick={toggleSecretVisibility}
                    disabled={loadingReveal()}
                    class="p-1 text-muted-foreground hover:text-foreground hover:bg-muted rounded transition-colors disabled:opacity-50"
                    title={showSecret() ? "Hide value" : "Show value"}
                  >
                    {loadingReveal() ? (
                      <Icon name="spinner" size="xs" class="animate-spin" />
                    ) : (
                      <Icon
                        name={showSecret() ? "eye-slash" : "eye"}
                        size="xs"
                      />
                    )}
                  </button>
                </Show>
              </section>
            }
          >
            <section class="flex items-center gap-1">
              <input
                type={isSensitive() && !showSecret() ? "password" : "text"}
                value={editValue()}
                onInput={(e) => handleInputChange(e.currentTarget.value)}
                onKeyDown={handleKeyDown}
                onBlur={submitValue}
                autofocus
                placeholder={isSensitive() ? "Enter new value" : ""}
                class="w-48 px-2 py-1 text-sm font-mono border border-primary rounded bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
              <Show when={isSensitive()}>
                <button
                  type="button"
                  onClick={() => setShowSecret(!showSecret())}
                  class="p-1 text-muted-foreground hover:text-foreground hover:bg-muted rounded transition-colors"
                  title={showSecret() ? "Hide value" : "Show value"}
                >
                  <Icon name={showSecret() ? "eye-slash" : "eye"} size="xs" />
                </button>
              </Show>
              <Show when={hasChanges() || isSensitive()}>
                <button
                  type="button"
                  onClick={submitValue}
                  class="p-1 text-primary hover:bg-primary/10 rounded transition-colors"
                  title="Save"
                >
                  <Icon name="check" size="xs" />
                </button>
              </Show>
              <button
                type="button"
                onClick={cancelEditing}
                class="p-1 text-muted-foreground hover:bg-muted rounded transition-colors"
                title="Cancel"
              >
                <Icon name="x" size="xs" />
              </button>
            </section>
          </Show>
        </Show>

        {/* Editable int input */}
        <Show when={props.config.type === "int"}>
          <Show
            when={isEditing()}
            fallback={
              <button
                type="button"
                onClick={startEditing}
                disabled={props.disabled || props.updating}
                class="text-sm font-mono px-2 py-1 border border-transparent rounded hover:border-border hover:bg-muted/50 transition-colors text-left disabled:cursor-not-allowed disabled:opacity-50"
                title="Click to edit"
              >
                {props.updating ? (
                  <Icon name="spinner" size="xs" class="animate-spin" />
                ) : (
                  String(value())
                )}
              </button>
            }
          >
            <section class="flex items-center gap-1">
              <input
                type="number"
                value={editValue()}
                onInput={(e) => handleInputChange(e.currentTarget.value)}
                onKeyDown={handleKeyDown}
                onBlur={submitValue}
                autofocus
                class="w-24 px-2 py-1 text-sm font-mono border border-primary rounded bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
              <Show when={hasChanges()}>
                <button
                  type="button"
                  onClick={submitValue}
                  class="p-1 text-primary hover:bg-primary/10 rounded transition-colors"
                  title="Save"
                >
                  <Icon name="check" size="xs" />
                </button>
              </Show>
              <button
                type="button"
                onClick={cancelEditing}
                class="p-1 text-muted-foreground hover:bg-muted rounded transition-colors"
                title="Cancel"
              >
                <Icon name="x" size="xs" />
              </button>
            </section>
          </Show>
        </Show>
      </section>
    </article>
  );
};

// Component for a setting group with optional variants
const SettingGroup: Component<{
  group: SettingGroupConfig;
  settings: ServerSetting[];
  updatingKeys: Set<string>;
  onUpdate: (key: string, value: string | number | boolean) => void;
  onResetDatabase?: () => void;
}> = (props) => {
  const [expanded, setExpanded] = createSignal(true);

  // Get the setting value for a key
  const getSetting = (key: string) => props.settings.find((s) => s.key === key);

  // For storage, get the current type
  const currentStorageType = createMemo(() => {
    if (props.group.key === "storage") {
      const typeSetting = getSetting("storage.type");
      return (typeSetting?.value as string) || "local";
    }
    return null;
  });

  // Check if group has a reboot required setting
  const hasRebootRequired = createMemo(() => {
    const checkSettings = (configs: SettingConfig[] | undefined): boolean => {
      if (!configs) return false;
      return configs.some((c) => {
        const setting = getSetting(c.key);
        return setting?.rebootRequired;
      });
    };

    if (props.group.children) {
      return checkSettings(props.group.children);
    }
    if (props.group.variants) {
      return props.group.variants.some((v) => checkSettings(v.children));
    }
    return false;
  });

  return (
    <article class="border border-border rounded-lg overflow-hidden">
      {/* Group Header */}
      <header
        class="flex items-center gap-3 px-4 py-3 bg-muted/30 cursor-pointer hover:bg-muted/50 transition-colors"
        onClick={() => setExpanded(!expanded())}
      >
        <Icon
          name={expanded() ? "caret-down" : "caret-right"}
          size="sm"
          class="text-muted-foreground"
        />
        <Icon name={props.group.icon} size="sm" class="text-primary" />
        <section class="flex-1 min-w-0">
          <h3 class="font-medium">{props.group.label}</h3>
          <p class="text-xs text-muted-foreground">{props.group.description}</p>
        </section>
        <Show when={hasRebootRequired()}>
          <span class="text-xs text-amber-500 flex items-center gap-1">
            <Icon name="warning" size="xs" />
            {t("settings.serverPanel.requiresRestart")}
          </span>
        </Show>
      </header>

      {/* Group Content */}
      <Show when={expanded()}>
        <section class="divide-y divide-border">
          {/* Regular children settings */}
          <Show when={props.group.children}>
            <For each={props.group.children}>
              {(config) => (
                <SettingInput
                  setting={getSetting(config.key)}
                  config={config}
                  disabled={false}
                  updating={props.updatingKeys.has(config.key)}
                  onUpdate={props.onUpdate}
                />
              )}
            </For>
          </Show>

          {/* Database reset button - only shown for database group */}
          <Show when={props.group.key === "database" && props.onResetDatabase}>
            <article class="flex items-center justify-between py-3 pl-8 pr-4 gap-4 border-t border-border">
              <section class="flex flex-col min-w-0 flex-1">
                <span class="text-sm font-medium text-destructive">
                  {t("settings.serverPanel.resetDatabase.title")}
                </span>
                <span class="text-xs text-muted-foreground">
                  {t("settings.serverPanel.resetDatabase.description")}
                </span>
              </section>
              <button
                type="button"
                onClick={props.onResetDatabase}
                class="px-3 py-1.5 text-sm text-destructive border border-destructive rounded hover:bg-destructive/10 transition-colors"
              >
                {t("common.actions.reset")}
              </button>
            </article>
          </Show>

          {/* Variant-based settings (like storage) */}
          <Show when={props.group.variants}>
            <For each={props.group.variants}>
              {(variant) => {
                const isActive = () => currentStorageType() === variant.key;

                return (
                  <section
                    class={`transition-opacity ${isActive() ? "" : "opacity-50"}`}
                  >
                    {/* Variant header with radio-like selection */}
                    <article class="flex items-center gap-3 pl-4 pr-4 py-3 border-b border-border/50">
                      <button
                        type="button"
                        onClick={() =>
                          props.onUpdate("storage.type", variant.key)
                        }
                        disabled={props.updatingKeys.has("storage.type")}
                        class={`w-4 h-4 rounded-full border-2 flex items-center justify-center transition-colors ${
                          isActive()
                            ? "border-primary bg-primary"
                            : "border-muted-foreground hover:border-primary"
                        } disabled:cursor-not-allowed disabled:opacity-50`}
                      >
                        <Show when={isActive()}>
                          <span class="w-1.5 h-1.5 rounded-full bg-background" />
                        </Show>
                      </button>
                      <section class="flex-1">
                        <span class="text-sm font-medium">{variant.label}</span>
                        <Show when={isActive()}>
                          <span class="ml-2 text-xs text-primary">
                            (active)
                          </span>
                        </Show>
                      </section>
                      <Show when={!isActive()}>
                        <span class="text-xs text-muted-foreground">
                          Click to enable
                        </span>
                      </Show>
                    </article>

                    {/* Variant children - only interactive when active */}
                    <Show when={isActive()}>
                      <For each={variant.children}>
                        {(config) => (
                          <SettingInput
                            setting={getSetting(config.key)}
                            config={config}
                            disabled={!isActive()}
                            updating={props.updatingKeys.has(config.key)}
                            onUpdate={props.onUpdate}
                          />
                        )}
                      </For>
                    </Show>
                  </section>
                );
              }}
            </For>
          </Show>
        </section>
      </Show>
    </article>
  );
};

export const ServerSettingsPanel: Component<ServerSettingsPanelProps> = (
  props,
) => {
  return (
    <section class="flex flex-col gap-4">
      <Show when={props.loading}>
        <article class="flex items-center justify-center py-8 text-muted-foreground">
          <Icon name="spinner" size="md" class="animate-spin mr-2" />
          <span>{t("settings.serverPanel.loading")}</span>
        </article>
      </Show>

      <Show when={props.error}>
        <article class="flex flex-col items-center justify-center py-8 gap-3">
          <Icon name="warning" size="lg" class="text-destructive" />
          <p class="text-muted-foreground">{props.error}</p>
          <button
            onClick={props.onRetry}
            class="px-4 py-2 text-sm border border-border rounded hover:bg-muted/50 transition-colors"
          >
            {t("common.actions.retry")}
          </button>
        </article>
      </Show>

      <Show when={!props.loading && !props.error}>
        <For each={getSettingsStructure()}>
          {(group) => (
            <SettingGroup
              group={group}
              settings={props.settings}
              updatingKeys={props.updatingKeys}
              onUpdate={props.onUpdate}
              onResetDatabase={props.onResetDatabase}
            />
          )}
        </For>
      </Show>
    </section>
  );
};
