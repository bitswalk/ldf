import type { Component, JSX } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
import { Datagrid } from "../../components/Datagrid";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "../../components/DropdownMenu";
import { SummaryToggle } from "../../components/Summary";
import {
  listArtifacts,
  deleteArtifact,
  getArtifactURL,
  uploadArtifact,
  formatFileSize,
  type Artifact,
} from "../../services/artifact";
import {
  listDistributions,
  type Distribution,
} from "../../services/distribution";
import { t } from "../../services/i18n";
import { isAdmin } from "../../utils/auth";
import { useListView } from "../../composables/useListView";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface ArtifactsProps {
  isLoggedIn?: boolean;
  user?: UserInfo | null;
}

export const Artifacts: Component<ArtifactsProps> = (props) => {
  const lv = useListView<Artifact>();
  const [artifacts, setArtifacts] = createSignal<Artifact[]>([]);
  const [showOnlyMine, setShowOnlyMine] = createSignal(false);

  // Upload modal state
  const [uploadModalOpen, setUploadModalOpen] = createSignal(false);
  const [distributions, setDistributions] = createSignal<Distribution[]>([]);
  const [selectedDistribution, setSelectedDistribution] = createSignal<
    string | null
  >(null);
  const [selectedFile, setSelectedFile] = createSignal<File | null>(null);
  const [customPath, setCustomPath] = createSignal("");
  const [isUploading, setIsUploading] = createSignal(false);
  const [uploadProgress, setUploadProgress] = createSignal(0);
  const [uploadError, setUploadError] = createSignal<string | null>(null);

  const fetchArtifacts = async () => {
    lv.setIsLoading(true);
    lv.setError(null);

    const result = await listArtifacts();

    lv.setIsLoading(false);

    if (result.success) {
      setArtifacts(result.artifacts);
    } else {
      if (result.error === "service_unavailable") {
        lv.setError(t("artifacts.errors.storageNotConfigured"));
      } else {
        lv.setError(result.message);
      }
    }
  };

  const fetchDistributions = async () => {
    const result = await listDistributions();
    if (result.success) {
      setDistributions(result.distributions);
    }
  };

  onMount(() => {
    if (props.isLoggedIn) {
      fetchArtifacts();
      fetchDistributions();
    }
  });

  const openDeleteModal = (arts: Artifact[]) => {
    lv.setItemsToDelete(arts);
    lv.openDeleteModal();
  };

  const handleDeleteArtifact = (artifact: Artifact) => {
    openDeleteModal([artifact]);
  };

  const handleSelectionChange = (selected: Artifact[]) => {
    lv.setSelected(selected);
  };

  const handleDeleteSelected = () => {
    const selected = lv.selected();
    if (selected.length === 0) return;
    openDeleteModal(selected);
  };

  const confirmDelete = async () => {
    const toDelete = lv.itemsToDelete();
    if (toDelete.length === 0) return;

    lv.setIsDeleting(true);
    lv.setError(null);

    for (const artifact of toDelete) {
      const result = await deleteArtifact(
        artifact.distribution_id,
        artifact.key,
      );
      if (!result.success) {
        lv.setError(result.message);
        lv.setIsDeleting(false);
        return;
      }
    }

    // Remove deleted artifacts from the list
    const deletedKeys = new Set(toDelete.map((a) => a.full_key));
    setArtifacts((prev) => prev.filter((a) => !deletedKeys.has(a.full_key)));
    lv.setSelected([]);
    lv.setIsDeleting(false);
    lv.closeDeleteModal();
    lv.setItemsToDelete([]);
  };

  const cancelDelete = () => {
    lv.closeDeleteModal();
    lv.setItemsToDelete([]);
  };

  const handleDownload = async (artifact: Artifact) => {
    const result = await getArtifactURL(artifact.distribution_id, artifact.key);
    if (result.success) {
      // Prefer web URL if available, otherwise use presigned URL
      const downloadUrl = result.webUrl || result.url;
      window.open(downloadUrl, "_blank");
    } else {
      lv.setError(result.message);
    }
  };

  const handleCopyUrl = async (artifact: Artifact) => {
    const result = await getArtifactURL(
      artifact.distribution_id,
      artifact.key,
      3600, // 1 hour expiry
    );
    if (result.success) {
      const url = result.webUrl || result.url;
      await navigator.clipboard.writeText(url);
    } else {
      lv.setError(result.message);
    }
  };

  // Upload handlers
  const openUploadModal = () => {
    setSelectedDistribution(null);
    setSelectedFile(null);
    setCustomPath("");
    setUploadProgress(0);
    setUploadError(null);
    setUploadModalOpen(true);
  };

  const cancelUpload = () => {
    setUploadModalOpen(false);
    setSelectedFile(null);
    setCustomPath("");
    setUploadError(null);
  };

  const handleFileSelect = (e: Event) => {
    const input = e.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      setSelectedFile(input.files[0]);
    }
  };

  const handleUpload = async () => {
    const distId = selectedDistribution();
    const file = selectedFile();

    if (!distId || !file) return;

    setIsUploading(true);
    setUploadProgress(0);
    setUploadError(null);

    const path = customPath().trim() || undefined;

    const result = await uploadArtifact(distId, file, path, (progress) => {
      setUploadProgress(progress);
    });

    setIsUploading(false);

    if (result.success) {
      setUploadModalOpen(false);
      fetchArtifacts(); // Refresh the list
    } else {
      setUploadError(result.message);
    }
  };

  const formatDate = (
    value: Artifact[keyof Artifact],
    _row: Artifact,
  ): JSX.Element => {
    const dateString = value as string;
    const date = new Date(dateString);
    return <span>{date.toLocaleDateString()}</span>;
  };

  const admin = () => isAdmin(props.user);

  const filteredArtifacts = () => {
    if (showOnlyMine() && props.user?.id) {
      return artifacts().filter((a) => a.owner_id === props.user?.id);
    }
    return artifacts();
  };

  const renderSize = (
    value: Artifact[keyof Artifact],
    _row: Artifact,
  ): JSX.Element => {
    const size = value as number;
    return <span class="font-mono text-sm">{formatFileSize(size)}</span>;
  };

  const renderDistribution = (
    _value: Artifact[keyof Artifact],
    row: Artifact,
  ): JSX.Element => {
    return (
      <span class="flex items-center gap-2">
        <Icon name="cube" size="sm" class="text-muted-foreground" />
        <span>{row.distribution_name}</span>
      </span>
    );
  };

  const ActionsCell: Component<{ value: any; row: Artifact }> = (cellProps) => {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger class="inline-flex items-center justify-center px-2 py-1 rounded-md hover:bg-muted transition-colors">
          <Icon
            name="dots-three-vertical"
            size="lg"
            class="text-muted-foreground hover:text-primary transition-colors"
          />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            onSelect={() => handleDownload(cellProps.row)}
            class="gap-2"
          >
            <Icon name="download" size="sm" />
            <span>{t("artifacts.actions.download")}</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => handleCopyUrl(cellProps.row)}
            class="gap-2"
          >
            <Icon name="link" size="sm" />
            <span>{t("artifacts.actions.copyUrl")}</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => handleDeleteArtifact(cellProps.row)}
            class="gap-2 text-destructive focus:text-destructive"
          >
            <Icon name="trash" size="sm" />
            <span>{t("common.actions.delete")}</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  };

  return (
    <section class="h-full w-full relative">
      <Show
        when={props.isLoggedIn}
        fallback={
          <section class="h-full flex flex-col items-center justify-center text-center p-8">
            <Icon
              name="package"
              size="2xl"
              class="text-muted-foreground mb-4"
            />
            <h1 class="text-4xl font-bold mb-4">{t("artifacts.title")}</h1>
            <p class="text-lg text-muted-foreground">
              {t("artifacts.welcome.loginRequired")}
            </p>
          </section>
        }
      >
        <section class="h-full flex flex-col p-8 gap-6">
          <header class="flex items-center justify-between">
            <article>
              <h1 class="text-4xl font-bold">{t("artifacts.title")}</h1>
              <p class="text-muted-foreground mt-2">
                {t("artifacts.subtitle")}
              </p>
            </article>
            <nav class="flex items-center gap-4">
              <Show when={admin()}>
                <label class="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer select-none">
                  <span>{t("artifacts.filter.showOnlyMine")}</span>
                  <SummaryToggle
                    checked={showOnlyMine()}
                    onChange={setShowOnlyMine}
                  />
                </label>
              </Show>
              <button
                onClick={handleDeleteSelected}
                disabled={lv.selected().length === 0}
                class={`px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors ${
                  lv.selected().length > 0
                    ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    : "bg-muted text-muted-foreground cursor-not-allowed"
                }`}
              >
                <Icon name="trash" size="sm" />
                <span>
                  {t("common.actions.delete")} ({lv.selected().length})
                </span>
              </button>
              <button
                onClick={openUploadModal}
                class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
              >
                <Icon name="plus" size="sm" />
                <span>{t("artifacts.actions.newArtifact")}</span>
              </button>
            </nav>
          </header>

          <Show when={lv.error()}>
            <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
              {lv.error()}
            </aside>
          </Show>

          <section class="flex-1 overflow-visible">
            <Show
              when={!lv.isLoading()}
              fallback={
                <section class="h-full flex items-center justify-center">
                  <Spinner size="lg" />
                </section>
              }
            >
              <Show
                when={filteredArtifacts().length > 0}
                fallback={
                  <section class="h-full flex flex-col items-center justify-center text-center">
                    <Icon
                      name="package"
                      size="2xl"
                      class="text-muted-foreground mb-4"
                    />
                    <h2 class="text-xl font-medium text-muted-foreground">
                      {t("artifacts.empty.title")}
                    </h2>
                    <p class="text-sm text-muted-foreground mt-2">
                      {t("artifacts.empty.description")}
                    </p>
                  </section>
                }
              >
                <Datagrid
                  columns={[
                    {
                      key: "key",
                      label: t("artifacts.table.columns.name"),
                      sortable: true,
                      class: "font-medium font-mono",
                    },
                    {
                      key: "distribution_name",
                      label: t("artifacts.table.columns.distribution"),
                      sortable: true,
                      render: renderDistribution,
                    },
                    {
                      key: "size",
                      label: t("artifacts.table.columns.size"),
                      sortable: true,
                      render: renderSize,
                    },
                    {
                      key: "content_type",
                      label: t("artifacts.table.columns.type"),
                      sortable: true,
                      class: "font-mono text-xs",
                      render: (value, _row) => (value as string) || "—",
                    },
                    {
                      key: "last_modified",
                      label: t("artifacts.table.columns.modified"),
                      sortable: true,
                      class: "font-mono",
                      render: formatDate,
                    },
                    {
                      key: "full_key",
                      label: t("artifacts.table.columns.actions"),
                      class: "text-right relative",
                      component: ActionsCell,
                    },
                  ]}
                  data={filteredArtifacts()}
                  rowKey="full_key"
                  selectable={true}
                  onSelectionChange={handleSelectionChange}
                />
              </Show>
            </Show>
          </section>

          <Show when={lv.selected().length > 0}>
            <footer class="flex justify-end pt-4">
              <button
                onClick={handleDeleteSelected}
                class="px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                <Icon name="trash" size="sm" />
                <span>
                  {t("artifacts.actions.deleteSelected", {
                    count: lv.selected().length,
                  })}
                </span>
              </button>
            </footer>
          </Show>
        </section>
      </Show>

      {/* Upload Modal */}
      <Modal
        isOpen={uploadModalOpen()}
        onClose={cancelUpload}
        title={t("artifacts.upload.title")}
      >
        <section class="flex flex-col gap-6">
          <Show when={uploadError()}>
            <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
              {uploadError()}
            </aside>
          </Show>

          {/* Distribution Select */}
          <article class="flex flex-col gap-2">
            <label class="text-sm font-medium">
              {t("artifacts.upload.distribution")}
            </label>
            <select
              value={selectedDistribution() ?? ""}
              onChange={(e) => {
                const val = e.target.value;
                setSelectedDistribution(val || null);
              }}
              disabled={isUploading()}
              class="px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
            >
              <option value="">
                {t("artifacts.upload.selectDistribution")}
              </option>
              <For each={distributions()}>
                {(dist) => <option value={dist.id}>{dist.name}</option>}
              </For>
            </select>
            <p class="text-xs text-muted-foreground">
              {t("artifacts.upload.distributionHelp")}
            </p>
          </article>

          {/* File Input */}
          <article class="flex flex-col gap-2">
            <label class="text-sm font-medium">
              {t("artifacts.upload.file")}
            </label>
            <input
              type="file"
              onChange={handleFileSelect}
              disabled={isUploading()}
              class="px-3 py-2 border border-border rounded-md bg-background text-foreground file:mr-4 file:py-1 file:px-3 file:rounded file:border-0 file:text-sm file:font-medium file:bg-primary file:text-primary-foreground hover:file:bg-primary/90 disabled:opacity-50"
            />
            <Show when={selectedFile()}>
              <p class="text-xs text-muted-foreground">
                {t("artifacts.upload.selected")}: {selectedFile()?.name} (
                {formatFileSize(selectedFile()?.size || 0)})
              </p>
            </Show>
          </article>

          {/* Custom Path (optional) */}
          <article class="flex flex-col gap-2">
            <label class="text-sm font-medium">
              {t("artifacts.upload.customPath")}{" "}
              <span class="text-muted-foreground font-normal">
                ({t("common.optional")})
              </span>
            </label>
            <input
              type="text"
              value={customPath()}
              onInput={(e) => setCustomPath(e.currentTarget.value)}
              placeholder={t("artifacts.upload.customPathPlaceholder")}
              disabled={isUploading()}
              class="px-3 py-2 border border-border rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50 font-mono text-sm"
            />
            <p class="text-xs text-muted-foreground">
              {t("artifacts.upload.customPathHelp")}
            </p>
          </article>

          {/* Progress Bar */}
          <Show when={isUploading()}>
            <article class="flex flex-col gap-2">
              <section class="flex justify-between text-sm">
                <span>{t("artifacts.upload.uploading")}</span>
                <span>{uploadProgress()}%</span>
              </section>
              <section class="w-full h-2 bg-muted rounded-full overflow-hidden">
                <section
                  class="h-full bg-primary transition-all duration-300"
                  style={{ width: `${uploadProgress()}%` }}
                />
              </section>
            </article>
          </Show>

          {/* Actions */}
          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelUpload}
              disabled={isUploading()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={handleUpload}
              disabled={
                isUploading() || !selectedDistribution() || !selectedFile()
              }
              class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show
                when={isUploading()}
                fallback={<Icon name="upload" size="sm" />}
              >
                <Spinner size="sm" />
              </Show>
              <span>
                {isUploading()
                  ? t("artifacts.upload.uploading")
                  : t("artifacts.upload.upload")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>

      {/* Delete Modal */}
      <Modal
        isOpen={lv.deleteModalOpen()}
        onClose={cancelDelete}
        title={t("artifacts.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            <Show
              when={lv.itemsToDelete().length === 1}
              fallback={
                <>
                  {t("artifacts.delete.confirmMultiple", {
                    count: lv.itemsToDelete().length,
                  })}
                </>
              }
            >
              {t("artifacts.delete.confirmSingle", {
                name: lv.itemsToDelete()[0]?.key || "",
              })}
            </Show>
          </p>

          <Show when={lv.itemsToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto font-mono">
              {lv.itemsToDelete().map((artifact) => (
                <li class="py-1">{artifact.key}</li>
              ))}
            </ul>
          </Show>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelDelete}
              disabled={lv.isDeleting()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={confirmDelete}
              disabled={lv.isDeleting()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={lv.isDeleting()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {lv.isDeleting()
                  ? t("artifacts.delete.deleting")
                  : `${t("common.actions.delete")}${lv.itemsToDelete().length > 1 ? ` (${lv.itemsToDelete().length})` : ""}`}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
