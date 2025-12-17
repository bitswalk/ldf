import type { Component } from "solid-js";
import { createSignal, createMemo, Show, For } from "solid-js";
import { Icon } from "../Icon";
import type { ServerSetting } from "../../services/settingsService";

interface ServerSettingsPanelProps {
  settings: ServerSetting[];
  loading: boolean;
  error: string | null;
  updatingKeys: Set<string>;
  onUpdate: (key: string, value: string | number | boolean) => void;
  onRetry: () => void;
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

// Define the settings structure
const settingsStructure: SettingGroupConfig[] = [
  {
    key: "server",
    label: "Server",
    description: "Core server configuration",
    icon: "desktop-tower",
    required: true,
    children: [
      {
        key: "server.bind",
        label: "Bind Address",
        description: "Network address to bind to",
        type: "string",
      },
      {
        key: "server.port",
        label: "Port",
        description: "Port for the server to listen on",
        type: "int",
      },
    ],
  },
  {
    key: "log",
    label: "Logging",
    description: "Logging configuration",
    icon: "note",
    required: false,
    children: [
      {
        key: "log.output",
        label: "Output",
        description: "Where to send log output",
        type: "string",
        options: [
          { value: "auto", label: "Auto" },
          { value: "stdout", label: "Standard Output" },
          { value: "journald", label: "Journald" },
        ],
      },
      {
        key: "log.level",
        label: "Level",
        description: "Minimum log level to display",
        type: "string",
        options: [
          { value: "debug", label: "Debug" },
          { value: "info", label: "Info" },
          { value: "warn", label: "Warning" },
          { value: "error", label: "Error" },
        ],
      },
    ],
  },
  {
    key: "database",
    label: "Database",
    description: "Database persistence settings",
    icon: "database",
    required: true,
    children: [
      {
        key: "database.path",
        label: "Persist Path",
        description: "Path to save database on shutdown",
        type: "string",
      },
    ],
  },
  {
    key: "storage",
    label: "Storage",
    description: "Artifact storage backend",
    icon: "hard-drives",
    required: true,
    variants: [
      {
        key: "local",
        label: "Local Filesystem",
        children: [
          {
            key: "storage.local.path",
            label: "Path",
            description: "Root directory for artifacts",
            type: "string",
          },
        ],
      },
      {
        key: "s3",
        label: "S3 Compatible",
        children: [
          {
            key: "storage.s3.endpoint",
            label: "Endpoint",
            description: "S3-compatible endpoint URL",
            type: "string",
          },
          {
            key: "storage.s3.region",
            label: "Region",
            description: "AWS/S3 region",
            type: "string",
          },
          {
            key: "storage.s3.bucket",
            label: "Bucket",
            description: "Bucket name for artifacts",
            type: "string",
          },
          {
            key: "storage.s3.access_key",
            label: "Access Key",
            description: "S3 access key ID",
            type: "string",
          },
          {
            key: "storage.s3.secret_key",
            label: "Secret Key",
            description: "S3 secret access key",
            type: "string",
          },
          {
            key: "storage.s3.path_style",
            label: "Path Style",
            description: "Use path-style addressing",
            type: "bool",
          },
        ],
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
      <section class="shrink-0">
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
        <Show when={props.config.type === "string" && !props.config.options}>
          <span class="text-sm text-muted-foreground font-mono max-w-48 truncate block">
            {String(value()) || "—"}
          </span>
        </Show>
        <Show when={props.config.type === "int"}>
          <span class="text-sm text-muted-foreground font-mono">
            {String(value())}
          </span>
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
            Requires restart
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
          <span>Loading server settings...</span>
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
            Retry
          </button>
        </article>
      </Show>

      <Show when={!props.loading && !props.error}>
        <For each={settingsStructure}>
          {(group) => (
            <SettingGroup
              group={group}
              settings={props.settings}
              updatingKeys={props.updatingKeys}
              onUpdate={props.onUpdate}
            />
          )}
        </For>
      </Show>
    </section>
  );
};
