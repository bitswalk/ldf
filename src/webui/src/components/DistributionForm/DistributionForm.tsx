import type { Component } from "solid-js";
import {
  createSignal,
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
  type SourceType,
} from "../../services/sourceVersions";
import {
  listComponents,
  groupByCategory,
  resolveVersionRule,
  type Component as LDFComponent,
  type VersionRule,
} from "../../services/components";
import { ComponentVersionModal } from "../ComponentVersionModal";
import { Icon } from "../Icon";

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
  partitioningType: string;
  initSystem: string;
  initSystemVersion?: string;
  filesystem: string;
  filesystemVersion?: string;
  partitioning: string;
  filesystemHierarchy: string;
  packageManager: string;
  securitySystem: string;
  securitySystemVersion?: string;
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
    packageManager: string;
  };
  security: {
    system: string;
    system_version?: string;
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
  sourceType: SourceType;
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
      const sourceType: SourceType = source.is_system ? "default" : "user";

      // Paginate through all versions
      const pageSize = 100;
      let offset = 0;
      let hasMore = true;

      while (hasMore) {
        const versionsResult = await listSourceVersions(
          source.id,
          sourceType,
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
              sourceType,
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
    partitioningType: "",
    initSystem: "",
    initSystemVersion: undefined,
    filesystem: "",
    filesystemVersion: undefined,
    partitioning: "",
    filesystemHierarchy: "",
    packageManager: "",
    securitySystem: "",
    securitySystemVersion: undefined,
    containerRuntime: "",
    containerRuntimeVersion: undefined,
    virtualizationRuntime: "",
    virtualizationRuntimeVersion: undefined,
    distributionType: "",
    desktopEnvironment: "",
    desktopEnvironmentVersion: undefined,
  });

  // Version modal state
  const [versionModalField, setVersionModalField] = createSignal<string | null>(
    null,
  );
  const [resolvedVersions, setResolvedVersions] = createSignal<
    Record<string, string>
  >({});

  // Helper to get component options
  const componentOptions = () => componentsData()?.options;

  // Helper to get component map
  const componentMap = () => componentsData()?.componentMap || {};

  // Get selected component for version modal
  const selectedComponentForModal = createMemo(() => {
    const field = versionModalField();
    if (!field) return null;

    const data = formData();
    let componentName: string | undefined;

    // Map field to component name
    switch (field) {
      case "kernel":
        componentName = "kernel";
        break;
      case "bootloader":
        componentName = data.bootloader;
        break;
      case "initSystem":
        componentName = data.initSystem;
        break;
      case "filesystem":
        componentName = data.filesystem;
        break;
      case "securitySystem":
        componentName = data.securitySystem;
        break;
      case "containerRuntime":
        componentName = data.containerRuntime;
        break;
      case "virtualizationRuntime":
        componentName = data.virtualizationRuntime;
        break;
      case "desktopEnvironment":
        componentName = data.desktopEnvironment;
        break;
    }

    if (!componentName || componentName === "none") return null;
    return componentMap()[componentName] || null;
  });

  // Resolve default version for a component
  const resolveDefaultVersionFor = async (componentName: string) => {
    const component = componentMap()[componentName];
    if (!component) return;

    const rule = component.default_version_rule || "latest-stable";
    if (rule === "pinned" && component.default_version) {
      setResolvedVersions((prev) => ({
        ...prev,
        [componentName]: component.default_version!,
      }));
      return;
    }

    const result = await resolveVersionRule(component.id, rule);
    if (result.success && result.data.resolved_version) {
      setResolvedVersions((prev) => ({
        ...prev,
        [componentName]: result.data.resolved_version,
      }));
    }
  };

  // Handle version selection from modal
  const handleVersionSelect = (version: string | undefined) => {
    const field = versionModalField();
    if (!field) return;

    const versionField = VERSION_FIELD_MAP[field];
    if (versionField) {
      updateFormData(versionField, version || "");
    }
    setVersionModalField(null);
  };

  // Get current version for a field (either selected or resolved default)
  const getDisplayVersion = (
    field: string,
    componentName: string | undefined,
  ): string => {
    if (!componentName || componentName === "none") return "";

    const versionField = VERSION_FIELD_MAP[field];
    const selectedVersion = versionField
      ? (formData()[versionField] as string)
      : undefined;
    if (selectedVersion) return selectedVersion;

    return resolvedVersions()[componentName] || "";
  };

  // Check if a component has synced versions available
  const componentHasVersions = (componentName: string): boolean => {
    // For kernel, check the kernelVersions resource
    if (componentName === "kernel") {
      return (kernelVersions()?.length ?? 0) > 0;
    }
    // For other components, we check if there's a resolved version
    // (if resolving worked, versions exist)
    return !!resolvedVersions()[componentName];
  };

  // Version badge component
  const VersionBadge = (badgeProps: {
    field: string;
    componentName: string | undefined;
  }) => {
    const displayVersion = () => {
      if (!badgeProps.componentName || badgeProps.componentName === "none")
        return "";

      const version = getDisplayVersion(
        badgeProps.field,
        badgeProps.componentName,
      );
      if (version) return version;

      // Check if we're still loading
      if (!componentHasVersions(badgeProps.componentName)) {
        return t("distribution.form.version.notSynced");
      }
      return t("distribution.form.version.loading");
    };

    const handleClick = (e: MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (badgeProps.componentName && badgeProps.componentName !== "none") {
        // Trigger version resolution if not already done
        if (!resolvedVersions()[badgeProps.componentName]) {
          resolveDefaultVersionFor(badgeProps.componentName);
        }
        setVersionModalField(badgeProps.field);
      }
    };

    return (
      <button
        type="button"
        onClick={handleClick}
        class="text-xs px-2 py-0.5 rounded-full bg-muted hover:bg-primary/20 font-mono transition-colors flex items-center gap-1"
      >
        <data
          value={getDisplayVersion(badgeProps.field, badgeProps.componentName)}
        >
          {displayVersion()}
        </data>
        <Icon name="caret-down" size="xs" />
      </button>
    );
  };

  const updateFormData = (field: keyof DistributionFormData, value: string) => {
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
        packageManager: data.packageManager,
      },
      security: {
        system: data.securitySystem,
        system_version:
          data.securitySystem !== "none"
            ? getVersion(data.securitySystemVersion, data.securitySystem)
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
              <figure class="flex items-center justify-between p-3 border border-border rounded-md bg-background">
                <figcaption class="font-medium">Linux Kernel</figcaption>
                <VersionBadge field="kernel" componentName="kernel" />
              </figure>
              <Show
                when={!kernelVersions.loading && kernelVersions()?.length === 0}
              >
                <p class="text-xs text-muted-foreground">
                  {t("distribution.form.fields.kernelVersion.noVersionsHint")}
                </p>
              </Show>
            </fieldset>

            {/* Bootloader */}
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.form.fields.bootloader")}
              </legend>
              <section class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.bootloader || []}>
                  {(bootloader) => (
                    <label class="flex items-start p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="bootloader"
                        value={bootloader.id}
                        checked={formData().bootloader === bootloader.id}
                        onChange={(e) => {
                          updateFormData("bootloader", e.target.value);
                          // Trigger version resolution when selecting
                          resolveDefaultVersionFor(e.target.value);
                        }}
                        class="mr-3 mt-0.5"
                      />
                      <article class="flex-1">
                        <header class="flex items-center justify-between">
                          <strong class="font-medium">{bootloader.name}</strong>
                          <Show when={formData().bootloader === bootloader.id}>
                            <VersionBadge
                              field="bootloader"
                              componentName={bootloader.id}
                            />
                          </Show>
                        </header>
                        <p class="text-sm text-muted-foreground">
                          {bootloader.description}
                        </p>
                      </article>
                    </label>
                  )}
                </For>
              </section>
            </fieldset>

            {/* Partitioning Type */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.partitioningSystem")}
              </label>
              <div class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.partitioningTypes || []}>
                  {(option) => (
                    <label class="flex items-start p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
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
              <section class="grid grid-cols-2 gap-2">
                <For each={componentOptions()?.init || []}>
                  {(option) => (
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="initSystem"
                        value={option.id}
                        checked={formData().initSystem === option.id}
                        onChange={(e) => {
                          updateFormData("initSystem", e.target.value);
                          resolveDefaultVersionFor(e.target.value);
                        }}
                        class="mr-3"
                      />
                      <article class="flex-1 flex items-center justify-between">
                        <strong class="font-medium">{option.name}</strong>
                        <Show when={formData().initSystem === option.id}>
                          <VersionBadge
                            field="initSystem"
                            componentName={option.id}
                          />
                        </Show>
                      </article>
                    </label>
                  )}
                </For>
              </section>
            </fieldset>

            {/* Filesystem */}
            <fieldset class="space-y-2">
              <legend class="text-sm font-medium">
                {t("distribution.form.fields.filesystem")}
              </legend>
              <section class="grid grid-cols-3 gap-2">
                <For each={componentOptions()?.filesystem || []}>
                  {(option) => (
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="filesystem"
                        value={option.id}
                        checked={formData().filesystem === option.id}
                        onChange={(e) => {
                          updateFormData("filesystem", e.target.value);
                          resolveDefaultVersionFor(e.target.value);
                        }}
                        class="mr-3"
                      />
                      <article class="flex-1 flex items-center justify-between">
                        <strong class="font-medium">{option.name}</strong>
                        <Show when={formData().filesystem === option.id}>
                          <VersionBadge
                            field="filesystem"
                            componentName={option.id}
                          />
                        </Show>
                      </article>
                    </label>
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
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
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
                    <label class="flex items-start p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
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
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
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
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="securitySystem"
                        value={option.id}
                        checked={formData().securitySystem === option.id}
                        onChange={(e) => {
                          updateFormData("securitySystem", e.target.value);
                          if (e.target.value !== "none") {
                            resolveDefaultVersionFor(e.target.value);
                          }
                        }}
                        class="mr-3"
                      />
                      <article class="flex-1 flex items-center justify-between">
                        <strong class="font-medium">{option.name}</strong>
                        <Show
                          when={
                            formData().securitySystem === option.id &&
                            option.id !== "none"
                          }
                        >
                          <VersionBadge
                            field="securitySystem"
                            componentName={option.id}
                          />
                        </Show>
                      </article>
                    </label>
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
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="containerRuntime"
                        value={option.id}
                        checked={formData().containerRuntime === option.id}
                        onChange={(e) => {
                          updateFormData("containerRuntime", e.target.value);
                          if (e.target.value !== "none") {
                            resolveDefaultVersionFor(e.target.value);
                          }
                        }}
                        class="mr-3"
                      />
                      <article class="flex-1 flex items-center justify-between">
                        <strong class="font-medium">{option.name}</strong>
                        <Show
                          when={
                            formData().containerRuntime === option.id &&
                            option.id !== "none"
                          }
                        >
                          <VersionBadge
                            field="containerRuntime"
                            componentName={option.id}
                          />
                        </Show>
                      </article>
                    </label>
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
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="virtualizationRuntime"
                        value={option.id}
                        checked={formData().virtualizationRuntime === option.id}
                        onChange={(e) => {
                          updateFormData(
                            "virtualizationRuntime",
                            e.target.value,
                          );
                          if (e.target.value !== "none") {
                            resolveDefaultVersionFor(e.target.value);
                          }
                        }}
                        class="mr-3"
                      />
                      <article class="flex-1 flex items-center justify-between">
                        <strong class="font-medium">{option.name}</strong>
                        <Show
                          when={
                            formData().virtualizationRuntime === option.id &&
                            option.id !== "none"
                          }
                        >
                          <VersionBadge
                            field="virtualizationRuntime"
                            componentName={option.id}
                          />
                        </Show>
                      </article>
                    </label>
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
              <div class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.distributionTypes || []}>
                  {(option) => (
                    <label class="flex items-start p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
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
                      <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                        <input
                          type="radio"
                          name="desktopEnvironment"
                          value={option.id}
                          checked={formData().desktopEnvironment === option.id}
                          onChange={(e) => {
                            updateFormData(
                              "desktopEnvironment",
                              e.target.value,
                            );
                            resolveDefaultVersionFor(e.target.value);
                          }}
                          class="mr-3"
                        />
                        <article class="flex-1 flex items-center justify-between">
                          <strong class="font-medium">{option.name}</strong>
                          <Show
                            when={formData().desktopEnvironment === option.id}
                          >
                            <VersionBadge
                              field="desktopEnvironment"
                              componentName={option.id}
                            />
                          </Show>
                        </article>
                      </label>
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

      {/* Version Selection Modal */}
      <Show when={versionModalField() && selectedComponentForModal()}>
        <ComponentVersionModal
          isOpen={!!versionModalField()}
          onClose={() => setVersionModalField(null)}
          component={selectedComponentForModal()!}
          currentVersion={
            versionModalField()
              ? (formData()[
                  VERSION_FIELD_MAP[versionModalField()!]
                ] as string) || undefined
              : undefined
          }
          onSelectVersion={handleVersionSelect}
        />
      </Show>
    </form>
  );
};
