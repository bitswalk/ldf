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
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [artifacts, setArtifacts] = createSignal<Artifact[]>([]);
  const [selectedArtifacts, setSelectedArtifacts] = createSignal<Artifact[]>(
    [],
  );
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [artifactsToDelete, setArtifactsToDelete] = createSignal<Artifact[]>(
    [],
  );
  const [isDeleting, setIsDeleting] = createSignal(false);
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
    setIsLoading(true);
    setError(null);

    const result = await listArtifacts();

    setIsLoading(false);

    if (result.success) {
      setArtifacts(result.artifacts);
    } else {
      if (result.error === "service_unavailable") {
        setError("Storage service is not configured on the server.");
      } else {
        setError(result.message);
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
    setArtifactsToDelete(arts);
    setDeleteModalOpen(true);
  };

  const handleDeleteArtifact = (artifact: Artifact) => {
    openDeleteModal([artifact]);
  };

  const handleSelectionChange = (selected: Artifact[]) => {
    setSelectedArtifacts(selected);
  };

  const handleDeleteSelected = () => {
    const selected = selectedArtifacts();
    if (selected.length === 0) return;
    openDeleteModal(selected);
  };

  const confirmDelete = async () => {
    const toDelete = artifactsToDelete();
    if (toDelete.length === 0) return;

    setIsDeleting(true);
    setError(null);

    for (const artifact of toDelete) {
      const result = await deleteArtifact(
        artifact.distribution_id,
        artifact.key,
      );
      if (!result.success) {
        setError(result.message);
        setIsDeleting(false);
        return;
      }
    }

    // Remove deleted artifacts from the list
    const deletedKeys = new Set(toDelete.map((a) => a.full_key));
    setArtifacts((prev) => prev.filter((a) => !deletedKeys.has(a.full_key)));
    setSelectedArtifacts([]);
    setIsDeleting(false);
    setDeleteModalOpen(false);
    setArtifactsToDelete([]);
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
    setArtifactsToDelete([]);
  };

  const handleDownload = async (artifact: Artifact) => {
    const result = await getArtifactURL(artifact.distribution_id, artifact.key);
    if (result.success) {
      // Prefer web URL if available, otherwise use presigned URL
      const downloadUrl = result.webUrl || result.url;
      window.open(downloadUrl, "_blank");
    } else {
      setError(result.message);
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
      setError(result.message);
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

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString();
  };

  const isAdmin = () => props.user?.role === "root";

  const filteredArtifacts = () => {
    if (showOnlyMine() && props.user?.id) {
      return artifacts().filter((a) => a.owner_id === props.user?.id);
    }
    return artifacts();
  };

  const renderSize = (size: number): JSX.Element => {
    return <span class="font-mono text-sm">{formatFileSize(size)}</span>;
  };

  const renderDistribution = (_: string, artifact: Artifact): JSX.Element => {
    return (
      <span class="flex items-center gap-2">
        <Icon name="cube" size="sm" class="text-muted-foreground" />
        <span>{artifact.distribution_name}</span>
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
            <span>Download</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => handleCopyUrl(cellProps.row)}
            class="gap-2"
          >
            <Icon name="link" size="sm" />
            <span>Copy URL</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => handleDeleteArtifact(cellProps.row)}
            class="gap-2 text-destructive focus:text-destructive"
          >
            <Icon name="trash" size="sm" />
            <span>Delete</span>
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
            <h1 class="text-4xl font-bold mb-4">Artifacts</h1>
            <p class="text-lg text-muted-foreground">
              Please log in to view and manage your build artifacts.
            </p>
          </section>
        }
      >
        <section class="h-full flex flex-col p-8 gap-6">
          <header class="flex items-center justify-between">
            <article>
              <h1 class="text-4xl font-bold">Artifacts</h1>
              <p class="text-muted-foreground mt-2">
                Manage your build artifacts and packages
              </p>
            </article>
            <nav class="flex items-center gap-4">
              <Show when={isAdmin()}>
                <label class="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer select-none">
                  <span>Show only mine</span>
                  <SummaryToggle
                    checked={showOnlyMine()}
                    onChange={setShowOnlyMine}
                  />
                </label>
              </Show>
              <button
                onClick={handleDeleteSelected}
                disabled={selectedArtifacts().length === 0}
                class={`px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors ${
                  selectedArtifacts().length > 0
                    ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    : "bg-muted text-muted-foreground cursor-not-allowed"
                }`}
              >
                <Icon name="trash" size="sm" />
                <span>Delete ({selectedArtifacts().length})</span>
              </button>
              <button
                onClick={openUploadModal}
                class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
              >
                <Icon name="plus" size="sm" />
                <span>New Artifact</span>
              </button>
            </nav>
          </header>

          <Show when={error()}>
            <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
              {error()}
            </aside>
          </Show>

          <section class="flex-1 overflow-visible">
            <Show
              when={!isLoading()}
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
                      No artifacts found
                    </h2>
                    <p class="text-sm text-muted-foreground mt-2">
                      Click "New Artifact" to upload your first artifact.
                    </p>
                  </section>
                }
              >
                <Datagrid
                  columns={[
                    {
                      key: "key",
                      label: "Name",
                      sortable: true,
                      class: "font-medium font-mono",
                    },
                    {
                      key: "distribution_name",
                      label: "Distribution",
                      sortable: true,
                      render: renderDistribution,
                    },
                    {
                      key: "size",
                      label: "Size",
                      sortable: true,
                      render: renderSize,
                    },
                    {
                      key: "content_type",
                      label: "Type",
                      sortable: true,
                      class: "font-mono text-xs",
                      render: (type: string) => type || "—",
                    },
                    {
                      key: "last_modified",
                      label: "Modified",
                      sortable: true,
                      class: "font-mono",
                      render: formatDate,
                    },
                    {
                      key: "full_key",
                      label: "Actions",
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

          <Show when={selectedArtifacts().length > 0}>
            <footer class="flex justify-end pt-4">
              <button
                onClick={handleDeleteSelected}
                class="px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                <Icon name="trash" size="sm" />
                <span>Delete Selected ({selectedArtifacts().length})</span>
              </button>
            </footer>
          </Show>
        </section>
      </Show>

      {/* Upload Modal */}
      <Modal
        isOpen={uploadModalOpen()}
        onClose={cancelUpload}
        title="Upload New Artifact"
      >
        <section class="flex flex-col gap-6">
          <Show when={uploadError()}>
            <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
              {uploadError()}
            </aside>
          </Show>

          {/* Distribution Select */}
          <article class="flex flex-col gap-2">
            <label class="text-sm font-medium">Distribution</label>
            <select
              value={selectedDistribution() ?? ""}
              onChange={(e) => {
                const val = e.target.value;
                setSelectedDistribution(val || null);
              }}
              disabled={isUploading()}
              class="px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
            >
              <option value="">Select a distribution...</option>
              <For each={distributions()}>
                {(dist) => <option value={dist.id}>{dist.name}</option>}
              </For>
            </select>
            <p class="text-xs text-muted-foreground">
              Choose the distribution this artifact belongs to.
            </p>
          </article>

          {/* File Input */}
          <article class="flex flex-col gap-2">
            <label class="text-sm font-medium">File</label>
            <input
              type="file"
              onChange={handleFileSelect}
              disabled={isUploading()}
              class="px-3 py-2 border border-border rounded-md bg-background text-foreground file:mr-4 file:py-1 file:px-3 file:rounded file:border-0 file:text-sm file:font-medium file:bg-primary file:text-primary-foreground hover:file:bg-primary/90 disabled:opacity-50"
            />
            <Show when={selectedFile()}>
              <p class="text-xs text-muted-foreground">
                Selected: {selectedFile()?.name} (
                {formatFileSize(selectedFile()?.size || 0)})
              </p>
            </Show>
          </article>

          {/* Custom Path (optional) */}
          <article class="flex flex-col gap-2">
            <label class="text-sm font-medium">
              Custom Path{" "}
              <span class="text-muted-foreground font-normal">(optional)</span>
            </label>
            <input
              type="text"
              value={customPath()}
              onInput={(e) => setCustomPath(e.currentTarget.value)}
              placeholder="e.g., boot/vmlinuz or packages/myapp.deb"
              disabled={isUploading()}
              class="px-3 py-2 border border-border rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50 font-mono text-sm"
            />
            <p class="text-xs text-muted-foreground">
              Leave empty to use the original filename.
            </p>
          </article>

          {/* Progress Bar */}
          <Show when={isUploading()}>
            <article class="flex flex-col gap-2">
              <section class="flex justify-between text-sm">
                <span>Uploading...</span>
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
              Cancel
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
              <span>{isUploading() ? "Uploading..." : "Upload"}</span>
            </button>
          </nav>
        </section>
      </Modal>

      {/* Delete Modal */}
      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title="Confirm Deletion"
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            Are you sure you want to delete{" "}
            <Show
              when={artifactsToDelete().length === 1}
              fallback={
                <span class="text-foreground font-medium">
                  {artifactsToDelete().length} artifacts
                </span>
              }
            >
              <span class="text-foreground font-medium font-mono">
                "{artifactsToDelete()[0]?.key}"
              </span>
            </Show>
            ? This action cannot be undone.
          </p>

          <Show when={artifactsToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto font-mono">
              {artifactsToDelete().map((artifact) => (
                <li class="py-1">{artifact.key}</li>
              ))}
            </ul>
          </Show>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelDelete}
              disabled={isDeleting()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={confirmDelete}
              disabled={isDeleting()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={isDeleting()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {isDeleting()
                  ? "Deleting..."
                  : `Delete${artifactsToDelete().length > 1 ? ` (${artifactsToDelete().length})` : ""}`}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
