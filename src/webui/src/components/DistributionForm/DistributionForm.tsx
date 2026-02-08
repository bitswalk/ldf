import type { Component } from "solid-js";
import {
  createSignal,
  createEffect,
  Show,
  For,
  onMount,
  createResource,
  createMemo,
} from "solid-js";
import { debugLog } from "../../lib/utils";
import { t } from "../../services/i18n";
import { listSources, type Source } from "../../services/sources";
import {
  listSourceVersions,
  type SourceVersion,
} from "../../services/sourceVersions";
import {
  listComponents,
  groupByCategory,
  resolveVersionRule,
  getComponentVersions,
  type Component as LDFComponent,
  type VersionRule,
} from "../../services/components";
import { Icon } from "../Icon";
import {
  SearchableSelect,
  type SearchableSelectOption,
} from "../SearchableSelect";

// Component option structure for form display
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
  // Keep static options for items that don't come from components
  partitioningTypes: ComponentOption[];
  partitioningModes: ComponentOption[];
  filesystemHierarchies: ComponentOption[];
  packageManagers: ComponentOption[];
  distributionTypes: ComponentOption[];
}

// Form data structure (internal state)
interface DistributionFormData {
  name: string;
  kernelVersion: string;
  bootloader: string;
  bootloaderVersion?: string;
  toolchain: string;
  partitioningType: string;
  initSystem: string;
  initSystemVersion?: string;
  filesystem: string;
  filesystemVersion?: string;
  filesystemUserspaceEnabled: boolean; // Include userspace tools for hybrid filesystem components
  partitioning: string;
  filesystemHierarchy: string;
  packageManager: string;
  securitySystem: string;
  securitySystemVersion?: string;
  securitySystemUserspaceEnabled: boolean; // Include userspace tools for hybrid security components
  containerRuntime: string;
  containerRuntimeVersion?: string;
  virtualizationRuntime: string;
  virtualizationRuntimeVersion?: string;
  distributionType: string;
  desktopEnvironment?: string;
  desktopEnvironmentVersion?: string;
}

// Final JSON structure to send to LDF server
interface LDFDistributionConfig {
  name: string;
  core: {
    kernel: {
      version: string;
    };
    bootloader: string;
    bootloader_version?: string;
    toolchain?: string;
    partitioning: {
      type: string;
      mode: string;
    };
  };
  system: {
    init: string;
    init_version?: string;
    filesystem: {
      type: string;
      hierarchy: string;
    };
    filesystem_version?: string;
    filesystem_userspace?: boolean; // Include userspace tools for hybrid filesystem components
    packageManager: string;
  };
  security: {
    system: string;
    system_version?: string;
    system_userspace?: boolean; // Include userspace tools for hybrid security components
  };
  runtime: {
    container: string;
    container_version?: string;
    virtualization: string;
    virtualization_version?: string;
  };
  target: {
    type: string;
    desktop?: {
      environment: string;
      environment_version?: string;
      displayServer: "wayland";
    };
  };
}

interface DistributionFormProps {
  onSubmit: (data: LDFDistributionConfig) => void;
  onCancel: () => void;
}

// Kernel version with stability type derived from source
interface KernelVersion {
  version: string;
  type: "stable" | "lts" | "mainline";
  sourceId: string;
}

// Function to fetch kernel versions from synced sources
async function fetchKernelVersions(): Promise<KernelVersion[]> {
  try {
    // First, get all components to find the kernel component
    const componentsResult = await listComponents();
    if (!componentsResult.success) {
      debugLog("Failed to fetch components:", componentsResult.message);
      return [];
    }

    const kernelComponent = componentsResult.components.find(
      (c) => c.name === "kernel" && c.category === "core",
    );
    if (!kernelComponent) {
      debugLog("Kernel component not found");
      return [];
    }

    // Get all sources (merged defaults + user sources)
    const sourcesResult = await listSources();
    if (!sourcesResult.success) {
      debugLog("Failed to fetch sources:", sourcesResult.message);
      return [];
    }

    // Filter sources that are linked to the kernel component
    const kernelSources = sourcesResult.sources.filter(
      (s) => s.component_ids.includes(kernelComponent.id) && s.enabled,
    );

    if (kernelSources.length === 0) {
      debugLog("No kernel sources found");
      return [];
    }

    // Fetch versions from all kernel sources, paginating to get all versions
    const allVersions: KernelVersion[] = [];
    for (const source of kernelSources) {
      // Paginate through all versions
      const pageSize = 100;
      let offset = 0;
      let hasMore = true;

      while (hasMore) {
        const versionsResult = await listSourceVersions(
          source.id,
          pageSize,
          offset,
          undefined, // No filter - include all version types
        );

        if (versionsResult.success) {
          for (const v of versionsResult.versions) {
            // Use the version_type from the API, map to form display type
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

          // Check if there are more pages
          hasMore = versionsResult.versions.length === pageSize;
          offset += pageSize;
        } else {
          hasMore = false;
        }
      }
    }

    // Sort versions by semantic version (descending)
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

    // Remove duplicates (same version from different sources)
    const uniqueVersions = allVersions.filter(
      (v, i, arr) => arr.findIndex((x) => x.version === v.version) === i,
    );

    return uniqueVersions;
  } catch (err) {
    debugLog("Error fetching kernel versions:", err);
    return [];
  }
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
    {
      id: "desktop",
      name: "Desktop",
      description: "Graphical user interface with Wayland",
    },
    {
      id: "server",
      name: "Server",
      description: "Text-only interface (TTY)",
    },
  ],
};

// Result type for fetching components with full data
interface FetchComponentsResult {
  options: ComponentsByCategory;
  componentMap: Record<string, LDFComponent>;
}

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

  const defaultResult: FetchComponentsResult = {
    options: defaultOptions,
    componentMap: {},
  };

  try {
    const result = await listComponents();
    if (!result.success) {
      debugLog("Failed to fetch components:", result.message);
      return defaultResult;
    }

    const grouped = groupByCategory(result.components);

    // Build a map of component name to full component object
    const componentMap: Record<string, LDFComponent> = {};
    result.components.forEach((c) => {
      componentMap[c.name] = c;
    });

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
    return defaultResult;
  }
}

// Version field mapping type
type VersionFieldKey =
  | "kernelVersion"
  | "bootloaderVersion"
  | "initSystemVersion"
  | "filesystemVersion"
  | "securitySystemVersion"
  | "containerRuntimeVersion"
  | "virtualizationRuntimeVersion"
  | "desktopEnvironmentVersion";

// Mapping from component field to version field
const VERSION_FIELD_MAP: Record<string, VersionFieldKey> = {
  kernel: "kernelVersion",
  bootloader: "bootloaderVersion",
  initSystem: "initSystemVersion",
  filesystem: "filesystemVersion",
  securitySystem: "securitySystemVersion",
  containerRuntime: "containerRuntimeVersion",
  virtualizationRuntime: "virtualizationRuntimeVersion",
  desktopEnvironment: "desktopEnvironmentVersion",
};

export const DistributionForm: Component<DistributionFormProps> = (props) => {
  const [currentStep, setCurrentStep] = createSignal(1);
  const [kernelVersions] = createResource(fetchKernelVersions);
  const [componentsData] = createResource(fetchComponentOptions);
  const [formData, setFormData] = createSignal<DistributionFormData>({
    name: "",
    kernelVersion: "",
    bootloader: "",
    bootloaderVersion: undefined,
    toolchain: "gcc",
    partitioningType: "",
    initSystem: "",
    initSystemVersion: undefined,
    filesystem: "",
    filesystemVersion: undefined,
    filesystemUserspaceEnabled: true, // Default to including userspace tools for hybrid components
    partitioning: "",
    filesystemHierarchy: "",
    packageManager: "",
    securitySystem: "",
    securitySystemVersion: undefined,
    securitySystemUserspaceEnabled: true, // Default to including userspace tools for hybrid components
    containerRuntime: "",
    containerRuntimeVersion: undefined,
    virtualizationRuntime: "",
    virtualizationRuntimeVersion: undefined,
    distributionType: "",
    desktopEnvironment: "",
    desktopEnvironmentVersion: undefined,
  });

  // Component versions state - stores fetched versions per component
  const [componentVersions, setComponentVersions] = createSignal<
    Record<string, SourceVersion[]>
  >({});
  const [loadingVersions, setLoadingVersions] = createSignal<
    Record<string, boolean>
  >({});
  const [resolvedVersions, setResolvedVersions] = createSignal<
    Record<string, string>
  >({});

  // Auto-fill kernel version when versions are loaded
  createEffect(() => {
    const versions = kernelVersions();
    if (versions && versions.length > 0 && !formData().kernelVersion) {
      // Set to the first (latest) version
      setFormData((prev) => ({ ...prev, kernelVersion: versions[0].version }));
    }
  });

  // Helper to get component options
  const componentOptions = () => componentsData()?.options;

  // Helper to get component map
  const componentMap = () => componentsData()?.componentMap || {};

  // Fetch versions for a component and auto-fill with default version
  const fetchVersionsForComponent = async (
    componentName: string,
    versionField?: VersionFieldKey,
  ) => {
    const component = componentMap()[componentName];
    if (!component || componentVersions()[componentName]) {
      // If versions already fetched, still auto-fill if we have a resolved version and field
      if (versionField && resolvedVersions()[componentName]) {
        const currentValue = formData()[versionField];
        if (!currentValue) {
          setFormData((prev) => ({
            ...prev,
            [versionField]: resolvedVersions()[componentName],
          }));
        }
      }
      return;
    }

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
    let resolvedVersion: string | undefined;

    if (rule === "pinned" && component.default_version) {
      resolvedVersion = component.default_version;
      setResolvedVersions((prev) => ({
        ...prev,
        [componentName]: component.default_version!,
      }));
    } else {
      const resolveResult = await resolveVersionRule(component.id, rule);
      if (resolveResult.success && resolveResult.data.resolved_version) {
        resolvedVersion = resolveResult.data.resolved_version;
        setResolvedVersions((prev) => ({
          ...prev,
          [componentName]: resolveResult.data.resolved_version,
        }));
      }
    }

    // Auto-fill the version field with the resolved default version
    if (versionField && resolvedVersion) {
      setFormData((prev) => ({ ...prev, [versionField]: resolvedVersion }));
    }

    setLoadingVersions((prev) => ({ ...prev, [componentName]: false }));
  };

  // Get version options for SearchableSelect
  const getVersionOptions = (
    componentName: string,
  ): SearchableSelectOption[] => {
    const versions = componentVersions()[componentName] || [];
    const resolved = resolvedVersions()[componentName];

    const options: SearchableSelectOption[] = versions.map((v) => ({
      value: v.version,
      label: v.version,
      sublabel: v.version_type === "longterm" ? "LTS" : v.version_type,
    }));

    // If we have a resolved default and it's not in the list, add it
    if (resolved && !versions.find((v) => v.version === resolved)) {
      options.unshift({
        value: resolved,
        label: resolved,
        sublabel: "default",
      });
    }

    return options;
  };

  // Get display value for version selector
  const getVersionDisplayValue = (
    componentName: string,
    selectedVersion: string | undefined,
  ): string => {
    if (selectedVersion) return selectedVersion;
    return resolvedVersions()[componentName] || "";
  };

  // Check if a component has synced versions available
  const componentHasVersions = (componentName: string): boolean => {
    // For kernel, check the kernelVersions resource
    if (componentName === "kernel") {
      return (kernelVersions()?.length ?? 0) > 0;
    }
    // For other components, check fetched versions
    return (componentVersions()[componentName]?.length ?? 0) > 0;
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

  const updateFormData = (
    field: keyof DistributionFormData,
    value: string | boolean,
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const nextStep = () => {
    if (currentStep() < 4) {
      setCurrentStep((prev) => prev + 1);
    }
  };

  const prevStep = () => {
    if (currentStep() > 1) {
      setCurrentStep((prev) => prev - 1);
    }
  };

  const handleSubmit = () => {
    const data = formData();
    const resolved = resolvedVersions();

    // Helper to get version: use selected version or fall back to resolved default
    const getVersion = (
      selectedVersion: string | undefined,
      componentName: string | undefined,
    ): string | undefined => {
      if (selectedVersion) return selectedVersion;
      if (componentName && componentName !== "none") {
        return resolved[componentName] || undefined;
      }
      return undefined;
    };

    // Transform form data into LDF server JSON format
    const ldfConfig: LDFDistributionConfig = {
      name: data.name.trim() || `custom-distribution-${Date.now()}`,
      core: {
        kernel: {
          version: data.kernelVersion || resolved["kernel"] || "",
        },
        bootloader: data.bootloader,
        bootloader_version: getVersion(data.bootloaderVersion, data.bootloader),
        toolchain: data.toolchain !== "gcc" ? data.toolchain : undefined,
        partitioning: {
          type: data.partitioningType,
          mode: data.partitioning,
        },
      },
      system: {
        init: data.initSystem,
        init_version: getVersion(data.initSystemVersion, data.initSystem),
        filesystem: {
          type: data.filesystem,
          hierarchy: data.filesystemHierarchy,
        },
        filesystem_version: getVersion(data.filesystemVersion, data.filesystem),
        filesystem_userspace: data.filesystemUserspaceEnabled,
        packageManager: data.packageManager,
      },
      security: {
        system: data.securitySystem,
        system_version:
          data.securitySystem !== "none"
            ? getVersion(data.securitySystemVersion, data.securitySystem)
            : undefined,
        system_userspace:
          data.securitySystem !== "none"
            ? data.securitySystemUserspaceEnabled
            : undefined,
      },
      runtime: {
        container: data.containerRuntime,
        container_version:
          data.containerRuntime !== "none"
            ? getVersion(data.containerRuntimeVersion, data.containerRuntime)
            : undefined,
        virtualization: data.virtualizationRuntime,
        virtualization_version:
          data.virtualizationRuntime !== "none"
            ? getVersion(
                data.virtualizationRuntimeVersion,
                data.virtualizationRuntime,
              )
            : undefined,
      },
      target: {
        type: data.distributionType,
        ...(data.distributionType === "desktop" && data.desktopEnvironment
          ? {
              desktop: {
                environment: data.desktopEnvironment,
                environment_version: getVersion(
                  data.desktopEnvironmentVersion,
                  data.desktopEnvironment,
                ),
                displayServer: "wayland" as const,
              },
            }
          : {}),
      },
    };

    debugLog("=== LDF Distribution Configuration ===");
    debugLog(JSON.stringify(ldfConfig, null, 2));
    debugLog("======================================");

    props.onSubmit(ldfConfig);
  };

  const isStepValid = (step: number): boolean => {
    const data = formData();
    switch (step) {
      case 1:
        // Allow "none" as valid kernel version when no synced versions are available
        const hasValidKernel =
          data.kernelVersion &&
          (data.kernelVersion === "none" ||
            (kernelVersions()?.some((k) => k.version === data.kernelVersion) ??
              false));
        return !!(hasValidKernel && data.bootloader && data.partitioningType);
      case 2:
        return !!(
          data.initSystem &&
          data.filesystem &&
          data.partitioning &&
          data.filesystemHierarchy &&
          data.packageManager
        );
      case 3:
        return !!(
          data.securitySystem &&
          data.containerRuntime &&
          data.virtualizationRuntime
        );
      case 4:
        if (!data.distributionType) return false;
        if (data.distributionType === "desktop") {
          return !!data.desktopEnvironment;
        }
        return true;
      default:
        return false;
    }
  };

  return (
    <form class="flex flex-col h-full w-full">
      {/* Progress indicator */}
      <section class="flex items-center justify-between mb-6 w-full">
        <For each={[1, 2, 3, 4]}>
          {(step) => (
            <div class="flex items-center flex-1">
              <div
                class={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-semibold ${
                  currentStep() === step
                    ? "bg-primary text-primary-foreground"
                    : currentStep() > step
                      ? "bg-primary/50 text-primary-foreground"
                      : "bg-muted text-muted-foreground"
                }`}
              >
                {step}
              </div>
              <Show when={step < 4}>
                <div
                  class={`flex-1 h-0.5 mx-2 ${
                    currentStep() > step ? "bg-primary" : "bg-muted"
                  }`}
                />
              </Show>
            </div>
          )}
        </For>
      </section>

      {/* Form content */}
      <section class="flex-1 overflow-y-auto mb-6 w-full">
        {/* Step 1: Core System */}
        <Show when={currentStep() === 1}>
          <article class="space-y-6 w-full">
            <header>
              <h3 class="text-lg font-semibold mb-1">
                {t("distribution.form.steps.coreSystem.title")}
              </h3>
              <p class="text-sm text-muted-foreground">
                {t("distribution.form.steps.coreSystem.description")}
              </p>
            </header>

            {/* Distribution Name */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.name.label")}{" "}
                <span class="text-muted-foreground">
                  ({t("distribution.form.name.optional")})
                </span>
              </label>
              <input
                type="text"
                class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary"
                placeholder={t("distribution.form.name.placeholder")}
                value={formData().name}
                onInput={(e) => updateFormData("name", e.target.value)}
              />
              <p class="text-xs text-muted-foreground">
                {t("distribution.form.name.help")}
              </p>
            </div>

            {/* Kernel Version */}
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.form.fields.kernelVersion.label")}
              </legend>
              <article class="flex items-stretch border border-border rounded-md hover:border-muted-foreground transition-colors">
                <header class="flex-1 flex items-center p-3">
                  <strong class="font-medium">Linux Kernel</strong>
                </header>
                <aside class="w-52 border-l border-border">
                  <SearchableSelect
                    value={formData().kernelVersion}
                    options={
                      (kernelVersions() || []).map((kernel) => ({
                        value: kernel.version,
                        label: kernel.version,
                        sublabel: kernel.type,
                      })) as SearchableSelectOption[]
                    }
                    onChange={(value) => updateFormData("kernelVersion", value)}
                    placeholder={
                      kernelVersions.loading
                        ? t("distribution.form.version.loading")
                        : kernelVersions()?.length === 0
                          ? t("distribution.form.version.notSynced")
                          : t("distribution.form.version.select")
                    }
                    searchPlaceholder={t(
                      "distribution.form.fields.kernelVersion.searchPlaceholder",
                    )}
                    loading={kernelVersions.loading}
                    maxDisplayed={50}
                    fullWidth
                    seamless
                  />
                </aside>
              </article>
              <Show
                when={!kernelVersions.loading && kernelVersions()?.length === 0}
              >
                <p class="text-xs text-muted-foreground">
                  {t("distribution.form.fields.kernelVersion.noVersionsHint")}
                </p>
              </Show>
            </fieldset>

            {/* Toolchain */}
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.form.fields.toolchain.label")}
              </legend>
              <section class="grid grid-cols-2 gap-2">
                <For
                  each={[
                    {
                      id: "gcc",
                      name: t("distribution.form.fields.toolchain.gcc"),
                      description: t(
                        "distribution.form.fields.toolchain.gccDesc",
                      ),
                    },
                    {
                      id: "llvm",
                      name: t("distribution.form.fields.toolchain.llvm"),
                      description: t(
                        "distribution.form.fields.toolchain.llvmDesc",
                      ),
                    },
                  ]}
                >
                  {(option) => (
                    <label
                      class={`flex items-start p-3 border rounded-md cursor-pointer transition-colors ${
                        formData().toolchain === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted"
                      }`}
                    >
                      <input
                        type="radio"
                        name="toolchain"
                        value={option.id}
                        checked={formData().toolchain === option.id}
                        onChange={(e) =>
                          updateFormData("toolchain", e.target.value)
                        }
                        class="mr-3 mt-0.5"
                      />
                      <div>
                        <div class="font-medium">{option.name}</div>
                        <div class="text-sm text-muted-foreground">
                          {option.description}
                        </div>
                      </div>
                    </label>
                  )}
                </For>
              </section>
            </fieldset>

            {/* Bootloader */}
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.form.fields.bootloader")}
              </legend>
              <section class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.bootloader || []}>
                  {(bootloader) => (
                    <article
                      class={`flex items-stretch border rounded-md transition-colors ${
                        formData().bootloader === bootloader.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:border-muted-foreground"
                      }`}
                    >
                      <label class="flex-1 flex items-start p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                        <input
                          type="radio"
                          name="bootloader"
                          value={bootloader.id}
                          checked={formData().bootloader === bootloader.id}
                          onChange={(e) => {
                            updateFormData("bootloader", e.target.value);
                            fetchVersionsForComponent(
                              e.target.value,
                              "bootloaderVersion",
                            );
                          }}
                          class="mr-3 mt-0.5"
                        />
                        <section class="flex-1">
                          <strong class="font-medium block">
                            {bootloader.name}
                          </strong>
                          <p class="text-sm text-muted-foreground">
                            {bootloader.description}
                          </p>
                        </section>
                      </label>
                      <Show when={formData().bootloader === bootloader.id}>
                        <aside class="w-52 border-l border-border">
                          <SearchableSelect
                            value={formData().bootloaderVersion || ""}
                            options={getVersionOptions(bootloader.id)}
                            onChange={(value) =>
                              updateFormData("bootloaderVersion", value)
                            }
                            placeholder={getVersionPlaceholder(bootloader.id)}
                            searchPlaceholder={t(
                              "distribution.versionModal.searchPlaceholder",
                            )}
                            loading={loadingVersions()[bootloader.id]}
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

            {/* Partitioning Type */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.partitioningSystem")}
              </label>
              <div class="grid grid-cols-2 gap-2">
                <For each={componentOptions()?.partitioningTypes || []}>
                  {(option) => (
                    <label
                      class={`flex items-start p-3 border rounded-md cursor-pointer transition-colors ${
                        formData().partitioningType === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted"
                      }`}
                    >
                      <input
                        type="radio"
                        name="partitioningType"
                        value={option.id}
                        checked={formData().partitioningType === option.id}
                        onChange={(e) =>
                          updateFormData("partitioningType", e.target.value)
                        }
                        class="mr-3 mt-0.5"
                      />
                      <div>
                        <div class="font-medium">{option.name}</div>
                        <div class="text-sm text-muted-foreground">
                          {option.description}
                        </div>
                      </div>
                    </label>
                  )}
                </For>
              </div>
            </div>
          </article>
        </Show>

        {/* Step 2: System Services */}
        <Show when={currentStep() === 2}>
          <article class="space-y-6 w-full">
            <header>
              <h3 class="text-lg font-semibold mb-1">
                {t("distribution.form.steps.systemServices.title")}
              </h3>
              <p class="text-sm text-muted-foreground">
                {t("distribution.form.steps.systemServices.description")}
              </p>
            </header>

            {/* Init System */}
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.form.fields.initSystem")}
              </legend>
              <section class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.init || []}>
                  {(option) => (
                    <article
                      class={`flex items-stretch border rounded-md transition-colors ${
                        formData().initSystem === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:border-muted-foreground"
                      }`}
                    >
                      <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                        <input
                          type="radio"
                          name="initSystem"
                          value={option.id}
                          checked={formData().initSystem === option.id}
                          onChange={(e) => {
                            updateFormData("initSystem", e.target.value);
                            fetchVersionsForComponent(
                              e.target.value,
                              "initSystemVersion",
                            );
                          }}
                          class="mr-3"
                        />
                        <strong class="font-medium">{option.name}</strong>
                      </label>
                      <Show when={formData().initSystem === option.id}>
                        <aside class="w-52 border-l border-border">
                          <SearchableSelect
                            value={formData().initSystemVersion || ""}
                            options={getVersionOptions(option.id)}
                            onChange={(value) =>
                              updateFormData("initSystemVersion", value)
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
                {t("distribution.form.fields.filesystem")}
              </legend>
              <section class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.filesystem || []}>
                  {(option) => (
                    <article
                      class={`flex flex-col border rounded-md transition-colors ${
                        formData().filesystem === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:border-muted-foreground"
                      }`}
                    >
                      <section class="flex items-stretch">
                        <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                          <input
                            type="radio"
                            name="filesystem"
                            value={option.id}
                            checked={formData().filesystem === option.id}
                            onChange={(e) => {
                              updateFormData("filesystem", e.target.value);
                              fetchVersionsForComponent(
                                e.target.value,
                                "filesystemVersion",
                              );
                            }}
                            class="mr-3"
                          />
                          <strong class="font-medium">{option.name}</strong>
                          <span class="ml-2 text-xs px-1.5 py-0.5 bg-muted text-muted-foreground rounded">
                            {t("distribution.form.hybrid")}
                          </span>
                        </label>
                      </section>
                      <Show when={formData().filesystem === option.id}>
                        <section class="border-t border-border p-3 bg-muted/30">
                          <label class="flex items-center cursor-pointer">
                            <input
                              type="checkbox"
                              checked={formData().filesystemUserspaceEnabled}
                              onChange={(e) =>
                                updateFormData(
                                  "filesystemUserspaceEnabled",
                                  e.target.checked,
                                )
                              }
                              class="mr-2"
                            />
                            <span class="text-sm">
                              {t("distribution.form.includeUserspaceTools")}
                            </span>
                          </label>
                          <Show when={formData().filesystemUserspaceEnabled}>
                            <aside class="mt-2">
                              <SearchableSelect
                                value={formData().filesystemVersion || ""}
                                options={getVersionOptions(option.id)}
                                onChange={(value) =>
                                  updateFormData("filesystemVersion", value)
                                }
                                placeholder={getVersionPlaceholder(option.id)}
                                searchPlaceholder={t(
                                  "distribution.versionModal.searchPlaceholder",
                                )}
                                loading={loadingVersions()[option.id]}
                                maxDisplayed={30}
                                fullWidth
                              />
                            </aside>
                          </Show>
                        </section>
                      </Show>
                    </article>
                  )}
                </For>
              </section>
            </fieldset>

            {/* Partitioning */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.partitioning")}
              </label>
              <div class="grid grid-cols-2 gap-2">
                <For each={componentOptions()?.partitioningModes || []}>
                  {(option) => (
                    <label
                      class={`flex items-center p-3 border rounded-md cursor-pointer transition-colors ${
                        formData().partitioning === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted"
                      }`}
                    >
                      <input
                        type="radio"
                        name="partitioning"
                        value={option.id}
                        checked={formData().partitioning === option.id}
                        onChange={(e) =>
                          updateFormData("partitioning", e.target.value)
                        }
                        class="mr-3"
                      />
                      <span>{option.name}</span>
                    </label>
                  )}
                </For>
              </div>
            </div>

            {/* Filesystem Hierarchy */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.filesystemHierarchy")}
              </label>
              <div class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.filesystemHierarchies || []}>
                  {(option) => (
                    <label
                      class={`flex items-start p-3 border rounded-md cursor-pointer transition-colors ${
                        formData().filesystemHierarchy === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted"
                      }`}
                    >
                      <input
                        type="radio"
                        name="filesystemHierarchy"
                        value={option.id}
                        checked={formData().filesystemHierarchy === option.id}
                        onChange={(e) =>
                          updateFormData("filesystemHierarchy", e.target.value)
                        }
                        class="mr-3 mt-0.5"
                      />
                      <div>
                        <div class="font-medium">{option.name}</div>
                        <div class="text-sm text-muted-foreground">
                          {option.description}
                        </div>
                      </div>
                    </label>
                  )}
                </For>
              </div>
            </div>

            {/* Package Manager */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.packageManager")}
              </label>
              <div class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.packageManagers || []}>
                  {(option) => (
                    <label
                      class={`flex items-center p-3 border rounded-md cursor-pointer transition-colors ${
                        formData().packageManager === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted"
                      }`}
                    >
                      <input
                        type="radio"
                        name="packageManager"
                        value={option.id}
                        checked={formData().packageManager === option.id}
                        onChange={(e) =>
                          updateFormData("packageManager", e.target.value)
                        }
                        class="mr-3"
                      />
                      <span>{option.name}</span>
                    </label>
                  )}
                </For>
              </div>
            </div>
          </article>
        </Show>

        {/* Step 3: Security & Runtime */}
        <Show when={currentStep() === 3}>
          <article class="space-y-6 w-full">
            <header>
              <h3 class="text-lg font-semibold mb-1">
                {t("distribution.form.steps.securityRuntime.title")}
              </h3>
              <p class="text-sm text-muted-foreground">
                {t("distribution.form.steps.securityRuntime.description")}
              </p>
            </header>

            {/* Security System */}
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.form.fields.securitySystem")}
              </legend>
              <section class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.security || []}>
                  {(option) => (
                    <article
                      class={`flex flex-col border rounded-md transition-colors ${
                        formData().securitySystem === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:border-muted-foreground"
                      }`}
                    >
                      <section class="flex items-stretch">
                        <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                          <input
                            type="radio"
                            name="securitySystem"
                            value={option.id}
                            checked={formData().securitySystem === option.id}
                            onChange={(e) => {
                              updateFormData("securitySystem", e.target.value);
                              if (e.target.value !== "none") {
                                fetchVersionsForComponent(
                                  e.target.value,
                                  "securitySystemVersion",
                                );
                              }
                            }}
                            class="mr-3"
                          />
                          <strong class="font-medium">{option.name}</strong>
                          <Show when={option.id !== "none"}>
                            <span class="ml-2 text-xs px-1.5 py-0.5 bg-muted text-muted-foreground rounded">
                              {t("distribution.form.hybrid")}
                            </span>
                          </Show>
                        </label>
                      </section>
                      <Show
                        when={
                          formData().securitySystem === option.id &&
                          option.id !== "none"
                        }
                      >
                        <section class="border-t border-border p-3 bg-muted/30">
                          <label class="flex items-center cursor-pointer">
                            <input
                              type="checkbox"
                              checked={
                                formData().securitySystemUserspaceEnabled
                              }
                              onChange={(e) =>
                                updateFormData(
                                  "securitySystemUserspaceEnabled",
                                  e.target.checked,
                                )
                              }
                              class="mr-2"
                            />
                            <span class="text-sm">
                              {t("distribution.form.includeUserspaceTools")}
                            </span>
                          </label>
                          <Show
                            when={formData().securitySystemUserspaceEnabled}
                          >
                            <aside class="mt-2">
                              <SearchableSelect
                                value={formData().securitySystemVersion || ""}
                                options={getVersionOptions(option.id)}
                                onChange={(value) =>
                                  updateFormData("securitySystemVersion", value)
                                }
                                placeholder={getVersionPlaceholder(option.id)}
                                searchPlaceholder={t(
                                  "distribution.versionModal.searchPlaceholder",
                                )}
                                loading={loadingVersions()[option.id]}
                                maxDisplayed={30}
                                fullWidth
                              />
                            </aside>
                          </Show>
                        </section>
                      </Show>
                    </article>
                  )}
                </For>
              </section>
            </fieldset>

            {/* Container Runtime */}
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.form.fields.containerRuntime")}
              </legend>
              <section class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.container || []}>
                  {(option) => (
                    <article
                      class={`flex items-stretch border rounded-md transition-colors ${
                        formData().containerRuntime === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:border-muted-foreground"
                      }`}
                    >
                      <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                        <input
                          type="radio"
                          name="containerRuntime"
                          value={option.id}
                          checked={formData().containerRuntime === option.id}
                          onChange={(e) => {
                            updateFormData("containerRuntime", e.target.value);
                            if (e.target.value !== "none") {
                              fetchVersionsForComponent(
                                e.target.value,
                                "containerRuntimeVersion",
                              );
                            }
                          }}
                          class="mr-3"
                        />
                        <strong class="font-medium">{option.name}</strong>
                      </label>
                      <Show
                        when={
                          formData().containerRuntime === option.id &&
                          option.id !== "none"
                        }
                      >
                        <aside class="w-52 border-l border-border">
                          <SearchableSelect
                            value={formData().containerRuntimeVersion || ""}
                            options={getVersionOptions(option.id)}
                            onChange={(value) =>
                              updateFormData("containerRuntimeVersion", value)
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
                {t("distribution.form.fields.virtualizationRuntime")}
              </legend>
              <section class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.virtualization || []}>
                  {(option) => (
                    <article
                      class={`flex items-stretch border rounded-md transition-colors ${
                        formData().virtualizationRuntime === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:border-muted-foreground"
                      }`}
                    >
                      <label class="flex-1 flex items-center p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                        <input
                          type="radio"
                          name="virtualizationRuntime"
                          value={option.id}
                          checked={
                            formData().virtualizationRuntime === option.id
                          }
                          onChange={(e) => {
                            updateFormData(
                              "virtualizationRuntime",
                              e.target.value,
                            );
                            if (e.target.value !== "none") {
                              fetchVersionsForComponent(
                                e.target.value,
                                "virtualizationRuntimeVersion",
                              );
                            }
                          }}
                          class="mr-3"
                        />
                        <strong class="font-medium">{option.name}</strong>
                      </label>
                      <Show
                        when={
                          formData().virtualizationRuntime === option.id &&
                          option.id !== "none"
                        }
                      >
                        <aside class="w-52 border-l border-border">
                          <SearchableSelect
                            value={
                              formData().virtualizationRuntimeVersion || ""
                            }
                            options={getVersionOptions(option.id)}
                            onChange={(value) =>
                              updateFormData(
                                "virtualizationRuntimeVersion",
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
          </article>
        </Show>

        {/* Step 4: Distribution Type */}
        <Show when={currentStep() === 4}>
          <article class="space-y-6 w-full">
            <header>
              <h3 class="text-lg font-semibold mb-1">
                {t("distribution.form.steps.distributionType.title")}
              </h3>
              <p class="text-sm text-muted-foreground">
                {t("distribution.form.steps.distributionType.description")}
              </p>
            </header>

            {/* Distribution Type */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.targetEnvironment")}
              </label>
              <div class="grid grid-cols-2 gap-2">
                <For each={componentOptions()?.distributionTypes || []}>
                  {(option) => (
                    <label
                      class={`flex items-start p-3 border rounded-md cursor-pointer transition-colors ${
                        formData().distributionType === option.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted"
                      }`}
                    >
                      <input
                        type="radio"
                        name="distributionType"
                        value={option.id}
                        checked={formData().distributionType === option.id}
                        onChange={(e) =>
                          updateFormData("distributionType", e.target.value)
                        }
                        class="mr-3 mt-0.5"
                      />
                      <div>
                        <div class="font-medium">{option.name}</div>
                        <div class="text-sm text-muted-foreground">
                          {option.description}
                        </div>
                      </div>
                    </label>
                  )}
                </For>
              </div>
            </div>

            {/* Desktop Environment (conditional) */}
            <Show when={formData().distributionType === "desktop"}>
              <fieldset class="space-y-2">
                <legend class="text-sm font-medium">
                  {t("distribution.form.fields.desktopEnvironment.label")}
                </legend>
                <p class="text-xs text-muted-foreground mb-2">
                  {t("distribution.form.fields.desktopEnvironment.waylandOnly")}
                </p>
                <section class="grid grid-cols-1 gap-2">
                  <For each={componentOptions()?.desktop || []}>
                    {(option) => (
                      <article
                        class={`flex items-stretch border rounded-md transition-colors ${
                          formData().desktopEnvironment === option.id
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
                              formData().desktopEnvironment === option.id
                            }
                            onChange={(e) => {
                              updateFormData(
                                "desktopEnvironment",
                                e.target.value,
                              );
                              fetchVersionsForComponent(
                                e.target.value,
                                "desktopEnvironmentVersion",
                              );
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
                          when={formData().desktopEnvironment === option.id}
                        >
                          <aside class="w-52 border-l border-border">
                            <SearchableSelect
                              value={formData().desktopEnvironmentVersion || ""}
                              options={getVersionOptions(option.id)}
                              onChange={(value) =>
                                updateFormData(
                                  "desktopEnvironmentVersion",
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
            </Show>
          </article>
        </Show>
      </section>

      {/* Navigation buttons */}
      <section class="flex items-center justify-between pt-3 border-t border-border">
        <button
          type="button"
          onClick={props.onCancel}
          class="px-3 py-1.5 text-sm border border-border rounded-md hover:bg-muted transition-colors"
        >
          {t("distribution.form.actions.cancel")}
        </button>

        <div class="flex items-center gap-2">
          <Show when={currentStep() > 1}>
            <button
              type="button"
              onClick={prevStep}
              class="px-3 py-1.5 text-sm border border-border rounded-md hover:bg-muted transition-colors"
            >
              {t("distribution.form.actions.previous")}
            </button>
          </Show>

          <Show
            when={currentStep() < 4}
            fallback={
              <button
                type="button"
                onClick={handleSubmit}
                disabled={!isStepValid(4)}
                class="px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t("distribution.form.actions.create")}
              </button>
            }
          >
            <button
              type="button"
              onClick={nextStep}
              disabled={!isStepValid(currentStep())}
              class="px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t("distribution.form.actions.next")}
            </button>
          </Show>
        </div>
      </section>
    </form>
  );
};
