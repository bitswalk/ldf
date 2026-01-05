import type { Component } from "solid-js";
import { createSignal, Show, For, onMount, createResource } from "solid-js";
import { t } from "../../services/i18n";
import { debugLog } from "../../lib/utils";
import { listSources } from "../../services/sources";
import {
  listSourceVersions,
  type SourceVersion,
  type SourceType,
} from "../../services/sourceVersions";
import {
  listComponents,
  groupByCategory,
  type Component as LDFComponent,
} from "../../services/components";
import {
  SearchableSelect,
  type SearchableSelectOption,
} from "../SearchableSelect";
import { Spinner } from "../Spinner";
import type {
  Distribution,
  DistributionConfig,
  UpdateDistributionRequest,
} from "../../services/distribution";

interface DistributionEditFormProps {
  distribution: Distribution;
  onSubmit: (data: UpdateDistributionRequest) => void;
  onCancel: () => void;
  isSubmitting?: boolean;
}

// Component option structure
interface ComponentOption {
  id: string;
  name: string;
  description?: string;
}

// Components grouped by category
interface ComponentsByCategory {
  bootloader: ComponentOption[];
  init: ComponentOption[];
  filesystem: ComponentOption[];
  security: ComponentOption[];
  container: ComponentOption[];
  virtualization: ComponentOption[];
  desktop: ComponentOption[];
  // Static options
  partitioningTypes: ComponentOption[];
  partitioningModes: ComponentOption[];
  filesystemHierarchies: ComponentOption[];
  packageManagers: ComponentOption[];
  distributionTypes: ComponentOption[];
  visibilities: ComponentOption[];
}

// Static options for items that don't come from component database
const staticOptions = {
  partitioningTypes: [
    { id: "a-b", name: "A/B Partitioning" },
    { id: "single", name: "Single Partition" },
  ],
  partitioningModes: [
    { id: "lvm", name: "LVM" },
    { id: "raw", name: "Raw" },
  ],
  filesystemHierarchies: [
    { id: "fhs", name: "FHS" },
    { id: "custom", name: "Custom" },
  ],
  packageManagers: [
    { id: "apt-deb", name: "APT/DEB" },
    { id: "rpm-dnf5", name: "RPM/DNF5" },
    { id: "none", name: "None" },
  ],
  distributionTypes: [
    { id: "desktop", name: "Desktop" },
    { id: "server", name: "Server" },
  ],
  visibilities: [
    { id: "public", name: "Public" },
    { id: "private", name: "Private" },
  ],
};

// Function to fetch and group components by category
async function fetchComponentOptions(): Promise<ComponentsByCategory> {
  const defaultOptions: ComponentsByCategory = {
    bootloader: [],
    init: [],
    filesystem: [],
    security: [],
    container: [],
    virtualization: [],
    desktop: [],
    ...staticOptions,
  };

  try {
    const result = await listComponents();
    if (!result.success) {
      debugLog("Failed to fetch components:", result.message);
      return defaultOptions;
    }

    const grouped = groupByCategory(result.components);

    // Map components to option format
    const mapToOptions = (
      components: LDFComponent[] | undefined,
    ): ComponentOption[] => {
      if (!components) return [];
      return components.map((c) => ({
        id: c.name,
        name: c.display_name,
        description: c.description,
      }));
    };

    // Add "none" option helper
    const withNoneOption = (options: ComponentOption[]): ComponentOption[] => {
      return [...options, { id: "none", name: "None" }];
    };

    return {
      bootloader: mapToOptions(grouped["bootloader"]),
      init: mapToOptions(grouped["init"]),
      filesystem: mapToOptions(grouped["filesystem"]),
      security: withNoneOption(mapToOptions(grouped["security"])),
      container: withNoneOption(mapToOptions(grouped["container"])),
      virtualization: withNoneOption(mapToOptions(grouped["virtualization"])),
      desktop: mapToOptions(grouped["desktop"]),
      ...staticOptions,
    };
  } catch (err) {
    debugLog("Error fetching components:", err);
    return defaultOptions;
  }
}

// Kernel version type
interface KernelVersion {
  version: string;
  type: "stable" | "lts" | "mainline";
  sourceId: string;
  sourceType: SourceType;
}

// Function to fetch kernel versions from synced sources
async function fetchKernelVersions(): Promise<KernelVersion[]> {
  try {
    const componentsResult = await listComponents();
    if (!componentsResult.success) {
      return [];
    }

    const kernelComponent = componentsResult.components.find(
      (c) => c.name === "kernel" && c.category === "core",
    );
    if (!kernelComponent) {
      return [];
    }

    const sourcesResult = await listSources();
    if (!sourcesResult.success) {
      return [];
    }

    const kernelSources = sourcesResult.sources.filter(
      (s) => s.component_ids.includes(kernelComponent.id) && s.enabled,
    );

    if (kernelSources.length === 0) {
      return [];
    }

    const allVersions: KernelVersion[] = [];
    for (const source of kernelSources) {
      const sourceType: SourceType = source.is_system ? "default" : "user";
      const pageSize = 100;
      let offset = 0;
      let hasMore = true;

      while (hasMore) {
        const versionsResult = await listSourceVersions(
          source.id,
          sourceType,
          pageSize,
          offset,
          undefined,
        );

        if (versionsResult.success) {
          for (const v of versionsResult.versions) {
            let displayType: "stable" | "lts" | "mainline" = "stable";
            if (
              v.version_type === "mainline" ||
              v.version_type === "linux-next"
            ) {
              displayType = "mainline";
            } else if (v.version_type === "longterm") {
              displayType = "lts";
            } else {
              displayType = "stable";
            }

            allVersions.push({
              version: v.version,
              type: displayType,
              sourceId: source.id,
              sourceType,
            });
          }
          hasMore = versionsResult.versions.length === pageSize;
          offset += pageSize;
        } else {
          hasMore = false;
        }
      }
    }

    allVersions.sort((a, b) => {
      const partsA = a.version.split(".").map((p) => parseInt(p, 10) || 0);
      const partsB = b.version.split(".").map((p) => parseInt(p, 10) || 0);
      for (let i = 0; i < Math.max(partsA.length, partsB.length); i++) {
        const numA = partsA[i] || 0;
        const numB = partsB[i] || 0;
        if (numA !== numB) return numB - numA;
      }
      return 0;
    });

    const uniqueVersions = allVersions.filter(
      (v, i, arr) => arr.findIndex((x) => x.version === v.version) === i,
    );

    return uniqueVersions;
  } catch (err) {
    return [];
  }
}

export const DistributionEditForm: Component<DistributionEditFormProps> = (
  props,
) => {
  const [kernelVersions] = createResource(fetchKernelVersions);
  const [componentOptions] = createResource(fetchComponentOptions);
  const [name, setName] = createSignal(props.distribution.name);
  const [visibility, setVisibility] = createSignal(
    props.distribution.visibility,
  );
  const [config, setConfig] = createSignal<DistributionConfig>(
    props.distribution.config
      ? JSON.parse(JSON.stringify(props.distribution.config))
      : createDefaultConfig(),
  );

  function createDefaultConfig(): DistributionConfig {
    return {
      core: {
        kernel: { version: "" },
        bootloader: "systemd-boot",
        partitioning: { type: "a-b", mode: "lvm" },
      },
      system: {
        init: "systemd",
        filesystem: { type: "btrfs", hierarchy: "fhs" },
        packageManager: "apt-deb",
      },
      security: { system: "selinux" },
      runtime: {
        container: "docker-podman",
        virtualization: "cloud-hypervisor",
      },
      target: { type: "desktop" },
    };
  }

  const updateConfig = (path: string, value: string) => {
    const newConfig = JSON.parse(JSON.stringify(config()));
    const parts = path.split(".");
    let current: any = newConfig;
    for (let i = 0; i < parts.length - 1; i++) {
      if (!current[parts[i]]) {
        current[parts[i]] = {};
      }
      current = current[parts[i]];
    }
    current[parts[parts.length - 1]] = value;
    setConfig(newConfig);
  };

  const handleSubmit = (e: Event) => {
    e.preventDefault();

    const updateReq: UpdateDistributionRequest = {
      name: name(),
      visibility: visibility(),
      config: config(),
    };

    props.onSubmit(updateReq);
  };

  const kernelVersionOptions = (): SearchableSelectOption[] => {
    return (kernelVersions() || []).map((v) => ({
      value: v.version,
      label: v.version,
      sublabel: v.type,
    }));
  };

  return (
    <form onSubmit={handleSubmit} class="flex flex-col h-full">
      <div class="flex-1 space-y-6 overflow-y-auto">
        {/* Basic Info Section */}
        <section class="space-y-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.editForm.basicInfo")}
          </h3>

          {/* Name */}
          <div class="space-y-2">
            <label class="text-sm font-medium">
              {t("distribution.form.name.label")}
            </label>
            <input
              type="text"
              class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
              value={name()}
              onInput={(e) => setName(e.target.value)}
            />
          </div>

          {/* Visibility */}
          <div class="space-y-2">
            <label class="text-sm font-medium">
              {t("distribution.table.columns.visibility")}
            </label>
            <div class="grid grid-cols-2 gap-2">
              <For each={componentOptions()?.visibilities || []}>
                {(option) => (
                  <label
                    class={`flex items-center justify-center p-2 border rounded-md cursor-pointer transition-colors ${
                      visibility() === option.id
                        ? "border-primary bg-primary/10"
                        : "border-border hover:bg-muted"
                    }`}
                  >
                    <input
                      type="radio"
                      name="visibility"
                      value={option.id}
                      checked={visibility() === option.id}
                      onChange={(e) =>
                        setVisibility(e.target.value as "public" | "private")
                      }
                      class="sr-only"
                    />
                    <span class="text-sm">
                      {t(`common.visibility.${option.id}`)}
                    </span>
                  </label>
                )}
              </For>
            </div>
          </div>
        </section>

        {/* Core System Section */}
        <section class="space-y-4 border-t border-border pt-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.detail.config.coreSystem")}
          </h3>

          {/* Kernel Version */}
          <div class="space-y-2">
            <label class="text-sm font-medium">
              {t("distribution.detail.config.kernelVersion")}
            </label>
            <SearchableSelect
              value={config().core.kernel.version}
              options={kernelVersionOptions()}
              onChange={(value) => updateConfig("core.kernel.version", value)}
              placeholder={t("distribution.detail.config.selectVersion")}
              searchPlaceholder={t("distribution.detail.config.searchVersions")}
              loading={kernelVersions.loading}
              maxDisplayed={50}
              fullWidth
            />
          </div>

          {/* Bootloader */}
          <div class="space-y-2">
            <label class="text-sm font-medium">
              {t("distribution.detail.config.bootloader")}
            </label>
            <select
              class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
              value={config().core.bootloader}
              onChange={(e) => updateConfig("core.bootloader", e.target.value)}
            >
              <For each={componentOptions()?.bootloader || []}>
                {(opt) => <option value={opt.id}>{opt.name}</option>}
              </For>
            </select>
          </div>

          {/* Partitioning */}
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.detail.config.partitioningType")}
              </label>
              <select
                class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
                value={config().core.partitioning.type}
                onChange={(e) =>
                  updateConfig("core.partitioning.type", e.target.value)
                }
              >
                <For each={componentOptions()?.partitioningTypes || []}>
                  {(opt) => <option value={opt.id}>{opt.name}</option>}
                </For>
              </select>
            </div>
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.detail.config.partitioningMode")}
              </label>
              <select
                class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
                value={config().core.partitioning.mode}
                onChange={(e) =>
                  updateConfig("core.partitioning.mode", e.target.value)
                }
              >
                <For each={componentOptions()?.partitioningModes || []}>
                  {(opt) => <option value={opt.id}>{opt.name}</option>}
                </For>
              </select>
            </div>
          </div>
        </section>

        {/* System Services Section */}
        <section class="space-y-4 border-t border-border pt-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.detail.config.systemServices")}
          </h3>

          {/* Init System */}
          <div class="space-y-2">
            <label class="text-sm font-medium">
              {t("distribution.detail.config.initSystem")}
            </label>
            <select
              class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
              value={config().system.init}
              onChange={(e) => updateConfig("system.init", e.target.value)}
            >
              <For each={componentOptions()?.init || []}>
                {(opt) => <option value={opt.id}>{opt.name}</option>}
              </For>
            </select>
          </div>

          {/* Filesystem */}
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.detail.config.filesystem")}
              </label>
              <select
                class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
                value={config().system.filesystem.type}
                onChange={(e) =>
                  updateConfig("system.filesystem.type", e.target.value)
                }
              >
                <For each={componentOptions()?.filesystem || []}>
                  {(opt) => <option value={opt.id}>{opt.name}</option>}
                </For>
              </select>
            </div>
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.detail.config.filesystemHierarchy")}
              </label>
              <select
                class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
                value={config().system.filesystem.hierarchy}
                onChange={(e) =>
                  updateConfig("system.filesystem.hierarchy", e.target.value)
                }
              >
                <For each={componentOptions()?.filesystemHierarchies || []}>
                  {(opt) => <option value={opt.id}>{opt.name}</option>}
                </For>
              </select>
            </div>
          </div>

          {/* Package Manager */}
          <div class="space-y-2">
            <label class="text-sm font-medium">
              {t("distribution.detail.config.packageManager")}
            </label>
            <select
              class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
              value={config().system.packageManager}
              onChange={(e) =>
                updateConfig("system.packageManager", e.target.value)
              }
            >
              <For each={componentOptions()?.packageManagers || []}>
                {(opt) => <option value={opt.id}>{opt.name}</option>}
              </For>
            </select>
          </div>
        </section>

        {/* Security & Runtime Section */}
        <section class="space-y-4 border-t border-border pt-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.detail.config.securityRuntime")}
          </h3>

          {/* Security System */}
          <div class="space-y-2">
            <label class="text-sm font-medium">
              {t("distribution.detail.config.securitySystem")}
            </label>
            <select
              class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
              value={config().security.system}
              onChange={(e) => updateConfig("security.system", e.target.value)}
            >
              <For each={componentOptions()?.security || []}>
                {(opt) => <option value={opt.id}>{opt.name}</option>}
              </For>
            </select>
          </div>

          {/* Container & Virtualization */}
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.detail.config.containerRuntime")}
              </label>
              <select
                class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
                value={config().runtime.container}
                onChange={(e) =>
                  updateConfig("runtime.container", e.target.value)
                }
              >
                <For each={componentOptions()?.container || []}>
                  {(opt) => <option value={opt.id}>{opt.name}</option>}
                </For>
              </select>
            </div>
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.detail.config.virtualization")}
              </label>
              <select
                class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
                value={config().runtime.virtualization}
                onChange={(e) =>
                  updateConfig("runtime.virtualization", e.target.value)
                }
              >
                <For each={componentOptions()?.virtualization || []}>
                  {(opt) => <option value={opt.id}>{opt.name}</option>}
                </For>
              </select>
            </div>
          </div>
        </section>

        {/* Target Environment Section */}
        <section class="space-y-4 border-t border-border pt-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.detail.config.targetEnvironment")}
          </h3>

          {/* Distribution Type */}
          <div class="space-y-2">
            <label class="text-sm font-medium">
              {t("distribution.detail.config.distributionType")}
            </label>
            <div class="grid grid-cols-2 gap-2">
              <For each={componentOptions()?.distributionTypes || []}>
                {(option) => (
                  <label
                    class={`flex items-center justify-center p-3 border rounded-md cursor-pointer transition-colors ${
                      config().target.type === option.id
                        ? "border-primary bg-primary/10"
                        : "border-border hover:bg-muted"
                    }`}
                  >
                    <input
                      type="radio"
                      name="distributionType"
                      value={option.id}
                      checked={config().target.type === option.id}
                      onChange={(e) => {
                        updateConfig("target.type", e.target.value);
                        if (e.target.value === "server") {
                          // Clear desktop config when switching to server
                          const newConfig = JSON.parse(
                            JSON.stringify(config()),
                          );
                          delete newConfig.target.desktop;
                          setConfig(newConfig);
                        }
                      }}
                      class="sr-only"
                    />
                    <span>{option.name}</span>
                  </label>
                )}
              </For>
            </div>
          </div>

          {/* Desktop Environment (conditional) */}
          <Show when={config().target.type === "desktop"}>
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.detail.config.desktopEnvironment")}
              </label>
              <select
                class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
                value={config().target.desktop?.environment || ""}
                onChange={(e) => {
                  const newConfig = JSON.parse(JSON.stringify(config()));
                  if (!newConfig.target.desktop) {
                    newConfig.target.desktop = {
                      environment: e.target.value,
                      displayServer: "wayland",
                    };
                  } else {
                    newConfig.target.desktop.environment = e.target.value;
                  }
                  setConfig(newConfig);
                }}
              >
                <option value="">
                  {t("distribution.editForm.selectDesktopEnvironment")}
                </option>
                <For each={componentOptions()?.desktop || []}>
                  {(opt) => <option value={opt.id}>{opt.name}</option>}
                </For>
              </select>
              <p class="text-xs text-muted-foreground">
                {t("distribution.form.fields.desktopEnvironment.waylandOnly")}
              </p>
            </div>
          </Show>
        </section>
      </div>

      {/* Form Actions */}
      <div class="flex justify-end gap-3 pt-4 border-t border-border mt-4">
        <button
          type="button"
          onClick={props.onCancel}
          disabled={props.isSubmitting}
          class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
        >
          {t("common.actions.cancel")}
        </button>
        <button
          type="submit"
          disabled={props.isSubmitting}
          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <Show when={props.isSubmitting}>
            <Spinner size="sm" />
          </Show>
          <span>
            {props.isSubmitting
              ? t("distribution.editForm.saving")
              : t("common.actions.save")}
          </span>
        </button>
      </div>
    </form>
  );
};
