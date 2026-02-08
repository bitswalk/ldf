import type { Component as SolidComponent, JSX } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
import { Datagrid } from "../../components/Datagrid";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import { ComponentForm } from "../../components/ComponentForm";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "../../components/DropdownMenu";
import {
  listComponents,
  createComponent,
  updateComponent,
  deleteComponent,
  getCategoryDisplayName,
  type Component,
  type CreateComponentRequest,
  type UpdateComponentRequest,
} from "../../services/components";
import { t } from "../../services/i18n";
import { getCategoryColor } from "../../utils/categoryStyles";
import { isAdmin } from "../../utils/auth";
import { useListView } from "../../composables/useListView";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface ComponentsProps {
  isLoggedIn?: boolean;
  user?: UserInfo | null;
  onViewComponent?: (componentId: string) => void;
}

export const Components: SolidComponent<ComponentsProps> = (props) => {
  const lv = useListView<Component>();
  const [components, setComponents] = createSignal<Component[]>([]);
  const [editingComponent, setEditingComponent] =
    createSignal<Component | null>(null);
  const [categoryFilter, setCategoryFilter] = createSignal<string>("all");

  const fetchComponents = async () => {
    lv.setIsLoading(true);
    lv.setError(null);

    const result = await listComponents();

    lv.setIsLoading(false);

    if (result.success) {
      setComponents(result.components);
    } else {
      lv.setError(result.message);
    }
  };

  onMount(() => {
    fetchComponents();
  });

  const handleCreateComponent = () => {
    setEditingComponent(null);
    lv.openModal();
  };

  const handleFormSubmit = async (formData: CreateComponentRequest) => {
    lv.setIsSubmitting(true);
    lv.setError(null);

    const editing = editingComponent();

    if (editing) {
      const updateReq: UpdateComponentRequest = {
        name: formData.name,
        category: formData.category,
        display_name: formData.display_name,
        description: formData.description,
        artifact_pattern: formData.artifact_pattern,
        default_url_template: formData.default_url_template,
        github_normalized_template: formData.github_normalized_template,
        is_optional: formData.is_optional,
      };
      const result = await updateComponent(editing.id, updateReq);

      lv.setIsSubmitting(false);

      if (result.success) {
        lv.closeModal();
        setEditingComponent(null);
        fetchComponents();
      } else {
        lv.setError(result.message);
      }
    } else {
      const result = await createComponent(formData);

      lv.setIsSubmitting(false);

      if (result.success) {
        lv.closeModal();
        fetchComponents();
      } else {
        lv.setError(result.message);
      }
    }
  };

  const handleFormCancel = () => {
    lv.closeModal();
    setEditingComponent(null);
  };

  const handleEditComponent = (component: Component) => {
    setEditingComponent(component);
    lv.openModal();
  };

  const openDeleteModal = (comps: Component[]) => {
    if (comps.length === 0) return;
    lv.setItemsToDelete(comps);
    lv.openDeleteModal();
  };

  const handleDeleteComponent = (id: string) => {
    const component = components().find((c) => c.id === id);
    if (component) {
      openDeleteModal([component]);
    }
  };

  const handleSelectionChange = (selected: Component[]) => {
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

    for (const component of toDelete) {
      const result = await deleteComponent(component.id);
      if (!result.success) {
        lv.setError(result.message);
        lv.setIsDeleting(false);
        return;
      }
    }

    const deletedIds = new Set(toDelete.map((c) => c.id));
    setComponents((prev) => prev.filter((c) => !deletedIds.has(c.id)));
    lv.setSelected([]);
    lv.setIsDeleting(false);
    lv.closeDeleteModal();
  };

  const cancelDelete = () => {
    lv.closeDeleteModal();
  };

  const formatDate = (
    value: Component[keyof Component],
    _row: Component,
  ): JSX.Element => {
    const dateString = value as string;
    const date = new Date(dateString);
    return <span>{date.toLocaleDateString()}</span>;
  };

  const admin = () => isAdmin(props.user);

  const categories = () => {
    const cats = new Set(components().map((c) => c.category));
    return Array.from(cats).sort();
  };

  const filteredComponents = () => {
    const filter = categoryFilter();
    if (filter === "all") {
      return components();
    }
    return components().filter((c) => c.category === filter);
  };

  const renderCategory = (
    value: Component[keyof Component],
    _row: Component,
  ): JSX.Element => {
    const category = value as string;

    return (
      <span
        class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${getCategoryColor(category)}`}
      >
        {getCategoryDisplayName(category)}
      </span>
    );
  };

  const renderOptional = (
    value: Component[keyof Component],
    _row: Component,
  ): JSX.Element => {
    const isOptional = value as boolean;
    return (
      <span
        class={`flex items-center gap-2 ${isOptional ? "text-muted-foreground" : "text-primary"}`}
      >
        <Icon name={isOptional ? "circle" : "check-circle"} size="sm" />
        <span>
          {isOptional
            ? t("components.table.optional")
            : t("components.table.required")}
        </span>
      </span>
    );
  };

  const handleViewComponent = (component: Component) => {
    props.onViewComponent?.(component.id);
  };

  const ActionsCell: SolidComponent<{ value: any; row: Component }> = (
    cellProps,
  ) => {
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
            onSelect={() => handleViewComponent(cellProps.row)}
            class="gap-2"
          >
            <Icon name="eye" size="sm" />
            <span>{t("common.actions.viewDetails")}</span>
          </DropdownMenuItem>
          <Show when={admin()}>
            <DropdownMenuItem
              onSelect={() => handleEditComponent(cellProps.row)}
              class="gap-2"
            >
              <Icon name="pencil" size="sm" />
              <span>{t("common.actions.edit")}</span>
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={() => handleDeleteComponent(cellProps.row.id)}
              class="gap-2 text-destructive focus:text-destructive"
            >
              <Icon name="trash" size="sm" />
              <span>{t("common.actions.delete")}</span>
            </DropdownMenuItem>
          </Show>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  };

  return (
    <section class="h-full w-full relative">
      <section class="h-full flex flex-col p-8 gap-6">
        <header class="flex items-center justify-between">
          <article>
            <h1 class="text-4xl font-bold">{t("components.title")}</h1>
            <p class="text-muted-foreground mt-2">{t("components.subtitle")}</p>
          </article>
          <nav class="flex items-center gap-4">
            <select
              class="px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              value={categoryFilter()}
              onChange={(e) => setCategoryFilter(e.target.value)}
            >
              <option value="all">
                {t("components.filter.allCategories")}
              </option>
              <For each={categories()}>
                {(cat) => (
                  <option value={cat}>{getCategoryDisplayName(cat)}</option>
                )}
              </For>
            </select>
            <Show when={admin()}>
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
                onClick={handleCreateComponent}
                class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
              >
                <Icon name="plus" size="sm" />
                <span>{t("components.create.button")}</span>
              </button>
            </Show>
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
              when={filteredComponents().length > 0}
              fallback={
                <section class="h-full flex flex-col items-center justify-center text-center">
                  <Icon
                    name="cube"
                    size="2xl"
                    class="text-muted-foreground mb-4"
                  />
                  <h2 class="text-xl font-medium text-muted-foreground">
                    {t("components.empty.title")}
                  </h2>
                  <p class="text-sm text-muted-foreground mt-2">
                    {t("components.empty.description")}
                  </p>
                </section>
              }
            >
              <Datagrid
                columns={[
                  {
                    key: "display_name",
                    label: t("components.table.columns.name"),
                    sortable: true,
                    class: "font-medium",
                    render: (value, row) => (
                      <button
                        onClick={() => handleViewComponent(row as Component)}
                        class="text-left hover:text-primary hover:underline transition-colors"
                      >
                        {value as string}
                      </button>
                    ),
                  },
                  {
                    key: "name",
                    label: t("components.table.columns.slug"),
                    sortable: true,
                    class: "font-mono text-sm text-muted-foreground",
                  },
                  {
                    key: "category",
                    label: t("components.table.columns.category"),
                    sortable: true,
                    render: renderCategory,
                  },
                  {
                    key: "description",
                    label: t("components.table.columns.description"),
                    class: "max-w-xs truncate",
                    render: (value, _row) => (value as string) || "—",
                  },
                  {
                    key: "is_optional",
                    label: t("components.table.columns.type"),
                    sortable: true,
                    render: renderOptional,
                  },
                  {
                    key: "created_at",
                    label: t("components.table.columns.created"),
                    sortable: true,
                    class: "font-mono text-sm",
                    render: formatDate,
                  },
                  {
                    key: "id",
                    label: t("components.table.columns.actions"),
                    class: "text-right relative",
                    component: ActionsCell,
                  },
                ]}
                data={filteredComponents()}
                rowKey="id"
                selectable={admin()}
                onSelectionChange={handleSelectionChange}
              />
            </Show>
          </Show>
        </section>

        <Show when={admin() && lv.selected().length > 0}>
          <footer class="flex justify-end pt-4">
            <button
              onClick={handleDeleteSelected}
              class="px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              <Icon name="trash" size="sm" />
              <span>
                {t("components.delete.deleteSelected", {
                  count: lv.selected().length,
                })}
              </span>
            </button>
          </footer>
        </Show>
      </section>

      <Modal
        isOpen={lv.isModalOpen()}
        onClose={handleFormCancel}
        title={
          editingComponent()
            ? t("components.create.editModalTitle")
            : t("components.create.modalTitle")
        }
      >
        <ComponentForm
          onSubmit={handleFormSubmit}
          onCancel={handleFormCancel}
          initialData={editingComponent() || undefined}
          isSubmitting={lv.isSubmitting()}
        />
      </Modal>

      <Modal
        isOpen={lv.deleteModalOpen()}
        onClose={cancelDelete}
        title={t("components.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            <Show
              when={lv.itemsToDelete().length === 1}
              fallback={
                <>
                  {t("components.delete.confirmMultiple", {
                    count: lv.itemsToDelete().length,
                  })}
                </>
              }
            >
              {t("components.delete.confirmSingle", {
                name: lv.itemsToDelete()[0]?.display_name || "",
              })}
            </Show>
          </p>

          <aside class="p-3 bg-amber-500/10 border border-amber-500/20 rounded-md text-amber-600 text-sm">
            <div class="flex items-start gap-2">
              <Icon name="warning" size="sm" class="mt-0.5 flex-shrink-0" />
              <span>{t("components.delete.warning")}</span>
            </div>
          </aside>

          <Show when={lv.itemsToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto">
              {lv.itemsToDelete().map((component) => (
                <li class="py-1">{component.display_name}</li>
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
                  ? t("components.delete.deleting")
                  : lv.itemsToDelete().length > 1
                    ? t("components.delete.deleteCount", {
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
