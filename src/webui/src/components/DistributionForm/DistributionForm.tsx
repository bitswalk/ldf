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
  type Component as LDFComponent,
} from "../../services/components";
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
  partitioningType: string;
  initSystem: string;
  filesystem: string;
  partitioning: string;
  filesystemHierarchy: string;
  packageManager: string;
  securitySystem: string;
  containerRuntime: string;
  virtualizationRuntime: string;
  distributionType: string;
  desktopEnvironment?: string;
}

// Final JSON structure to send to LDF server
interface LDFDistributionConfig {
  name: string;
  core: {
    kernel: {
      version: string;
    };
    bootloader: string;
    partitioning: {
      type: string;
      mode: string;
    };
  };
  system: {
    init: string;
    filesystem: {
      type: string;
      hierarchy: string;
    };
    packageManager: string;
  };
  security: {
    system: string;
  };
  runtime: {
    container: string;
    virtualization: string;
  };
  target: {
    type: string;
    desktop?: {
      environment: string;
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

export const DistributionForm: Component<DistributionFormProps> = (props) => {
  const [currentStep, setCurrentStep] = createSignal(1);
  const [kernelVersions] = createResource(fetchKernelVersions);
  const [componentOptions] = createResource(fetchComponentOptions);
  const [formData, setFormData] = createSignal<DistributionFormData>({
    name: "",
    kernelVersion: "",
    bootloader: "",
    partitioningType: "",
    initSystem: "",
    filesystem: "",
    partitioning: "",
    filesystemHierarchy: "",
    packageManager: "",
    securitySystem: "",
    containerRuntime: "",
    virtualizationRuntime: "",
    distributionType: "",
    desktopEnvironment: "",
  });

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

    // Transform form data into LDF server JSON format
    const ldfConfig: LDFDistributionConfig = {
      name: data.name.trim() || `custom-distribution-${Date.now()}`,
      core: {
        kernel: {
          version: data.kernelVersion,
        },
        bootloader: data.bootloader,
        partitioning: {
          type: data.partitioningType,
          mode: data.partitioning,
        },
      },
      system: {
        init: data.initSystem,
        filesystem: {
          type: data.filesystem,
          hierarchy: data.filesystemHierarchy,
        },
        packageManager: data.packageManager,
      },
      security: {
        system: data.securitySystem,
      },
      runtime: {
        container: data.containerRuntime,
        virtualization: data.virtualizationRuntime,
      },
      target: {
        type: data.distributionType,
        ...(data.distributionType === "desktop" && data.desktopEnvironment
          ? {
              desktop: {
                environment: data.desktopEnvironment,
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
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.kernelVersion.label")}
              </label>
              <Show
                when={!kernelVersions.loading && kernelVersions()?.length === 0}
                fallback={
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
                        ? t("distribution.form.fields.kernelVersion.loading")
                        : t(
                            "distribution.form.fields.kernelVersion.placeholder",
                          )
                    }
                    searchPlaceholder={t(
                      "distribution.form.fields.kernelVersion.searchPlaceholder",
                    )}
                    loading={kernelVersions.loading}
                    maxDisplayed={50}
                    fullWidth
                  />
                }
              >
                <select
                  class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary appearance-none"
                  style="background-image: url('data:image/svg+xml;charset=UTF-8,%3csvg xmlns=%27http://www.w3.org/2000/svg%27 viewBox=%270 0 24 24%27 fill=%27none%27 stroke=%27currentColor%27 stroke-width=%272%27 stroke-linecap=%27round%27 stroke-linejoin=%27round%27%3e%3cpolyline points=%276 9 12 15 18 9%27%3e%3c/polyline%3e%3c/svg%3e'); background-repeat: no-repeat; background-position: right 0.75rem center; background-size: 1.25rem; padding-right: 2.5rem;"
                  value={formData().kernelVersion}
                  onChange={(e) =>
                    updateFormData("kernelVersion", e.target.value)
                  }
                >
                  <option value="">
                    {t("distribution.form.fields.kernelVersion.placeholder")}
                  </option>
                  <option value="none">
                    {t("distribution.form.fields.kernelVersion.none")}
                  </option>
                </select>
                <p class="text-xs text-muted-foreground">
                  {t("distribution.form.fields.kernelVersion.noVersionsHint")}
                </p>
              </Show>
            </div>

            {/* Bootloader */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.bootloader")}
              </label>
              <div class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.bootloader || []}>
                  {(bootloader) => (
                    <label class="flex items-start p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="bootloader"
                        value={bootloader.id}
                        checked={formData().bootloader === bootloader.id}
                        onChange={(e) =>
                          updateFormData("bootloader", e.target.value)
                        }
                        class="mr-3 mt-0.5"
                      />
                      <div>
                        <div class="font-medium">{bootloader.name}</div>
                        <div class="text-sm text-muted-foreground">
                          {bootloader.description}
                        </div>
                      </div>
                    </label>
                  )}
                </For>
              </div>
            </div>

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
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.initSystem")}
              </label>
              <div class="grid grid-cols-2 gap-2">
                <For each={componentOptions()?.init || []}>
                  {(option) => (
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="initSystem"
                        value={option.id}
                        checked={formData().initSystem === option.id}
                        onChange={(e) =>
                          updateFormData("initSystem", e.target.value)
                        }
                        class="mr-3"
                      />
                      <span>{option.name}</span>
                    </label>
                  )}
                </For>
              </div>
            </div>

            {/* Filesystem */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.filesystem")}
              </label>
              <div class="grid grid-cols-3 gap-2">
                <For each={componentOptions()?.filesystem || []}>
                  {(option) => (
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="filesystem"
                        value={option.id}
                        checked={formData().filesystem === option.id}
                        onChange={(e) =>
                          updateFormData("filesystem", e.target.value)
                        }
                        class="mr-3"
                      />
                      <span>{option.name}</span>
                    </label>
                  )}
                </For>
              </div>
            </div>

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
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.securitySystem")}
              </label>
              <div class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.security || []}>
                  {(option) => (
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="securitySystem"
                        value={option.id}
                        checked={formData().securitySystem === option.id}
                        onChange={(e) =>
                          updateFormData("securitySystem", e.target.value)
                        }
                        class="mr-3"
                      />
                      <span>{option.name}</span>
                    </label>
                  )}
                </For>
              </div>
            </div>

            {/* Container Runtime */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.containerRuntime")}
              </label>
              <div class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.container || []}>
                  {(option) => (
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="containerRuntime"
                        value={option.id}
                        checked={formData().containerRuntime === option.id}
                        onChange={(e) =>
                          updateFormData("containerRuntime", e.target.value)
                        }
                        class="mr-3"
                      />
                      <span>{option.name}</span>
                    </label>
                  )}
                </For>
              </div>
            </div>

            {/* Virtualization Runtime */}
            <div class="space-y-2">
              <label class="text-sm font-medium">
                {t("distribution.form.fields.virtualizationRuntime")}
              </label>
              <div class="grid grid-cols-1 gap-2">
                <For each={componentOptions()?.virtualization || []}>
                  {(option) => (
                    <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                      <input
                        type="radio"
                        name="virtualizationRuntime"
                        value={option.id}
                        checked={formData().virtualizationRuntime === option.id}
                        onChange={(e) =>
                          updateFormData(
                            "virtualizationRuntime",
                            e.target.value,
                          )
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
              <div class="space-y-2">
                <label class="text-sm font-medium">
                  {t("distribution.form.fields.desktopEnvironment.label")}
                </label>
                <p class="text-xs text-muted-foreground mb-2">
                  {t("distribution.form.fields.desktopEnvironment.waylandOnly")}
                </p>
                <div class="grid grid-cols-1 gap-2">
                  <For each={componentOptions()?.desktop || []}>
                    {(option) => (
                      <label class="flex items-center p-3 border border-border rounded-md cursor-pointer hover:bg-muted transition-colors">
                        <input
                          type="radio"
                          name="desktopEnvironment"
                          value={option.id}
                          checked={formData().desktopEnvironment === option.id}
                          onChange={(e) =>
                            updateFormData("desktopEnvironment", e.target.value)
                          }
                          class="mr-3"
                        />
                        <span>{option.name}</span>
                      </label>
                    )}
                  </For>
                </div>
              </div>
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
