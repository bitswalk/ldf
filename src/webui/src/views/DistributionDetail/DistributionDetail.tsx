import type { Component } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import {
  DownloadStatus,
  type DownloadActions,
} from "../../components/DownloadStatus";
import { DistributionEditForm } from "../../components/DistributionEditForm";
import { BuildStartDialog } from "../../components/BuildStartDialog";
import { BuildsList } from "../../components/BuildsList";
import {
  getDistribution,
  updateDistribution,
  deleteDistribution,
  getDeletionPreview,
  uploadKernelConfig,
  type Distribution,
  type DistributionStatus,
  type DistributionConfig,
  type UpdateDistributionRequest,
  type DeletionPreview,
} from "../../services/distribution";
import { clearDistributionBuilds } from "../../services/builds";
import { t } from "../../services/i18n";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface DistributionDetailProps {
  distributionId: string;
  onBack: () => void;
  onDeleted?: () => void;
  onNavigateToBuild?: (buildId: string) => void;
  user?: UserInfo | null;
}

// Configuration options for display
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
  toolchains: [
    { id: "gcc", name: "GCC" },
    { id: "llvm", name: "LLVM/Clang" },
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
  const [editModalOpen, setEditModalOpen] = createSignal(false);
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [isSubmitting, setIsSubmitting] = createSignal(false);
  const [isDeleting, setIsDeleting] = createSignal(false);
  const [deletionPreview, setDeletionPreview] =
    createSignal<DeletionPreview | null>(null);
  const [loadingPreview, setLoadingPreview] = createSignal(false);
  const [buildDialogOpen, setBuildDialogOpen] = createSignal(false);
  const [kernelConfigModalOpen, setKernelConfigModalOpen] = createSignal(false);
  const [uploadingConfig, setUploadingConfig] = createSignal(false);
  const [uploadProgress, setUploadProgress] = createSignal(0);
  const [clearingBuilds, setClearingBuilds] = createSignal(false);
  const [clearBuildsModalOpen, setClearBuildsModalOpen] = createSignal(false);
  let refetchBuilds: (() => void) | undefined;
  const [downloadActions, setDownloadActions] =
    createSignal<DownloadActions | null>(null);

  const isAdmin = () => props.user?.role === "root";

  const fetchDistribution = async () => {
    setLoading(true);
    setError(null);

    const result = await getDistribution(props.distributionId);

    setLoading(false);

    if (result.success) {
      setDistribution(result.distribution);
    } else {
      setError(result.message);
    }
  };

  onMount(() => {
    fetchDistribution();
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
      case "building":
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
      case "building":
        return "text-primary";
      case "failed":
        return "text-red-500";
      case "deleted":
        return "text-muted-foreground";
      default:
        return "text-muted-foreground";
    }
  };

  const showNotification = (type: "success" | "error", message: string) => {
    setNotification({ type, message });
    setTimeout(() => setNotification(null), type === "success" ? 3000 : 5000);
  };

  const handleDownloadSuccess = (message: string) => {
    showNotification("success", message);
  };

  const handleDownloadError = (message: string) => {
    showNotification("error", message);
  };

  const handleEdit = () => {
    setEditModalOpen(true);
  };

  const handleEditSubmit = async (data: UpdateDistributionRequest) => {
    setIsSubmitting(true);
    setError(null);

    const result = await updateDistribution(props.distributionId, data);

    setIsSubmitting(false);

    if (result.success) {
      setEditModalOpen(false);
      setDistribution(result.distribution);
      showNotification("success", t("distribution.detail.info.configSaved"));
    } else {
      setError(result.message);
    }
  };

  const handleEditCancel = () => {
    setEditModalOpen(false);
  };

  const handleDeleteClick = async () => {
    setDeleteModalOpen(true);
    setLoadingPreview(true);
    setDeletionPreview(null);

    const result = await getDeletionPreview(props.distributionId);

    setLoadingPreview(false);

    if (result.success) {
      setDeletionPreview(result.preview);
    }
  };

  const confirmDelete = async () => {
    setIsDeleting(true);
    setError(null);

    const result = await deleteDistribution(props.distributionId);

    setIsDeleting(false);

    if (result.success) {
      setDeleteModalOpen(false);
      showNotification("success", t("distribution.detail.deleteSuccess"));
      props.onDeleted?.();
      props.onBack();
    } else {
      setError(result.message);
    }
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
  };

  const confirmClearBuilds = async () => {
    setClearingBuilds(true);
    const result = await clearDistributionBuilds(props.distributionId);
    setClearingBuilds(false);
    setClearBuildsModalOpen(false);

    if (result.success) {
      showNotification("success", t("build.list.clearSuccess"));
      refetchBuilds?.();
    } else {
      showNotification("error", result.message);
    }
  };

  const handleKernelConfigUpload = async (file: File) => {
    setUploadingConfig(true);
    setUploadProgress(0);

    const result = await uploadKernelConfig(
      props.distributionId,
      file,
      (progress) => setUploadProgress(progress),
    );

    setUploadingConfig(false);

    if (result.success) {
      setKernelConfigModalOpen(false);
      showNotification(
        "success",
        t("distribution.detail.quickActions.uploadKernelConfigSuccess"),
      );
      fetchDistribution();
    } else {
      showNotification("error", result.message);
    }
  };

  const config = () => distribution()?.config;

  // Display row component for config values
  const ConfigRow: Component<{
    label: string;
    value: string | undefined;
    version?: string;
    options?: Array<{ id: string; name: string }>;
  }> = (rowProps) => (
    <article class="flex items-center justify-between py-2">
      <span class="text-muted-foreground text-sm">{rowProps.label}</span>
      <span class="text-sm font-medium flex items-center gap-2">
        <span>
          {rowProps.options
            ? getOptionName(rowProps.options, rowProps.value)
            : rowProps.value || "-"}
        </span>
        <Show when={rowProps.version}>
          <span class="text-xs font-mono px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
            {rowProps.version}
          </span>
        </Show>
      </span>
    </article>
  );

  return (
    <section class="h-full w-full relative">
      <section class="h-full flex flex-col p-8 gap-6 overflow-auto">
        {/* Header */}
        <header class="flex items-center gap-4">
          <button
            onClick={props.onBack}
            class="p-2 rounded-md hover:bg-muted transition-colors"
            title={t("distribution.detail.back")}
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <div class="flex items-center gap-3">
              <h1 class="text-4xl font-bold">
                {distribution()?.name || t("distribution.detail.title")}
              </h1>
              <Show when={distribution()}>
                <span
                  class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${getStatusColor(distribution()!.status)} bg-current/10`}
                >
                  {t(`distribution.status.${distribution()!.status}`)}
                </span>
                <span
                  class={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${
                    distribution()!.visibility === "public"
                      ? "bg-green-500/10 text-green-500"
                      : "bg-muted text-muted-foreground"
                  }`}
                >
                  <Icon
                    name={
                      distribution()!.visibility === "public" ? "globe" : "lock"
                    }
                    size="xs"
                  />
                  {t(`common.visibility.${distribution()!.visibility}`)}
                </span>
              </Show>
            </div>
            <p class="text-muted-foreground mt-1">
              {t("distribution.detail.subtitle")}
            </p>
          </div>
          <Show when={isAdmin()}>
            <div class="flex items-center gap-2">
              <button
                onClick={handleEdit}
                class="flex items-center gap-2 px-4 py-2 border border-border rounded-md hover:bg-muted transition-colors"
              >
                <Icon name="pencil" size="sm" />
                <span>{t("common.actions.edit")}</span>
              </button>
              <button
                onClick={handleDeleteClick}
                class="flex items-center gap-2 px-4 py-2 border border-destructive/50 text-destructive rounded-md hover:bg-destructive/10 transition-colors"
              >
                <Icon name="trash" size="sm" />
                <span>{t("common.actions.delete")}</span>
              </button>
            </div>
          </Show>
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
            <Card header={{ title: t("distribution.detail.info.title") }}>
              <div class="space-y-4">
                {/* Status & Basic Info */}
                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">
                    {t("distribution.table.columns.status")}
                  </span>
                  <span
                    class={`flex items-center gap-2 ${getStatusColor(distribution()!.status)}`}
                  >
                    <Icon
                      name={getStatusIcon(distribution()!.status)}
                      size="sm"
                      class={
                        [
                          "pending",
                          "downloading",
                          "validating",
                          "building",
                        ].includes(distribution()!.status)
                          ? "animate-spin"
                          : ""
                      }
                    />
                    <span class="capitalize">
                      {t(`distribution.status.${distribution()!.status}`)}
                    </span>
                  </span>
                </div>

                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">
                    {t("distribution.table.columns.visibility")}
                  </span>
                  <span class="flex items-center gap-2">
                    <Icon
                      name={
                        distribution()!.visibility === "public"
                          ? "globe"
                          : "lock"
                      }
                      size="sm"
                    />
                    <span class="capitalize">
                      {t(`common.visibility.${distribution()!.visibility}`)}
                    </span>
                  </span>
                </div>

                <Show when={distribution()!.size_bytes > 0}>
                  <div class="flex items-center justify-between">
                    <span class="text-muted-foreground">
                      {t("distribution.detail.size")}
                    </span>
                    <span class="font-mono">
                      {formatBytes(distribution()!.size_bytes)}
                    </span>
                  </div>
                </Show>

                <Show when={distribution()!.checksum}>
                  <div class="flex items-center justify-between">
                    <span class="text-muted-foreground">
                      {t("distribution.detail.checksum")}
                    </span>
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
                      {t("distribution.detail.config.coreSystem")}
                    </h4>
                    <section class="space-y-1">
                      <ConfigRow
                        label={t("distribution.detail.config.kernelVersion")}
                        value={config()!.core.kernel.version}
                      />
                      <ConfigRow
                        label={t("distribution.detail.config.bootloader")}
                        value={config()!.core.bootloader}
                        version={config()!.core.bootloader_version}
                        options={configOptions.bootloaders}
                      />
                      <ConfigRow
                        label={t("distribution.detail.config.toolchain")}
                        value={config()!.core.toolchain || "gcc"}
                        options={configOptions.toolchains}
                      />
                      <ConfigRow
                        label={t("distribution.detail.config.partitioningType")}
                        value={config()!.core.partitioning.type}
                        options={configOptions.partitioningTypes}
                      />
                      <ConfigRow
                        label={t("distribution.detail.config.partitioningMode")}
                        value={config()!.core.partitioning.mode}
                        options={configOptions.partitioningModes}
                      />
                    </section>
                  </div>

                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      {t("distribution.detail.config.systemServices")}
                    </h4>
                    <section class="space-y-1">
                      <ConfigRow
                        label={t("distribution.detail.config.initSystem")}
                        value={config()!.system.init}
                        version={config()!.system.init_version}
                        options={configOptions.initSystems}
                      />
                      <ConfigRow
                        label={t("distribution.detail.config.filesystem")}
                        value={config()!.system.filesystem.type}
                        version={config()!.system.filesystem_version}
                        options={configOptions.filesystems}
                      />
                      <ConfigRow
                        label={t(
                          "distribution.detail.config.filesystemHierarchy",
                        )}
                        value={config()!.system.filesystem.hierarchy}
                        options={configOptions.filesystemHierarchies}
                      />
                      <ConfigRow
                        label={t("distribution.detail.config.packageManager")}
                        value={config()!.system.packageManager}
                        version={config()!.system.package_manager_version}
                        options={configOptions.packageManagers}
                      />
                    </section>
                  </div>

                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      {t("distribution.detail.config.securityRuntime")}
                    </h4>
                    <section class="space-y-1">
                      <ConfigRow
                        label={t("distribution.detail.config.securitySystem")}
                        value={config()!.security.system}
                        version={config()!.security.system_version}
                        options={configOptions.securitySystems}
                      />
                      <ConfigRow
                        label={t("distribution.detail.config.containerRuntime")}
                        value={config()!.runtime.container}
                        version={config()!.runtime.container_version}
                        options={configOptions.containerRuntimes}
                      />
                      <ConfigRow
                        label={t("distribution.detail.config.virtualization")}
                        value={config()!.runtime.virtualization}
                        version={config()!.runtime.virtualization_version}
                        options={configOptions.virtualizationRuntimes}
                      />
                    </section>
                  </div>

                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      {t("distribution.detail.config.targetEnvironment")}
                    </h4>
                    <section class="space-y-1">
                      <ConfigRow
                        label={t("distribution.detail.config.distributionType")}
                        value={config()!.target.type}
                        options={configOptions.distributionTypes}
                      />
                      <Show when={config()!.target.desktop}>
                        <ConfigRow
                          label={t(
                            "distribution.detail.config.desktopEnvironment",
                          )}
                          value={config()!.target.desktop?.environment}
                          version={
                            config()!.target.desktop?.environment_version
                          }
                          options={configOptions.desktopEnvironments}
                        />
                        <ConfigRow
                          label={t("distribution.detail.config.displayServer")}
                          value={
                            config()!.target.desktop?.displayServer || "Wayland"
                          }
                          version={
                            config()!.target.desktop?.display_server_version
                          }
                        />
                      </Show>
                    </section>
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
                      <p class="text-sm">
                        {t("distribution.detail.config.noConfig")}
                      </p>
                    </div>
                  </div>
                </Show>

                {/* Timestamps */}
                <div class="border-t border-border pt-4 mt-4 space-y-2">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">
                      {t("distribution.detail.created")}
                    </span>
                    <span class="font-mono text-xs">
                      {formatDate(distribution()!.created_at)}
                    </span>
                  </div>
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">
                      {t("distribution.detail.updated")}
                    </span>
                    <span class="font-mono text-xs">
                      {formatDate(distribution()!.updated_at)}
                    </span>
                  </div>
                  <Show when={distribution()!.owner_id}>
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">
                        {t("distribution.detail.ownerId")}
                      </span>
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
                        <span class="font-medium">
                          {t("distribution.detail.error")}
                        </span>
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
              <Card
                header={{ title: t("distribution.detail.quickActions.title") }}
              >
                <div class="space-y-3">
                  <button
                    class="w-full flex items-center gap-3 p-3 rounded-md border border-border hover:bg-muted transition-colors text-left"
                    onClick={() => setBuildDialogOpen(true)}
                  >
                    <Icon name="hammer" size="md" class="text-primary" />
                    <div>
                      <div class="font-medium">
                        {t("distribution.detail.quickActions.build")}
                      </div>
                      <div class="text-sm text-muted-foreground">
                        {t("distribution.detail.quickActions.buildDesc")}
                      </div>
                    </div>
                  </button>

                  <button
                    class="w-full flex items-center gap-3 p-3 rounded-md border border-border hover:bg-muted transition-colors text-left"
                    onClick={() => setKernelConfigModalOpen(true)}
                  >
                    <Icon name="gear-six" size="md" class="text-primary" />
                    <div>
                      <div class="font-medium">
                        {t(
                          "distribution.detail.quickActions.uploadKernelConfig",
                        )}
                      </div>
                      <div class="text-sm text-muted-foreground">
                        {t(
                          "distribution.detail.quickActions.uploadKernelConfigDesc",
                        )}
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
                      <div class="font-medium">
                        {t("distribution.detail.quickActions.download")}
                      </div>
                      <div class="text-sm text-muted-foreground">
                        {t("distribution.detail.quickActions.downloadDesc")}
                      </div>
                    </div>
                  </button>
                </div>
              </Card>

              {/* Recent Builds */}
              <Card
                header={{
                  title: t("build.list.title"),
                  actions: (
                    <button
                      onClick={() => setClearBuildsModalOpen(true)}
                      disabled={clearingBuilds()}
                      class="flex items-center gap-2 px-3 py-2 border border-border text-muted-foreground rounded-md hover:bg-muted hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                    >
                      <Show
                        when={!clearingBuilds()}
                        fallback={
                          <Icon
                            name="spinner-gap"
                            size="sm"
                            class="animate-spin"
                          />
                        }
                      >
                        <Icon name="trash" size="sm" />
                      </Show>
                      <span>{t("build.list.clear")}</span>
                    </button>
                  ),
                }}
              >
                <BuildsList
                  distributionId={props.distributionId}
                  onBuildClick={(buildId) => props.onNavigateToBuild?.(buildId)}
                  onRefetch={(fn) => {
                    refetchBuilds = fn;
                  }}
                  limit={5}
                />
              </Card>

              {/* Component Downloads */}
              <Card
                header={{
                  title: t("distribution.detail.componentDownloads.title"),
                  actions: (
                    <div class="flex items-center gap-2">
                      <Show when={downloadActions()?.hasJobs()}>
                        <button
                          onClick={() => downloadActions()?.flush()}
                          disabled={downloadActions()?.flushing()}
                          class="flex items-center gap-2 px-3 py-2 border border-border text-muted-foreground rounded-md hover:bg-muted hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                          <Show
                            when={!downloadActions()?.flushing()}
                            fallback={
                              <Icon
                                name="spinner-gap"
                                size="sm"
                                class="animate-spin"
                              />
                            }
                          >
                            <Icon name="trash" size="sm" />
                          </Show>
                          <span>{t("common.downloads.clear")}</span>
                        </button>
                      </Show>
                      <button
                        onClick={() => downloadActions()?.start()}
                        disabled={
                          downloadActions()?.starting() ||
                          downloadActions()?.hasActiveJobs()
                        }
                        class="flex items-center gap-2 px-3 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                      >
                        <Show
                          when={!downloadActions()?.starting()}
                          fallback={
                            <Icon
                              name="spinner-gap"
                              size="sm"
                              class="animate-spin"
                            />
                          }
                        >
                          <Icon name="cloud-arrow-down" size="sm" />
                        </Show>
                        <span>{t("common.downloads.startButton")}</span>
                      </button>
                    </div>
                  ),
                }}
              >
                <DownloadStatus
                  distributionId={props.distributionId}
                  onSuccess={handleDownloadSuccess}
                  onError={handleDownloadError}
                  onActions={setDownloadActions}
                  pollInterval={3000}
                />
              </Card>
            </div>
          </div>
        </Show>
      </section>

      {/* Edit Modal */}
      <Modal
        isOpen={editModalOpen()}
        onClose={handleEditCancel}
        title={t("distribution.editForm.modalTitle")}
      >
        <Show when={distribution()}>
          <DistributionEditForm
            distribution={distribution()!}
            onSubmit={handleEditSubmit}
            onCancel={handleEditCancel}
            isSubmitting={isSubmitting()}
          />
        </Show>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title={t("distribution.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("distribution.delete.confirmSingle", {
              name: distribution()?.name || "",
            })}
          </p>

          {/* Deletion Preview */}
          <Show when={loadingPreview()}>
            <div class="flex items-center justify-center py-4">
              <Spinner size="md" />
            </div>
          </Show>

          <Show when={!loadingPreview() && deletionPreview()}>
            {(preview) => {
              const hasRelatedItems =
                preview().download_jobs.count > 0 ||
                preview().artifacts.count > 0 ||
                preview().user_sources.count > 0;

              return (
                <Show when={hasRelatedItems}>
                  <div class="rounded-md border border-amber-500/30 bg-amber-500/10 p-4">
                    <div class="flex items-start gap-3">
                      <Icon
                        name="warning"
                        size="md"
                        class="text-amber-500 mt-0.5"
                      />
                      <div class="flex-1">
                        <h4 class="font-medium text-amber-500 mb-2">
                          {t("distribution.delete.cascadeWarning")}
                        </h4>
                        <ul class="space-y-2 text-sm">
                          <Show when={preview().download_jobs.count > 0}>
                            <li class="flex items-center gap-2">
                              <Icon
                                name="download"
                                size="sm"
                                class="text-muted-foreground"
                              />
                              <span>
                                {t("distribution.delete.downloadJobs", {
                                  count:
                                    preview().download_jobs.count.toString(),
                                })}
                              </span>
                            </li>
                          </Show>
                          <Show when={preview().artifacts.count > 0}>
                            <li class="flex items-center gap-2">
                              <Icon
                                name="file"
                                size="sm"
                                class="text-muted-foreground"
                              />
                              <span>
                                {t("distribution.delete.artifacts", {
                                  count: preview().artifacts.count.toString(),
                                })}
                              </span>
                            </li>
                          </Show>
                          <Show when={preview().user_sources.count > 0}>
                            <li class="flex items-center gap-2">
                              <Icon
                                name="database"
                                size="sm"
                                class="text-muted-foreground"
                              />
                              <span>
                                {t("distribution.delete.userSources", {
                                  count:
                                    preview().user_sources.count.toString(),
                                })}
                              </span>
                            </li>
                            <Show
                              when={
                                preview().user_sources.sources &&
                                preview().user_sources.sources!.length > 0
                              }
                            >
                              <ul class="ml-6 text-xs text-muted-foreground space-y-1">
                                <For each={preview().user_sources.sources}>
                                  {(source) => (
                                    <li class="flex items-center gap-1">
                                      <span class="w-1 h-1 rounded-full bg-muted-foreground" />
                                      <span>{source.name}</span>
                                    </li>
                                  )}
                                </For>
                              </ul>
                            </Show>
                          </Show>
                        </ul>
                      </div>
                    </div>
                  </div>
                </Show>
              );
            }}
          </Show>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelDelete}
              disabled={isDeleting()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={confirmDelete}
              disabled={isDeleting() || loadingPreview()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={isDeleting()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {isDeleting()
                  ? t("distribution.delete.deleting")
                  : t("common.actions.delete")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>

      {/* Build Start Dialog */}
      <BuildStartDialog
        isOpen={buildDialogOpen()}
        onClose={() => setBuildDialogOpen(false)}
        distributionId={props.distributionId}
        distributionName={distribution()?.name || ""}
        onBuildStarted={(buildId) => {
          showNotification("success", t("build.startDialog.success"));
          props.onNavigateToBuild?.(buildId);
        }}
      />

      {/* Kernel Config Upload Modal */}
      <Modal
        isOpen={kernelConfigModalOpen()}
        onClose={() => setKernelConfigModalOpen(false)}
        title={t("distribution.detail.quickActions.uploadKernelConfig")}
      >
        <section class="flex flex-col gap-4">
          <p class="text-sm text-muted-foreground">
            {t("distribution.detail.quickActions.uploadKernelConfigDesc")}
          </p>

          <label class="flex flex-col gap-2">
            <input
              type="file"
              accept=".config,*"
              class="block w-full text-sm text-muted-foreground
                file:mr-4 file:py-2 file:px-4
                file:rounded-md file:border file:border-border
                file:text-sm file:font-medium
                file:bg-muted file:text-foreground
                hover:file:bg-muted/80 file:cursor-pointer file:transition-colors"
              onChange={(e) => {
                const file = e.currentTarget.files?.[0];
                if (file) {
                  handleKernelConfigUpload(file);
                }
              }}
              disabled={uploadingConfig()}
            />
          </label>

          <Show when={uploadingConfig()}>
            <div class="flex flex-col gap-2">
              <div class="flex items-center gap-2">
                <Spinner size="sm" />
                <span class="text-sm text-muted-foreground">
                  {uploadProgress()}%
                </span>
              </div>
              <div class="w-full bg-muted rounded-full h-2">
                <div
                  class="bg-primary h-2 rounded-full transition-all"
                  style={{ width: `${uploadProgress()}%` }}
                />
              </div>
            </div>
          </Show>
        </section>
      </Modal>

      {/* Clear Builds Confirmation Modal */}
      <Modal
        isOpen={clearBuildsModalOpen()}
        onClose={() => setClearBuildsModalOpen(false)}
        title={t("build.list.clear")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">{t("build.list.clearConfirm")}</p>
          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setClearBuildsModalOpen(false)}
              disabled={clearingBuilds()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={confirmClearBuilds}
              disabled={clearingBuilds()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={clearingBuilds()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {clearingBuilds()
                  ? t("distribution.delete.deleting")
                  : t("build.list.clear")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
