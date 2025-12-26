import type { Component, JSX } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Card } from "../../components/Card";
import { Datagrid } from "../../components/Datagrid";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import { SourceForm } from "../../components/SourceForm";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "../../components/DropdownMenu";
import { SummaryToggle } from "../../components/Summary";
import {
  listSources,
  createSource,
  updateSource,
  deleteSource,
  type Source,
  type CreateSourceRequest,
  type UpdateSourceRequest,
} from "../../services/sources";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface SourcesProps {
  isLoggedIn?: boolean;
  user?: UserInfo | null;
  onViewSource?: (sourceId: string, sourceType: "default" | "user") => void;
}

export const Sources: Component<SourcesProps> = (props) => {
  const [isModalOpen, setIsModalOpen] = createSignal(false);
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [sources, setSources] = createSignal<Source[]>([]);
  const [selectedSources, setSelectedSources] = createSignal<Source[]>([]);
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [sourcesToDelete, setSourcesToDelete] = createSignal<Source[]>([]);
  const [isDeleting, setIsDeleting] = createSignal(false);
  const [showOnlyMine, setShowOnlyMine] = createSignal(false);
  const [editingSource, setEditingSource] = createSignal<Source | null>(null);
  const [isSubmitting, setIsSubmitting] = createSignal(false);

  const fetchSources = async () => {
    setIsLoading(true);
    setError(null);

    const result = await listSources();

    setIsLoading(false);

    if (result.success) {
      setSources(result.sources);
    } else {
      setError(result.message);
    }
  };

  onMount(() => {
    if (props.isLoggedIn) {
      fetchSources();
    }
  });

  const handleCreateSource = () => {
    setEditingSource(null);
    setIsModalOpen(true);
  };

  const handleFormSubmit = async (formData: CreateSourceRequest) => {
    setIsSubmitting(true);
    setError(null);

    const editing = editingSource();

    if (editing) {
      const updateReq: UpdateSourceRequest = {
        name: formData.name,
        url: formData.url,
        component_id: formData.component_id,
        retrieval_method: formData.retrieval_method,
        url_template: formData.url_template,
        priority: formData.priority,
        enabled: formData.enabled,
      };
      const result = await updateSource(editing.id, updateReq);

      setIsSubmitting(false);

      if (result.success) {
        setIsModalOpen(false);
        setEditingSource(null);
        fetchSources();
      } else {
        setError(result.message);
      }
    } else {
      const result = await createSource(formData);

      setIsSubmitting(false);

      if (result.success) {
        setIsModalOpen(false);
        fetchSources();
      } else {
        setError(result.message);
      }
    }
  };

  const handleFormCancel = () => {
    setIsModalOpen(false);
    setEditingSource(null);
  };

  const handleEditSource = (source: Source) => {
    // Navigate to source details view for editing
    const sourceType = source.is_system ? "default" : "user";
    props.onViewSource?.(source.id, sourceType);
  };

  const handleViewSource = (source: Source) => {
    const sourceType = source.is_system ? "default" : "user";
    props.onViewSource?.(source.id, sourceType);
  };

  const openDeleteModal = (srcs: Source[]) => {
    const userSources = srcs.filter((s) => !s.is_system);
    if (userSources.length === 0) {
      setError(
        "Cannot delete system sources. Only user sources can be deleted.",
      );
      return;
    }
    setSourcesToDelete(userSources);
    setDeleteModalOpen(true);
  };

  const handleDeleteSource = (id: string) => {
    const source = sources().find((s) => s.id === id);
    if (source) {
      openDeleteModal([source]);
    }
  };

  const handleSelectionChange = (selected: Source[]) => {
    setSelectedSources(selected);
  };

  const handleDeleteSelected = () => {
    const selected = selectedSources();
    if (selected.length === 0) return;
    openDeleteModal(selected);
  };

  const confirmDelete = async () => {
    const toDelete = sourcesToDelete();
    if (toDelete.length === 0) return;

    setIsDeleting(true);
    setError(null);

    for (const source of toDelete) {
      const result = await deleteSource(source.id);
      if (!result.success) {
        setError(result.message);
        setIsDeleting(false);
        return;
      }
    }

    const deletedIds = new Set(toDelete.map((s) => s.id));
    setSources((prev) => prev.filter((s) => !deletedIds.has(s.id)));
    setSelectedSources([]);
    setIsDeleting(false);
    setDeleteModalOpen(false);
    setSourcesToDelete([]);
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
    setSourcesToDelete([]);
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString();
  };

  const isAdmin = () => props.user?.role === "root";

  const filteredSources = () => {
    if (showOnlyMine() && props.user?.id) {
      return sources().filter((s) => s.owner_id === props.user?.id);
    }
    return sources();
  };

  const handleToggleEnabled = async (source: Source) => {
    if (source.is_system) {
      return;
    }

    const result = await updateSource(source.id, {
      enabled: !source.enabled,
    });

    if (result.success) {
      setSources((prev) =>
        prev.map((s) =>
          s.id === source.id ? { ...s, enabled: !s.enabled } : s,
        ),
      );
    } else {
      setError(result.message);
    }
  };

  const renderEnabled = (enabled: boolean): JSX.Element => {
    return (
      <span
        class={`flex items-center gap-2 ${enabled ? "text-primary" : "text-muted-foreground"}`}
      >
        <Icon name={enabled ? "check-circle" : "x-circle"} size="sm" />
        <span>{enabled ? "Enabled" : "Disabled"}</span>
      </span>
    );
  };

  const renderSourceType = (isSystem: boolean): JSX.Element => {
    return (
      <span
        class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
          isSystem
            ? "bg-primary/10 text-primary"
            : "bg-muted text-muted-foreground"
        }`}
      >
        {isSystem ? "System" : "User"}
      </span>
    );
  };

  const ActionsCell: Component<{ value: any; row: Source }> = (cellProps) => {
    const canEdit = () => {
      return (
        !cellProps.row.is_system &&
        (props.user?.id === cellProps.row.owner_id || isAdmin())
      );
    };

    const canEditSystem = () => {
      return cellProps.row.is_system && isAdmin();
    };

    const canDelete = () => {
      return (
        !cellProps.row.is_system &&
        (props.user?.id === cellProps.row.owner_id || isAdmin())
      );
    };

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
            onSelect={() => handleViewSource(cellProps.row)}
            class="gap-2"
          >
            <Icon name="eye" size="sm" />
            <span>View Details</span>
          </DropdownMenuItem>
          <Show when={canEdit() || canEditSystem()}>
            <DropdownMenuItem
              onSelect={() => handleEditSource(cellProps.row)}
              class="gap-2"
            >
              <Icon name="pencil" size="sm" />
              <span>Edit</span>
            </DropdownMenuItem>
          </Show>
          <Show when={canEdit()}>
            <DropdownMenuItem
              onSelect={() => handleToggleEnabled(cellProps.row)}
              class="gap-2"
            >
              <Icon
                name={cellProps.row.enabled ? "x-circle" : "check-circle"}
                size="sm"
              />
              <span>{cellProps.row.enabled ? "Disable" : "Enable"}</span>
            </DropdownMenuItem>
          </Show>
          <Show when={canDelete()}>
            <DropdownMenuItem
              onSelect={() => handleDeleteSource(cellProps.row.id)}
              class="gap-2 text-destructive focus:text-destructive"
            >
              <Icon name="trash" size="sm" />
              <span>Delete</span>
            </DropdownMenuItem>
          </Show>
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
            <h1 class="text-4xl font-bold mb-4">Sources</h1>
            <p class="text-lg text-muted-foreground mb-8">
              Please log in to view and manage upstream sources.
            </p>
          </section>
        }
      >
        <section class="h-full flex flex-col p-8 gap-6">
          <header class="flex items-center justify-between">
            <article>
              <h1 class="text-4xl font-bold">Sources</h1>
              <p class="text-muted-foreground mt-2">
                Manage upstream data sources for your distributions
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
                disabled={
                  selectedSources().filter((s) => !s.is_system).length === 0
                }
                class={`px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors ${
                  selectedSources().filter((s) => !s.is_system).length > 0
                    ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    : "bg-muted text-muted-foreground cursor-not-allowed"
                }`}
              >
                <Icon name="trash" size="sm" />
                <span>
                  Delete ({selectedSources().filter((s) => !s.is_system).length}
                  )
                </span>
              </button>
              <button
                onClick={handleCreateSource}
                class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
              >
                <Icon name="plus" size="sm" />
                <span>New Source</span>
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
                when={sources().length > 0}
                fallback={
                  <section class="h-full flex flex-col items-center justify-center text-center">
                    <Card
                      borderStyle="dashed"
                      header={{ title: "Add your first source" }}
                    >
                      <button
                        onClick={handleCreateSource}
                        class="cursor-pointer"
                      >
                        <Icon
                          name="plus"
                          size="2xl"
                          class="text-muted-foreground hover:text-primary transition-colors"
                        />
                      </button>
                    </Card>
                  </section>
                }
              >
                <Datagrid
                  columns={[
                    {
                      key: "name",
                      label: "Name",
                      sortable: true,
                      class: "font-medium",
                      render: (name: string, row: Source) => (
                        <button
                          onClick={() => handleViewSource(row)}
                          class="text-left hover:text-primary hover:underline transition-colors"
                        >
                          {name}
                        </button>
                      ),
                    },
                    {
                      key: "url",
                      label: "URL",
                      sortable: true,
                      class: "font-mono text-sm",
                    },
                    {
                      key: "priority",
                      label: "Priority",
                      sortable: true,
                      class: "font-mono text-center",
                    },
                    {
                      key: "enabled",
                      label: "Status",
                      sortable: true,
                      render: renderEnabled,
                    },
                    {
                      key: "is_system",
                      label: "Type",
                      sortable: true,
                      render: renderSourceType,
                    },
                    {
                      key: "created_at",
                      label: "Created",
                      sortable: true,
                      class: "font-mono",
                      render: formatDate,
                    },
                    {
                      key: "id",
                      label: "Actions",
                      class: "text-right relative",
                      component: ActionsCell,
                    },
                  ]}
                  data={filteredSources()}
                  rowKey="id"
                  selectable={true}
                  onSelectionChange={handleSelectionChange}
                />
              </Show>
            </Show>
          </section>

          <Show when={selectedSources().filter((s) => !s.is_system).length > 0}>
            <footer class="flex justify-end pt-4">
              <button
                onClick={handleDeleteSelected}
                class="px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                <Icon name="trash" size="sm" />
                <span>
                  Delete Selected (
                  {selectedSources().filter((s) => !s.is_system).length})
                </span>
              </button>
            </footer>
          </Show>
        </section>
      </Show>

      <Modal
        isOpen={isModalOpen()}
        onClose={handleFormCancel}
        title={editingSource() ? "Edit Source" : "Add New Source"}
      >
        <SourceForm
          key={editingSource()?.id || "new"}
          onSubmit={handleFormSubmit}
          onCancel={handleFormCancel}
          initialData={editingSource() || undefined}
          isSubmitting={isSubmitting()}
        />
      </Modal>

      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title="Confirm Deletion"
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            Are you sure you want to delete{" "}
            <Show
              when={sourcesToDelete().length === 1}
              fallback={
                <span class="text-foreground font-medium">
                  {sourcesToDelete().length} sources
                </span>
              }
            >
              <span class="text-foreground font-medium">
                "{sourcesToDelete()[0]?.name}"
              </span>
            </Show>
            ? This action cannot be undone.
          </p>

          <Show when={sourcesToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto">
              {sourcesToDelete().map((source) => (
                <li class="py-1">{source.name}</li>
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
                  : `Delete${sourcesToDelete().length > 1 ? ` (${sourcesToDelete().length})` : ""}`}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
