import type { Component } from "solid-js";
import { createSignal, onMount, Show, For, createEffect } from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { DownloadStatus } from "../../components/DownloadStatus";
import {
  SearchableSelect,
  type SearchableSelectOption,
} from "../../components/SearchableSelect";
import {
  getDistribution,
  updateDistribution,
  type Distribution,
  type DistributionStatus,
  type DistributionConfig,
} from "../../services/distribution";
import { listDefaultSources } from "../../services/sources";
import {
  listSourceVersions,
  type SourceVersion,
} from "../../services/sourceVersions";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface DistributionDetailProps {
  distributionId: string;
  onBack: () => void;
  user?: UserInfo | null;
}

// Configuration options (matching DistributionForm)
const configOptions = {
  bootloaders: [
    { id: "systemd-boot", name: "systemd-boot" },
    { id: "u-boot", name: "U-Boot" },
    { id: "grub2", name: "GRUB2" },
  ],
  partitioningTypes: [
    { id: "a-b", name: "A/B Partitioning" },
    { id: "single", name: "Single Partition" },
  ],
  partitioningModes: [
    { id: "lvm", name: "LVM" },
    { id: "raw", name: "Raw" },
  ],
  initSystems: [
    { id: "systemd", name: "systemd" },
    { id: "openrc", name: "OpenRC" },
  ],
  filesystems: [
    { id: "btrfs", name: "Btrfs" },
    { id: "xfs", name: "XFS" },
    { id: "ext4", name: "ext4" },
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
  securitySystems: [
    { id: "selinux", name: "SELinux" },
    { id: "apparmor", name: "AppArmor" },
    { id: "none", name: "None" },
  ],
  containerRuntimes: [
    { id: "docker-podman", name: "Docker/Podman" },
    { id: "runc", name: "runC" },
    { id: "cri-o", name: "CRI-O" },
    { id: "none", name: "None" },
  ],
  virtualizationRuntimes: [
    { id: "cloud-hypervisor", name: "Cloud Hypervisor" },
    { id: "qemu-kvm-libvirt", name: "QEMU/KVM" },
    { id: "none", name: "None" },
  ],
  distributionTypes: [
    { id: "desktop", name: "Desktop" },
    { id: "server", name: "Server" },
  ],
  desktopEnvironments: [
    { id: "kde", name: "KDE Plasma" },
    { id: "gnome", name: "GNOME" },
    { id: "swaywm", name: "SwayWM" },
  ],
};

// Helper to get display name from options
const getOptionName = (
  options: Array<{ id: string; name: string }>,
  id: string | undefined,
): string => {
  if (!id) return "-";
  const option = options.find((o) => o.id === id);
  return option?.name || id;
};

export const DistributionDetail: Component<DistributionDetailProps> = (
  props,
) => {
  const [distribution, setDistribution] = createSignal<Distribution | null>(
    null,
  );
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [notification, setNotification] = createSignal<{
    type: "success" | "error";
    message: string;
  } | null>(null);
  const [isEditing, setIsEditing] = createSignal(false);
  const [editedConfig, setEditedConfig] =
    createSignal<DistributionConfig | null>(null);
  const [saving, setSaving] = createSignal(false);
  const [kernelVersions, setKernelVersions] = createSignal<SourceVersion[]>([]);
  const [kernelVersionsLoading, setKernelVersionsLoading] = createSignal(false);
  const [allKernelVersions, setAllKernelVersions] = createSignal<
    SourceVersion[]
  >([]);

  const fetchKernelVersions = async () => {
    setKernelVersionsLoading(true);

    // First, find the kernel source from default sources
    const sourcesResult = await listDefaultSources();
    if (!sourcesResult.success) {
      setKernelVersionsLoading(false);
      return;
    }

    // Find a kernel source (look for "kernel" in name or component_id)
    const kernelSource = sourcesResult.sources.find(
      (s) =>
        s.name.toLowerCase().includes("kernel") ||
        s.component_id?.toLowerCase().includes("kernel"),
    );

    if (!kernelSource) {
      setKernelVersionsLoading(false);
      return;
    }

    // Fetch versions from this source (get more to allow searching)
    const versionsResult = await listSourceVersions(
      kernelSource.id,
      "default",
      500, // Fetch more for search
      0,
      true, // stable only
    );

    if (versionsResult.success) {
      setAllKernelVersions(versionsResult.versions);
      setKernelVersions(versionsResult.versions.slice(0, 10));
    }

    setKernelVersionsLoading(false);
  };

  const kernelVersionOptions = (): SearchableSelectOption[] => {
    return allKernelVersions().map((v) => ({
      value: v.version,
      label: v.version,
      sublabel: v.release_date
        ? new Date(v.release_date).toLocaleDateString()
        : undefined,
    }));
  };

  const fetchDistribution = async () => {
    setLoading(true);
    setError(null);

    const result = await getDistribution(props.distributionId);

    setLoading(false);

    if (result.success) {
      setDistribution(result.distribution);
      if (result.distribution.config) {
        setEditedConfig(JSON.parse(JSON.stringify(result.distribution.config)));
      }
    } else {
      setError(result.message);
    }
  };

  onMount(() => {
    fetchDistribution();
    fetchKernelVersions();
  });

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const getStatusIcon = (status: DistributionStatus): string => {
    switch (status) {
      case "ready":
        return "check-circle";
      case "pending":
      case "downloading":
      case "validating":
        return "spinner-gap";
      case "failed":
        return "x-circle";
      case "deleted":
        return "trash";
      default:
        return "circle";
    }
  };

  const getStatusColor = (status: DistributionStatus): string => {
    switch (status) {
      case "ready":
        return "text-green-500";
      case "pending":
      case "downloading":
      case "validating":
        return "text-primary";
      case "failed":
        return "text-red-500";
      case "deleted":
        return "text-muted-foreground";
      default:
        return "text-muted-foreground";
    }
  };

  const handleDownloadSuccess = (message: string) => {
    setNotification({ type: "success", message });
    setTimeout(() => setNotification(null), 3000);
  };

  const handleDownloadError = (message: string) => {
    setNotification({ type: "error", message });
    setTimeout(() => setNotification(null), 5000);
  };

  const handleEditToggle = () => {
    if (isEditing()) {
      // Cancel editing - restore original config
      const dist = distribution();
      if (dist?.config) {
        setEditedConfig(JSON.parse(JSON.stringify(dist.config)));
      }
      setIsEditing(false);
    } else {
      setIsEditing(true);
    }
  };

  const handleSaveConfig = async () => {
    const config = editedConfig();
    const dist = distribution();
    if (!config || !dist) return;

    setSaving(true);

    // Send update to the server
    const result = await updateDistribution(dist.id, { config });

    if (result.success) {
      // Update local state with the response from server
      setDistribution(result.distribution);
      setEditedConfig(JSON.parse(JSON.stringify(result.distribution.config)));
      setIsEditing(false);
      setNotification({ type: "success", message: "Configuration saved" });
    } else {
      setNotification({ type: "error", message: result.message });
    }

    setSaving(false);
    setTimeout(() => setNotification(null), 3000);
  };

  const updateConfig = (path: string, value: string) => {
    const config = editedConfig();
    if (!config) return;

    const newConfig = JSON.parse(JSON.stringify(config));
    const parts = path.split(".");
    let current: any = newConfig;
    for (let i = 0; i < parts.length - 1; i++) {
      current = current[parts[i]];
    }
    current[parts[parts.length - 1]] = value;
    setEditedConfig(newConfig);
  };

  const ConfigSelect: Component<{
    label: string;
    value: string | undefined;
    options: Array<{ id: string; name: string }>;
    path: string;
    disabled?: boolean;
  }> = (selectProps) => (
    <div class="flex items-center justify-between py-2">
      <span class="text-muted-foreground text-sm">{selectProps.label}</span>
      <Show
        when={isEditing() && !selectProps.disabled}
        fallback={
          <span class="text-sm font-medium">
            {getOptionName(selectProps.options, selectProps.value)}
          </span>
        }
      >
        <select
          class="px-2 py-1 text-sm bg-background border border-border rounded-md focus:outline-none focus:border-primary"
          value={selectProps.value || ""}
          onChange={(e) => updateConfig(selectProps.path, e.target.value)}
        >
          <For each={selectProps.options}>
            {(opt) => <option value={opt.id}>{opt.name}</option>}
          </For>
        </select>
      </Show>
    </div>
  );

  const config = () => (isEditing() ? editedConfig() : distribution()?.config);

  return (
    <section class="h-full w-full relative">
      <section class="h-full flex flex-col p-8 gap-6 overflow-auto">
        {/* Header */}
        <header class="flex items-center gap-4">
          <button
            onClick={props.onBack}
            class="p-2 rounded-md hover:bg-muted transition-colors"
            title="Back to distributions"
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <h1 class="text-4xl font-bold">
              {distribution()?.name || "Distribution Details"}
            </h1>
            <p class="text-muted-foreground mt-1">
              Manage distribution configuration and downloads
            </p>
          </div>
        </header>

        {/* Notification */}
        <Show when={notification()}>
          <div
            class={`p-3 rounded-md ${
              notification()?.type === "success"
                ? "bg-green-500/10 border border-green-500/20 text-green-500"
                : "bg-red-500/10 border border-red-500/20 text-red-500"
            }`}
          >
            <div class="flex items-center gap-2">
              <Icon
                name={
                  notification()?.type === "success"
                    ? "check-circle"
                    : "warning-circle"
                }
                size="md"
              />
              <span>{notification()?.message}</span>
            </div>
          </div>
        </Show>

        {/* Error state */}
        <Show when={error()}>
          <div class="p-4 bg-red-500/10 border border-red-500/20 rounded-md">
            <div class="flex items-center gap-2 text-red-500">
              <Icon name="warning-circle" size="md" />
              <span>{error()}</span>
            </div>
          </div>
        </Show>

        {/* Loading state */}
        <Show when={loading()}>
          <div class="flex items-center justify-center py-16">
            <Spinner size="lg" />
          </div>
        </Show>

        {/* Content */}
        <Show when={!loading() && distribution()}>
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Distribution Info & Configuration */}
            <Card
              header={{
                title: "Distribution Info",
                actions: (
                  <div class="flex items-center gap-2">
                    <Show when={isEditing()}>
                      <button
                        onClick={handleSaveConfig}
                        disabled={saving()}
                        class="flex items-center gap-1 px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 transition-colors"
                      >
                        <Show when={saving()}>
                          <Icon
                            name="spinner-gap"
                            size="sm"
                            class="animate-spin"
                          />
                        </Show>
                        <span>Save</span>
                      </button>
                    </Show>
                    <button
                      onClick={handleEditToggle}
                      class={`flex items-center gap-1 px-3 py-1.5 text-sm rounded-md transition-colors ${
                        isEditing()
                          ? "border border-border hover:bg-muted"
                          : "border border-border hover:bg-muted"
                      }`}
                    >
                      <Icon name={isEditing() ? "x" : "pencil"} size="sm" />
                      <span>{isEditing() ? "Cancel" : "Edit"}</span>
                    </button>
                  </div>
                ),
              }}
            >
              <div class="space-y-4">
                {/* Status & Basic Info */}
                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">Status</span>
                  <span
                    class={`flex items-center gap-2 ${getStatusColor(distribution()!.status)}`}
                  >
                    <Icon
                      name={getStatusIcon(distribution()!.status)}
                      size="sm"
                      class={
                        ["pending", "downloading", "validating"].includes(
                          distribution()!.status,
                        )
                          ? "animate-spin"
                          : ""
                      }
                    />
                    <span class="capitalize">{distribution()!.status}</span>
                  </span>
                </div>

                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">Visibility</span>
                  <span class="flex items-center gap-2">
                    <Icon
                      name={
                        distribution()!.visibility === "public"
                          ? "globe"
                          : "lock"
                      }
                      size="sm"
                    />
                    <span class="capitalize">{distribution()!.visibility}</span>
                  </span>
                </div>

                <Show when={distribution()!.size_bytes > 0}>
                  <div class="flex items-center justify-between">
                    <span class="text-muted-foreground">Size</span>
                    <span class="font-mono">
                      {formatBytes(distribution()!.size_bytes)}
                    </span>
                  </div>
                </Show>

                <Show when={distribution()!.checksum}>
                  <div class="flex items-center justify-between">
                    <span class="text-muted-foreground">Checksum</span>
                    <span
                      class="font-mono text-xs truncate max-w-[200px]"
                      title={distribution()!.checksum}
                    >
                      {distribution()!.checksum}
                    </span>
                  </div>
                </Show>

                {/* Configuration Section */}
                <Show when={config()}>
                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      CORE SYSTEM
                    </h4>
                    <div class="space-y-1">
                      <div class="flex items-center justify-between py-2">
                        <span class="text-muted-foreground text-sm">
                          Kernel Version
                        </span>
                        <Show
                          when={isEditing()}
                          fallback={
                            <span class="text-sm font-mono font-medium">
                              {config()!.core.kernel.version}
                            </span>
                          }
                        >
                          <SearchableSelect
                            value={config()!.core.kernel.version}
                            options={kernelVersionOptions()}
                            onChange={(value) =>
                              updateConfig("core.kernel.version", value)
                            }
                            placeholder="Select version"
                            searchPlaceholder="Search kernel versions..."
                            loading={kernelVersionsLoading()}
                            maxDisplayed={10}
                          />
                        </Show>
                      </div>
                      <ConfigSelect
                        label="Bootloader"
                        value={config()!.core.bootloader}
                        options={configOptions.bootloaders}
                        path="core.bootloader"
                      />
                      <ConfigSelect
                        label="Partitioning Type"
                        value={config()!.core.partitioning.type}
                        options={configOptions.partitioningTypes}
                        path="core.partitioning.type"
                      />
                      <ConfigSelect
                        label="Partitioning Mode"
                        value={config()!.core.partitioning.mode}
                        options={configOptions.partitioningModes}
                        path="core.partitioning.mode"
                      />
                    </div>
                  </div>

                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      SYSTEM SERVICES
                    </h4>
                    <div class="space-y-1">
                      <ConfigSelect
                        label="Init System"
                        value={config()!.system.init}
                        options={configOptions.initSystems}
                        path="system.init"
                      />
                      <ConfigSelect
                        label="Filesystem"
                        value={config()!.system.filesystem.type}
                        options={configOptions.filesystems}
                        path="system.filesystem.type"
                      />
                      <ConfigSelect
                        label="Filesystem Hierarchy"
                        value={config()!.system.filesystem.hierarchy}
                        options={configOptions.filesystemHierarchies}
                        path="system.filesystem.hierarchy"
                      />
                      <ConfigSelect
                        label="Package Manager"
                        value={config()!.system.packageManager}
                        options={configOptions.packageManagers}
                        path="system.packageManager"
                      />
                    </div>
                  </div>

                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      SECURITY & RUNTIME
                    </h4>
                    <div class="space-y-1">
                      <ConfigSelect
                        label="Security System"
                        value={config()!.security.system}
                        options={configOptions.securitySystems}
                        path="security.system"
                      />
                      <ConfigSelect
                        label="Container Runtime"
                        value={config()!.runtime.container}
                        options={configOptions.containerRuntimes}
                        path="runtime.container"
                      />
                      <ConfigSelect
                        label="Virtualization"
                        value={config()!.runtime.virtualization}
                        options={configOptions.virtualizationRuntimes}
                        path="runtime.virtualization"
                      />
                    </div>
                  </div>

                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      TARGET ENVIRONMENT
                    </h4>
                    <div class="space-y-1">
                      <ConfigSelect
                        label="Distribution Type"
                        value={config()!.target.type}
                        options={configOptions.distributionTypes}
                        path="target.type"
                      />
                      <Show when={config()!.target.desktop}>
                        <ConfigSelect
                          label="Desktop Environment"
                          value={config()!.target.desktop?.environment}
                          options={configOptions.desktopEnvironments}
                          path="target.desktop.environment"
                        />
                        <div class="flex items-center justify-between py-2">
                          <span class="text-muted-foreground text-sm">
                            Display Server
                          </span>
                          <span class="text-sm font-medium">
                            {config()!.target.desktop?.displayServer ||
                              "Wayland"}
                          </span>
                        </div>
                      </Show>
                    </div>
                  </div>
                </Show>

                {/* No config fallback */}
                <Show when={!config()}>
                  <div class="border-t border-border pt-4 mt-4">
                    <div class="text-center py-4 text-muted-foreground">
                      <Icon
                        name="gear"
                        size="xl"
                        class="mx-auto mb-2 opacity-50"
                      />
                      <p class="text-sm">No configuration available</p>
                    </div>
                  </div>
                </Show>

                {/* Timestamps */}
                <div class="border-t border-border pt-4 mt-4 space-y-2">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">Created</span>
                    <span class="font-mono">
                      {formatDate(distribution()!.created_at)}
                    </span>
                  </div>
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">Updated</span>
                    <span class="font-mono">
                      {formatDate(distribution()!.updated_at)}
                    </span>
                  </div>
                  <Show when={distribution()!.owner_id}>
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">Owner ID</span>
                      <span class="font-mono text-xs">
                        {distribution()!.owner_id}
                      </span>
                    </div>
                  </Show>
                </div>

                <Show when={distribution()!.error_message}>
                  <div class="border-t border-border pt-4 mt-4">
                    <div class="text-sm text-red-500">
                      <div class="flex items-center gap-1 mb-1">
                        <Icon name="warning" size="sm" />
                        <span class="font-medium">Error</span>
                      </div>
                      <p>{distribution()!.error_message}</p>
                    </div>
                  </div>
                </Show>
              </div>
            </Card>

            {/* Right Column: Quick Actions + Component Downloads */}
            <div class="flex flex-col gap-6">
              {/* Quick Actions */}
              <Card header={{ title: "Quick Actions" }}>
                <div class="space-y-3">
                  <button class="w-full flex items-center gap-3 p-3 rounded-md border border-border hover:bg-muted transition-colors text-left opacity-50 cursor-not-allowed">
                    <Icon name="hammer" size="md" class="text-primary" />
                    <div>
                      <div class="font-medium">Build Distribution</div>
                      <div class="text-sm text-muted-foreground">
                        Compile and assemble distribution
                      </div>
                    </div>
                  </button>

                  <button class="w-full flex items-center gap-3 p-3 rounded-md border border-border hover:bg-muted transition-colors text-left opacity-50 cursor-not-allowed">
                    <Icon
                      name="download-simple"
                      size="md"
                      class="text-primary"
                    />
                    <div>
                      <div class="font-medium">Download Distribution</div>
                      <div class="text-sm text-muted-foreground">
                        Download as ISO or archive
                      </div>
                    </div>
                  </button>
                </div>
              </Card>

              {/* Component Downloads */}
              <Card header={{ title: "Component Downloads" }}>
                <DownloadStatus
                  distributionId={props.distributionId}
                  onSuccess={handleDownloadSuccess}
                  onError={handleDownloadError}
                  pollInterval={3000}
                />
              </Card>
            </div>
          </div>
        </Show>
      </section>
    </section>
  );
};
