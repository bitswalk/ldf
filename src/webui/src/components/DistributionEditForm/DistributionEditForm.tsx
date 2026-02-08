import type { Component } from "solid-js";
import { createSignal, Show, For, createResource } from "solid-js";
import { t } from "../../services/i18n";
import { debugLog } from "../../lib/utils";
import { listSources } from "../../services/sources";
import {
  listSourceVersions,
  type SourceVersion,
} from "../../services/sourceVersions";
import {
  listComponents,
  groupByCategory,
  getComponentVersions,
  resolveVersionRule,
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

interface FetchComponentsResult {
  options: ComponentsByCategory;
  componentMap: Record<string, LDFComponent>;
}

// Static options for items that don't come from component database
const staticOptions = {
  partitioningTypes: [
    {
      id: "a-b",
      name: "A/B Partitioning",
      description: "Dual partition for atomic updates",
    },
    {
      id: "single",
      name: "Single Partition",
      description: "Traditional single partition layout",
    },
  ],
  partitioningModes: [
    { id: "lvm", name: "LVM" },
    { id: "raw", name: "Raw" },
  ],
  filesystemHierarchies: [
    {
      id: "fhs",
      name: "FHS (Filesystem Hierarchy Standard)",
      description: "Standard Linux directory structure",
    },
    {
      id: "custom",
      name: "Custom",
      description: "Define your own directory structure",
    },
  ],
  packageManagers: [
    { id: "apt-deb", name: "APT/DEB (Debian-based)" },
    { id: "rpm-dnf5", name: "RPM/DNF5 (Red Hat-based)" },
    { id: "none", name: "None (Immutable system)" },
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
async function fetchComponentOptions(): Promise<FetchComponentsResult> {
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
      return { options: defaultOptions, componentMap: {} };
    }

    const grouped = groupByCategory(result.components);

    // Build component map by name for version lookups
    const componentMap: Record<string, LDFComponent> = {};
    for (const component of result.components) {
      componentMap[component.name] = component;
    }

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
      options: {
        bootloader: mapToOptions(grouped["bootloader"]),
        init: mapToOptions(grouped["init"]),
        filesystem: mapToOptions(grouped["filesystem"]),
        security: withNoneOption(mapToOptions(grouped["security"])),
        container: withNoneOption(mapToOptions(grouped["container"])),
        virtualization: withNoneOption(mapToOptions(grouped["virtualization"])),
        desktop: mapToOptions(grouped["desktop"]),
        ...staticOptions,
      },
      componentMap,
    };
  } catch (err) {
    debugLog("Error fetching components:", err);
    return { options: defaultOptions, componentMap: {} };
  }
}

// Kernel version type
interface KernelVersion {
  version: string;
  type: "stable" | "lts" | "mainline";
  sourceId: string;
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
      const pageSize = 100;
      let offset = 0;
      let hasMore = true;

      while (hasMore) {
        const versionsResult = await listSourceVersions(
          source.id,
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
  const [componentsData] = createResource(fetchComponentOptions);
  const [name, setName] = createSignal(props.distribution.name);
  const [visibility, setVisibility] = createSignal(
    props.distribution.visibility,
  );
  const [config, setConfig] = createSignal<DistributionConfig>(
    props.distribution.config
      ? JSON.parse(JSON.stringify(props.distribution.config))
      : createDefaultConfig(),
  );

  // Component versions state
  const [componentVersions, setComponentVersions] = createSignal<
    Record<string, SourceVersion[]>
  >({});
  const [loadingVersions, setLoadingVersions] = createSignal<
    Record<string, boolean>
  >({});
  const [resolvedVersions, setResolvedVersions] = createSignal<
    Record<string, string>
  >({});

  const componentOptions = () => componentsData()?.options;
  const componentMap = () => componentsData()?.componentMap || {};

  // Fetch versions for a component
  const fetchVersionsForComponent = async (componentName: string) => {
    const component = componentMap()[componentName];
    if (!component || componentVersions()[componentName]) return;

    setLoadingVersions((prev) => ({ ...prev, [componentName]: true }));

    const result = await getComponentVersions(component.id, { limit: 100 });
    if (result.success) {
      setComponentVersions((prev) => ({
        ...prev,
        [componentName]: result.data.versions,
      }));
    }

    // Also resolve default version
    const rule = component.default_version_rule || "latest-stable";
    if (rule === "pinned" && component.default_version) {
      setResolvedVersions((prev) => ({
        ...prev,
        [componentName]: component.default_version!,
      }));
    } else {
      const resolveResult = await resolveVersionRule(component.id, rule);
      if (resolveResult.success && resolveResult.data.resolved_version) {
        setResolvedVersions((prev) => ({
          ...prev,
          [componentName]: resolveResult.data.resolved_version,
        }));
      }
    }

    setLoadingVersions((prev) => ({ ...prev, [componentName]: false }));
  };

  // Check if component has versions
  const componentHasVersions = (componentName: string): boolean => {
    return (componentVersions()[componentName]?.length ?? 0) > 0;
  };

  // Get version options for SearchableSelect
  const getVersionOptions = (
    componentName: string,
  ): SearchableSelectOption[] => {
    const versions = componentVersions()[componentName] || [];
    return versions.map((v) => ({
      value: v.version,
      label: v.version,
      sublabel: v.version_type,
    }));
  };

  // Get placeholder text for version selector
  const getVersionPlaceholder = (componentName: string): string => {
    if (loadingVersions()[componentName]) {
      return t("distribution.form.version.loading");
    }
    if (!componentHasVersions(componentName)) {
      return t("distribution.form.version.notSynced");
    }
    return (
      resolvedVersions()[componentName] || t("distribution.form.version.select")
    );
  };

  function createDefaultConfig(): DistributionConfig {
    return {
      core: {
        kernel: { version: "" },
        bootloader: "systemd-boot",
        toolchain: "gcc",
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
    <form onSubmit={handleSubmit} class="flex flex-col">
      <section class="space-y-6">
        {/* Basic Info Section */}
        <section class="space-y-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.editForm.basicInfo")}
          </h3>

          {/* Name */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.form.name.label")}
            </legend>
            <input
              type="text"
              class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary"
              value={name()}
              onInput={(e) => setName(e.target.value)}
            />
          </fieldset>

          {/* Visibility */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.table.columns.visibility")}
            </legend>
            <section class="grid grid-cols-2 gap-2">
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
            </section>
          </fieldset>
        </section>

        {/* Core System Section */}
        <section class="space-y-4 border-t border-border pt-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.detail.config.coreSystem")}
          </h3>

          {/* Kernel Version */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.kernelVersion")}
            </legend>
            <article class="flex items-stretch border border-border rounded-md hover:border-muted-foreground transition-colors">
              <header class="flex-1 flex items-center p-3">
                <strong class="font-medium">Linux Kernel</strong>
              </header>
              <aside class="w-52 border-l border-border">
                <SearchableSelect
                  value={config().core.kernel.version}
                  options={kernelVersionOptions()}
                  onChange={(value) =>
                    updateConfig("core.kernel.version", value)
                  }
                  placeholder={
                    kernelVersions.loading
                      ? t("distribution.form.version.loading")
                      : kernelVersions()?.length === 0
                        ? t("distribution.form.version.notSynced")
                        : t("distribution.form.version.select")
                  }
                  searchPlaceholder={t(
                    "distribution.detail.config.searchVersions",
                  )}
                  loading={kernelVersions.loading}
                  maxDisplayed={50}
                  fullWidth
                  seamless
                />
              </aside>
            </article>
          </fieldset>

          {/* Bootloader */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.bootloader")}
            </legend>
            <section class="grid grid-cols-1 gap-2">
              <For each={componentOptions()?.bootloader || []}>
                {(option) => (
                  <article
                    class={`flex items-stretch border rounded-md transition-colors ${
                      config().core.bootloader === option.id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground"
                    }`}
                  >
                    <label class="flex-1 flex items-start p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                      <input
                        type="radio"
                        name="bootloader"
                        value={option.id}
                        checked={config().core.bootloader === option.id}
                        onChange={(e) => {
                          updateConfig("core.bootloader", e.target.value);
                          fetchVersionsForComponent(e.target.value);
                        }}
                        class="mr-3 mt-0.5"
                      />
                      <section class="flex-1">
                        <strong class="font-medium block">{option.name}</strong>
                        <Show when={option.description}>
                          <p class="text-sm text-muted-foreground">
                            {option.description}
                          </p>
                        </Show>
                      </section>
                    </label>
                    <Show when={config().core.bootloader === option.id}>
                      <aside class="w-52 border-l border-border">
                        <SearchableSelect
                          value={config().core.bootloader_version || ""}
                          options={getVersionOptions(option.id)}
                          onChange={(value) =>
                            updateConfig("core.bootloader_version", value)
                          }
                          placeholder={getVersionPlaceholder(option.id)}
                          searchPlaceholder={t(
                            "distribution.versionModal.searchPlaceholder",
                          )}
                          loading={loadingVersions()[option.id]}
                          maxDisplayed={30}
                          fullWidth
                          seamless
                        />
                      </aside>
                    </Show>
                  </article>
                )}
              </For>
            </section>
          </fieldset>

          {/* Toolchain */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.form.fields.toolchain.label")}
            </legend>
            <section class="grid grid-cols-2 gap-2">
              <label
                class={`flex flex-col p-3 border rounded-md cursor-pointer transition-colors ${
                  (config().core.toolchain || "gcc") === "gcc"
                    ? "border-primary bg-primary/10"
                    : "border-border hover:bg-muted"
                }`}
              >
                <input
                  type="radio"
                  name="toolchain"
                  value="gcc"
                  checked={(config().core.toolchain || "gcc") === "gcc"}
                  onChange={() => updateConfig("core.toolchain", "gcc")}
                  class="sr-only"
                />
                <strong class="font-medium">
                  {t("distribution.form.fields.toolchain.gcc")}
                </strong>
                <p class="text-xs text-muted-foreground mt-1">
                  {t("distribution.form.fields.toolchain.gccDesc")}
                </p>
              </label>
              <label
                class={`flex flex-col p-3 border rounded-md cursor-pointer transition-colors ${
                  config().core.toolchain === "llvm"
                    ? "border-primary bg-primary/10"
                    : "border-border hover:bg-muted"
                }`}
              >
                <input
                  type="radio"
                  name="toolchain"
                  value="llvm"
                  checked={config().core.toolchain === "llvm"}
                  onChange={() => updateConfig("core.toolchain", "llvm")}
                  class="sr-only"
                />
                <strong class="font-medium">
                  {t("distribution.form.fields.toolchain.llvm")}
                </strong>
                <p class="text-xs text-muted-foreground mt-1">
                  {t("distribution.form.fields.toolchain.llvmDesc")}
                </p>
              </label>
            </section>
          </fieldset>

          {/* Partitioning */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.partitioningType")}
            </legend>
            <section class="grid grid-cols-2 gap-2">
              <For each={componentOptions()?.partitioningTypes || []}>
                {(option) => (
                  <label
                    class={`flex items-start p-3 border rounded-md cursor-pointer transition-colors ${
                      config().core.partitioning.type === option.id
                        ? "border-primary bg-primary/10"
                        : "border-border hover:bg-muted"
                    }`}
                  >
                    <input
                      type="radio"
                      name="partitioningType"
                      value={option.id}
                      checked={config().core.partitioning.type === option.id}
                      onChange={(e) =>
                        updateConfig("core.partitioning.type", e.target.value)
                      }
                      class="mr-3 mt-0.5"
                    />
                    <section class="flex-1">
                      <strong class="font-medium block">{option.name}</strong>
                      <Show when={option.description}>
                        <p class="text-sm text-muted-foreground">
                          {option.description}
                        </p>
                      </Show>
                    </section>
                  </label>
                )}
              </For>
            </section>
          </fieldset>

          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.partitioningMode")}
            </legend>
            <section class="grid grid-cols-2 gap-2">
              <For each={componentOptions()?.partitioningModes || []}>
                {(option) => (
                  <label
                    class={`flex items-center p-3 border rounded-md cursor-pointer transition-colors ${
                      config().core.partitioning.mode === option.id
                        ? "border-primary bg-primary/10"
                        : "border-border hover:bg-muted"
                    }`}
                  >
                    <input
                      type="radio"
                      name="partitioningMode"
                      value={option.id}
                      checked={config().core.partitioning.mode === option.id}
                      onChange={(e) =>
                        updateConfig("core.partitioning.mode", e.target.value)
                      }
                      class="mr-3"
                    />
                    <span>{option.name}</span>
                  </label>
                )}
              </For>
            </section>
          </fieldset>
        </section>

        {/* System Services Section */}
        <section class="space-y-4 border-t border-border pt-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.detail.config.systemServices")}
          </h3>

          {/* Init System */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.initSystem")}
            </legend>
            <section class="grid grid-cols-1 gap-2">
              <For each={componentOptions()?.init || []}>
                {(option) => (
                  <article
                    class={`flex items-stretch border rounded-md transition-colors ${
                      config().system.init === option.id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground"
                    }`}
                  >
                    <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                      <input
                        type="radio"
                        name="initSystem"
                        value={option.id}
                        checked={config().system.init === option.id}
                        onChange={(e) => {
                          updateConfig("system.init", e.target.value);
                          fetchVersionsForComponent(e.target.value);
                        }}
                        class="mr-3"
                      />
                      <strong class="font-medium">{option.name}</strong>
                    </label>
                    <Show when={config().system.init === option.id}>
                      <aside class="w-52 border-l border-border">
                        <SearchableSelect
                          value={config().system.init_version || ""}
                          options={getVersionOptions(option.id)}
                          onChange={(value) =>
                            updateConfig("system.init_version", value)
                          }
                          placeholder={getVersionPlaceholder(option.id)}
                          searchPlaceholder={t(
                            "distribution.versionModal.searchPlaceholder",
                          )}
                          loading={loadingVersions()[option.id]}
                          maxDisplayed={30}
                          fullWidth
                          seamless
                        />
                      </aside>
                    </Show>
                  </article>
                )}
              </For>
            </section>
          </fieldset>

          {/* Filesystem */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.filesystem")}
            </legend>
            <section class="grid grid-cols-1 gap-2">
              <For each={componentOptions()?.filesystem || []}>
                {(option) => (
                  <article
                    class={`flex items-stretch border rounded-md transition-colors ${
                      config().system.filesystem.type === option.id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground"
                    }`}
                  >
                    <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                      <input
                        type="radio"
                        name="filesystem"
                        value={option.id}
                        checked={config().system.filesystem.type === option.id}
                        onChange={(e) => {
                          updateConfig(
                            "system.filesystem.type",
                            e.target.value,
                          );
                          fetchVersionsForComponent(e.target.value);
                        }}
                        class="mr-3"
                      />
                      <strong class="font-medium">{option.name}</strong>
                    </label>
                    <Show when={config().system.filesystem.type === option.id}>
                      <aside class="w-52 border-l border-border">
                        <SearchableSelect
                          value={config().system.filesystem_version || ""}
                          options={getVersionOptions(option.id)}
                          onChange={(value) =>
                            updateConfig("system.filesystem_version", value)
                          }
                          placeholder={getVersionPlaceholder(option.id)}
                          searchPlaceholder={t(
                            "distribution.versionModal.searchPlaceholder",
                          )}
                          loading={loadingVersions()[option.id]}
                          maxDisplayed={30}
                          fullWidth
                          seamless
                        />
                      </aside>
                    </Show>
                  </article>
                )}
              </For>
            </section>
          </fieldset>

          {/* Filesystem Hierarchy */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.filesystemHierarchy")}
            </legend>
            <section class="grid grid-cols-1 gap-2">
              <For each={componentOptions()?.filesystemHierarchies || []}>
                {(option) => (
                  <label
                    class={`flex items-start p-3 border rounded-md cursor-pointer transition-colors ${
                      config().system.filesystem.hierarchy === option.id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:bg-muted"
                    }`}
                  >
                    <input
                      type="radio"
                      name="filesystemHierarchy"
                      value={option.id}
                      checked={
                        config().system.filesystem.hierarchy === option.id
                      }
                      onChange={(e) =>
                        updateConfig(
                          "system.filesystem.hierarchy",
                          e.target.value,
                        )
                      }
                      class="mr-3 mt-0.5"
                    />
                    <section class="flex-1">
                      <strong class="font-medium block">{option.name}</strong>
                      <Show when={option.description}>
                        <p class="text-sm text-muted-foreground">
                          {option.description}
                        </p>
                      </Show>
                    </section>
                  </label>
                )}
              </For>
            </section>
          </fieldset>

          {/* Package Manager */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.packageManager")}
            </legend>
            <section class="grid grid-cols-1 gap-2">
              <For each={componentOptions()?.packageManagers || []}>
                {(option) => (
                  <label
                    class={`flex items-center p-3 border rounded-md cursor-pointer transition-colors ${
                      config().system.packageManager === option.id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:bg-muted"
                    }`}
                  >
                    <input
                      type="radio"
                      name="packageManager"
                      value={option.id}
                      checked={config().system.packageManager === option.id}
                      onChange={(e) =>
                        updateConfig("system.packageManager", e.target.value)
                      }
                      class="mr-3"
                    />
                    <span>{option.name}</span>
                  </label>
                )}
              </For>
            </section>
          </fieldset>
        </section>

        {/* Security & Runtime Section */}
        <section class="space-y-4 border-t border-border pt-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.detail.config.securityRuntime")}
          </h3>

          {/* Security System */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.securitySystem")}
            </legend>
            <section class="grid grid-cols-1 gap-2">
              <For each={componentOptions()?.security || []}>
                {(option) => (
                  <article
                    class={`flex items-stretch border rounded-md transition-colors ${
                      config().security.system === option.id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground"
                    }`}
                  >
                    <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                      <input
                        type="radio"
                        name="securitySystem"
                        value={option.id}
                        checked={config().security.system === option.id}
                        onChange={(e) => {
                          updateConfig("security.system", e.target.value);
                          if (e.target.value !== "none") {
                            fetchVersionsForComponent(e.target.value);
                          }
                        }}
                        class="mr-3"
                      />
                      <strong class="font-medium">{option.name}</strong>
                    </label>
                    <Show
                      when={
                        config().security.system === option.id &&
                        option.id !== "none"
                      }
                    >
                      <aside class="w-52 border-l border-border">
                        <SearchableSelect
                          value={config().security.system_version || ""}
                          options={getVersionOptions(option.id)}
                          onChange={(value) =>
                            updateConfig("security.system_version", value)
                          }
                          placeholder={getVersionPlaceholder(option.id)}
                          searchPlaceholder={t(
                            "distribution.versionModal.searchPlaceholder",
                          )}
                          loading={loadingVersions()[option.id]}
                          maxDisplayed={30}
                          fullWidth
                          seamless
                        />
                      </aside>
                    </Show>
                  </article>
                )}
              </For>
            </section>
          </fieldset>

          {/* Container Runtime */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.containerRuntime")}
            </legend>
            <section class="grid grid-cols-1 gap-2">
              <For each={componentOptions()?.container || []}>
                {(option) => (
                  <article
                    class={`flex items-stretch border rounded-md transition-colors ${
                      config().runtime.container === option.id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground"
                    }`}
                  >
                    <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                      <input
                        type="radio"
                        name="containerRuntime"
                        value={option.id}
                        checked={config().runtime.container === option.id}
                        onChange={(e) => {
                          updateConfig("runtime.container", e.target.value);
                          if (e.target.value !== "none") {
                            fetchVersionsForComponent(e.target.value);
                          }
                        }}
                        class="mr-3"
                      />
                      <strong class="font-medium">{option.name}</strong>
                    </label>
                    <Show
                      when={
                        config().runtime.container === option.id &&
                        option.id !== "none"
                      }
                    >
                      <aside class="w-52 border-l border-border">
                        <SearchableSelect
                          value={config().runtime.container_version || ""}
                          options={getVersionOptions(option.id)}
                          onChange={(value) =>
                            updateConfig("runtime.container_version", value)
                          }
                          placeholder={getVersionPlaceholder(option.id)}
                          searchPlaceholder={t(
                            "distribution.versionModal.searchPlaceholder",
                          )}
                          loading={loadingVersions()[option.id]}
                          maxDisplayed={30}
                          fullWidth
                          seamless
                        />
                      </aside>
                    </Show>
                  </article>
                )}
              </For>
            </section>
          </fieldset>

          {/* Virtualization Runtime */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.virtualization")}
            </legend>
            <section class="grid grid-cols-1 gap-2">
              <For each={componentOptions()?.virtualization || []}>
                {(option) => (
                  <article
                    class={`flex items-stretch border rounded-md transition-colors ${
                      config().runtime.virtualization === option.id
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground"
                    }`}
                  >
                    <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                      <input
                        type="radio"
                        name="virtualizationRuntime"
                        value={option.id}
                        checked={config().runtime.virtualization === option.id}
                        onChange={(e) => {
                          updateConfig(
                            "runtime.virtualization",
                            e.target.value,
                          );
                          if (e.target.value !== "none") {
                            fetchVersionsForComponent(e.target.value);
                          }
                        }}
                        class="mr-3"
                      />
                      <strong class="font-medium">{option.name}</strong>
                    </label>
                    <Show
                      when={
                        config().runtime.virtualization === option.id &&
                        option.id !== "none"
                      }
                    >
                      <aside class="w-52 border-l border-border">
                        <SearchableSelect
                          value={config().runtime.virtualization_version || ""}
                          options={getVersionOptions(option.id)}
                          onChange={(value) =>
                            updateConfig(
                              "runtime.virtualization_version",
                              value,
                            )
                          }
                          placeholder={getVersionPlaceholder(option.id)}
                          searchPlaceholder={t(
                            "distribution.versionModal.searchPlaceholder",
                          )}
                          loading={loadingVersions()[option.id]}
                          maxDisplayed={30}
                          fullWidth
                          seamless
                        />
                      </aside>
                    </Show>
                  </article>
                )}
              </For>
            </section>
          </fieldset>
        </section>

        {/* Target Environment Section */}
        <section class="space-y-4 border-t border-border pt-4">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("distribution.detail.config.targetEnvironment")}
          </h3>

          {/* Distribution Type */}
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium">
              {t("distribution.detail.config.distributionType")}
            </legend>
            <section class="grid grid-cols-2 gap-2">
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
            </section>
          </fieldset>

          {/* Desktop Environment (conditional) */}
          <Show when={config().target.type === "desktop"}>
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.detail.config.desktopEnvironment")}
              </legend>
              <p class="text-xs text-muted-foreground mb-2">
                {t("distribution.form.fields.desktopEnvironment.waylandOnly")}
              </p>
              <section class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.desktop || []}>
                  {(option) => (
                    <article
                      class={`flex items-stretch border rounded-md transition-colors ${
                        config().target.desktop?.environment === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:border-muted-foreground"
                      }`}
                    >
                      <label class="flex-1 flex items-start p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                        <input
                          type="radio"
                          name="desktopEnvironment"
                          value={option.id}
                          checked={
                            config().target.desktop?.environment === option.id
                          }
                          onChange={(e) => {
                            const newConfig = JSON.parse(
                              JSON.stringify(config()),
                            );
                            if (!newConfig.target.desktop) {
                              newConfig.target.desktop = {
                                environment: e.target.value,
                                displayServer: "wayland",
                              };
                            } else {
                              newConfig.target.desktop.environment =
                                e.target.value;
                            }
                            setConfig(newConfig);
                            fetchVersionsForComponent(e.target.value);
                          }}
                          class="mr-3 mt-0.5"
                        />
                        <section class="flex-1">
                          <strong class="font-medium block">
                            {option.name}
                          </strong>
                          <Show when={option.description}>
                            <p class="text-sm text-muted-foreground">
                              {option.description}
                            </p>
                          </Show>
                        </section>
                      </label>
                      <Show
                        when={
                          config().target.desktop?.environment === option.id
                        }
                      >
                        <aside class="w-52 border-l border-border">
                          <SearchableSelect
                            value={
                              config().target.desktop?.environment_version || ""
                            }
                            options={getVersionOptions(option.id)}
                            onChange={(value) => {
                              const newConfig = JSON.parse(
                                JSON.stringify(config()),
                              );
                              if (newConfig.target.desktop) {
                                newConfig.target.desktop.environment_version =
                                  value;
                              }
                              setConfig(newConfig);
                            }}
                            placeholder={getVersionPlaceholder(option.id)}
                            searchPlaceholder={t(
                              "distribution.versionModal.searchPlaceholder",
                            )}
                            loading={loadingVersions()[option.id]}
                            maxDisplayed={30}
                            fullWidth
                            seamless
                          />
                        </aside>
                      </Show>
                    </article>
                  )}
                </For>
              </section>
            </fieldset>
          </Show>
        </section>
      </section>

      {/* Form Actions */}
      <nav class="flex justify-end gap-3 pt-4 border-t border-border mt-4">
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
      </nav>
    </form>
  );
};
