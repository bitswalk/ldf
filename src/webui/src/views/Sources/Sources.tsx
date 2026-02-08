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
import { t } from "../../services/i18n";
import { isAdmin } from "../../utils/auth";
import { useListView } from "../../composables/useListView";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface SourcesProps {
  isLoggedIn?: boolean;
  user?: UserInfo | null;
  onViewSource?: (sourceId: string) => void;
}

export const Sources: Component<SourcesProps> = (props) => {
  const lv = useListView<Source>();
  const [sources, setSources] = createSignal<Source[]>([]);
  const [showOnlyMine, setShowOnlyMine] = createSignal(false);
  const [editingSource, setEditingSource] = createSignal<Source | null>(null);

  const fetchSources = async () => {
    lv.setIsLoading(true);
    lv.setError(null);

    const result = await listSources();

    lv.setIsLoading(false);

    if (result.success) {
      setSources(result.sources);
    } else {
      lv.setError(result.message);
    }
  };

  onMount(() => {
    if (props.isLoggedIn) {
      fetchSources();
    }
  });

  const handleCreateSource = () => {
    setEditingSource(null);
    lv.openModal();
  };

  const handleFormSubmit = async (formData: CreateSourceRequest) => {
    lv.setIsSubmitting(true);
    lv.setError(null);

    const editing = editingSource();

    if (editing) {
      const updateReq: UpdateSourceRequest = {
        name: formData.name,
        url: formData.url,
        component_ids: formData.component_ids,
        retrieval_method: formData.retrieval_method,
        url_template: formData.url_template,
        priority: formData.priority,
        enabled: formData.enabled,
      };
      const result = await updateSource(editing.id, updateReq);

      lv.setIsSubmitting(false);

      if (result.success) {
        lv.closeModal();
        setEditingSource(null);
        fetchSources();
      } else {
        lv.setError(result.message);
      }
    } else {
      const result = await createSource(formData);

      lv.setIsSubmitting(false);

      if (result.success) {
        lv.closeModal();
        fetchSources();
      } else {
        lv.setError(result.message);
      }
    }
  };

  const handleFormCancel = () => {
    lv.closeModal();
    setEditingSource(null);
  };

  const handleEditSource = (source: Source) => {
    // Navigate to source details view for editing
    props.onViewSource?.(source.id);
  };

  const handleViewSource = (source: Source) => {
    props.onViewSource?.(source.id);
  };

  const openDeleteModal = (srcs: Source[]) => {
    // Admins can delete any source, non-admins can only delete user sources
    const isAdminUser = isAdmin(props.user);
    const deletableSources = isAdminUser
      ? srcs
      : srcs.filter((s) => !s.is_system);

    if (deletableSources.length === 0) {
      lv.setError(t("sources.delete.systemSourceError"));
      return;
    }
    lv.setItemsToDelete(deletableSources);
    lv.openDeleteModal();
  };

  const handleDeleteSource = (id: string) => {
    const source = sources().find((s) => s.id === id);
    if (source) {
      openDeleteModal([source]);
    }
  };

  const handleSelectionChange = (selected: Source[]) => {
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

    for (const source of toDelete) {
      const result = await deleteSource(source.id);
      if (!result.success) {
        lv.setError(result.message);
        lv.setIsDeleting(false);
        return;
      }
    }

    const deletedIds = new Set(toDelete.map((s) => s.id));
    setSources((prev) => prev.filter((s) => !deletedIds.has(s.id)));
    lv.setSelected([]);
    lv.setIsDeleting(false);
    lv.closeDeleteModal();
  };

  const cancelDelete = () => {
    lv.closeDeleteModal();
  };

  const formatDate = (
    value: Source[keyof Source],
    _row: Source,
  ): JSX.Element => {
    const dateString = value as string;
    const date = new Date(dateString);
    return <span>{date.toLocaleDateString()}</span>;
  };

  const admin = () => isAdmin(props.user);

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
      lv.setError(result.message);
    }
  };

  const renderEnabled = (
    value: Source[keyof Source],
    _row: Source,
  ): JSX.Element => {
    const enabled = value as boolean;
    return (
      <span
        class={`flex items-center gap-2 ${enabled ? "text-primary" : "text-muted-foreground"}`}
      >
        <Icon name={enabled ? "check-circle" : "x-circle"} size="sm" />
        <span>
          {enabled ? t("common.status.enabled") : t("common.status.disabled")}
        </span>
      </span>
    );
  };

  const renderSourceType = (
    value: Source[keyof Source],
    _row: Source,
  ): JSX.Element => {
    const isSystem = value as boolean;
    return (
      <span
        class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
          isSystem
            ? "bg-primary/10 text-primary"
            : "bg-muted text-muted-foreground"
        }`}
      >
        {isSystem ? t("sources.type.system") : t("sources.type.user")}
      </span>
    );
  };

  const ActionsCell: Component<{ value: any; row: Source }> = (cellProps) => {
    const canEdit = () => {
      return (
        !cellProps.row.is_system &&
        (props.user?.id === cellProps.row.owner_id || admin())
      );
    };

    const canEditSystem = () => {
      return cellProps.row.is_system && admin();
    };

    const canDelete = () => {
      // Admins can delete any source (including system sources)
      // Non-admins can only delete their own user sources
      if (admin()) return true;
      return (
        !cellProps.row.is_system && props.user?.id === cellProps.row.owner_id
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
            <span>{t("sources.actions.viewDetails")}</span>
          </DropdownMenuItem>
          <Show when={canEdit() || canEditSystem()}>
            <DropdownMenuItem
              onSelect={() => handleEditSource(cellProps.row)}
              class="gap-2"
            >
              <Icon name="pencil" size="sm" />
              <span>{t("sources.actions.edit")}</span>
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
              <span>
                {cellProps.row.enabled
                  ? t("sources.actions.disable")
                  : t("sources.actions.enable")}
              </span>
            </DropdownMenuItem>
          </Show>
          <Show when={canDelete()}>
            <DropdownMenuItem
              onSelect={() => handleDeleteSource(cellProps.row.id)}
              class="gap-2 text-destructive focus:text-destructive"
            >
              <Icon name="trash" size="sm" />
              <span>{t("sources.actions.delete")}</span>
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
            <h1 class="text-4xl font-bold mb-4">{t("sources.title")}</h1>
            <p class="text-lg text-muted-foreground mb-8">
              {t("sources.welcome.loginRequired")}
            </p>
          </section>
        }
      >
        <section class="h-full flex flex-col p-8 gap-6">
          <header class="flex items-center justify-between">
            <article>
              <h1 class="text-4xl font-bold">{t("sources.title")}</h1>
              <p class="text-muted-foreground mt-2">{t("sources.subtitle")}</p>
            </article>
            <nav class="flex items-center gap-4">
              <Show when={admin()}>
                <label class="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer select-none">
                  <span>{t("sources.filter.showOnlyMine")}</span>
                  <SummaryToggle
                    checked={showOnlyMine()}
                    onChange={setShowOnlyMine}
                  />
                </label>
              </Show>
              <button
                onClick={handleDeleteSelected}
                disabled={
                  (admin()
                    ? lv.selected().length
                    : lv.selected().filter((s) => !s.is_system).length) === 0
                }
                class={`px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors ${
                  (admin()
                    ? lv.selected().length
                    : lv.selected().filter((s) => !s.is_system).length) > 0
                    ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    : "bg-muted text-muted-foreground cursor-not-allowed"
                }`}
              >
                <Icon name="trash" size="sm" />
                <span>
                  {t("common.actions.delete")} (
                  {admin()
                    ? lv.selected().length
                    : lv.selected().filter((s) => !s.is_system).length}
                  )
                </span>
              </button>
              <button
                onClick={handleCreateSource}
                class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
              >
                <Icon name="plus" size="sm" />
                <span>{t("sources.create.button")}</span>
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
                when={sources().length > 0}
                fallback={
                  <section class="h-full flex flex-col items-center justify-center text-center">
                    <Card
                      borderStyle="dashed"
                      header={{ title: t("sources.cardTitle") }}
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
                      label: t("sources.table.columns.name"),
                      sortable: true,
                      class: "font-medium",
                      render: (value, row) => (
                        <button
                          onClick={() => handleViewSource(row as Source)}
                          class="text-left hover:text-primary hover:underline transition-colors"
                        >
                          {value as string}
                        </button>
                      ),
                    },
                    {
                      key: "url",
                      label: t("sources.table.columns.url"),
                      sortable: true,
                      class: "font-mono text-sm",
                    },
                    {
                      key: "priority",
                      label: t("sources.table.columns.priority"),
                      sortable: true,
                      class: "font-mono text-center",
                    },
                    {
                      key: "enabled",
                      label: t("sources.table.columns.status"),
                      sortable: true,
                      render: renderEnabled,
                    },
                    {
                      key: "is_system",
                      label: t("sources.table.columns.type"),
                      sortable: true,
                      render: renderSourceType,
                    },
                    {
                      key: "created_at",
                      label: t("sources.table.columns.created"),
                      sortable: true,
                      class: "font-mono",
                      render: formatDate,
                    },
                    {
                      key: "id",
                      label: t("sources.table.columns.actions"),
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

          <Show
            when={
              (admin()
                ? lv.selected().length
                : lv.selected().filter((s) => !s.is_system).length) > 0
            }
          >
            <footer class="flex justify-end pt-4">
              <button
                onClick={handleDeleteSelected}
                class="px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                <Icon name="trash" size="sm" />
                <span>
                  {t("sources.delete.deleteSelected", {
                    count: admin()
                      ? lv.selected().length
                      : lv.selected().filter((s) => !s.is_system).length,
                  })}
                </span>
              </button>
            </footer>
          </Show>
        </section>
      </Show>

      <Modal
        isOpen={lv.isModalOpen()}
        onClose={handleFormCancel}
        title={
          editingSource()
            ? t("sources.create.editModalTitle")
            : t("sources.create.modalTitle")
        }
      >
        <SourceForm
          onSubmit={handleFormSubmit}
          onCancel={handleFormCancel}
          initialData={editingSource() || undefined}
          isSubmitting={lv.isSubmitting()}
          isAdmin={admin()}
        />
      </Modal>

      <Modal
        isOpen={lv.deleteModalOpen()}
        onClose={cancelDelete}
        title={t("sources.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            <Show
              when={lv.itemsToDelete().length === 1}
              fallback={
                <>
                  {t("sources.delete.confirmMultiple", {
                    count: lv.itemsToDelete().length,
                  })}
                </>
              }
            >
              {t("sources.delete.confirmSingle", {
                name: lv.itemsToDelete()[0]?.name || "",
              })}
            </Show>
          </p>

          <Show when={lv.itemsToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto">
              {lv.itemsToDelete().map((source) => (
                <li class="py-1">{source.name}</li>
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
                  ? t("sources.delete.deleting")
                  : lv.itemsToDelete().length > 1
                    ? t("sources.delete.deleteCount", {
                        count: lv.itemsToDelete().length,
                      })
                    : t("common.actions.delete")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
