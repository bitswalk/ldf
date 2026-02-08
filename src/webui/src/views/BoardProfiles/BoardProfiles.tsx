import type { Component } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Datagrid } from "../../components/Datagrid";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "../../components/DropdownMenu";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import { BoardProfileForm } from "../../components/BoardProfileForm";
import { t } from "../../services/i18n";
import { useListView } from "../../composables/useListView";
import {
  listBoardProfiles,
  createBoardProfile,
  deleteBoardProfile,
  type BoardProfile,
  type CreateBoardProfileRequest,
} from "../../services/boardProfiles";
import type { UserInfo } from "../../services/auth";

interface BoardProfilesProps {
  isLoggedIn: boolean;
  user: UserInfo | null;
  onViewProfile?: (id: string) => void;
}

export const BoardProfiles: Component<BoardProfilesProps> = (props) => {
  const lv = useListView<BoardProfile>();
  const [profiles, setProfiles] = createSignal<BoardProfile[]>([]);
  const [archFilter, setArchFilter] = createSignal<string>("");
  const [formError, setFormError] = createSignal<string | null>(null);

  const fetchProfiles = async () => {
    lv.setIsLoading(true);
    lv.setError(null);

    const filter = archFilter() || undefined;
    const result = await listBoardProfiles(filter);

    if (result.success) {
      setProfiles(result.profiles);
    } else {
      lv.setError(result.message);
    }

    lv.setIsLoading(false);
  };

  onMount(() => {
    if (props.isLoggedIn) {
      fetchProfiles();
    }
  });

  const handleArchFilterChange = (value: string) => {
    setArchFilter(value);
    fetchProfiles();
  };

  const handleSelectionChange = (selected: BoardProfile[]) => {
    lv.setSelected(selected);
  };

  const openCreateModal = () => {
    setFormError(null);
    lv.openModal();
  };

  const closeFormModal = () => {
    lv.closeModal();
    setFormError(null);
  };

  const handleFormSubmit = async (data: CreateBoardProfileRequest) => {
    lv.setIsSubmitting(true);
    setFormError(null);

    const result = await createBoardProfile(data);
    if (result.success) {
      closeFormModal();
      fetchProfiles();
    } else {
      setFormError(result.message);
    }

    lv.setIsSubmitting(false);
  };

  const openDeleteModal = (profile: BoardProfile) => {
    lv.setItemsToDelete([profile]);
    lv.openDeleteModal();
  };

  const handleDeleteSelected = () => {
    const selected = lv.selected().filter((p) => !p.is_system);
    if (selected.length === 0) return;
    lv.setItemsToDelete(selected);
    lv.openDeleteModal();
  };

  const confirmDelete = async () => {
    const toDelete = lv.itemsToDelete();
    if (toDelete.length === 0) return;

    lv.setIsDeleting(true);

    for (const profile of toDelete) {
      const result = await deleteBoardProfile(profile.id);
      if (!result.success) {
        lv.setError(result.message);
        lv.setIsDeleting(false);
        lv.closeDeleteModal();
        lv.setItemsToDelete([]);
        return;
      }
    }

    const deletedIds = new Set(toDelete.map((p) => p.id));
    setProfiles((prev) => prev.filter((p) => !deletedIds.has(p.id)));
    lv.setSelected([]);
    lv.setIsDeleting(false);
    lv.closeDeleteModal();
    lv.setItemsToDelete([]);
  };

  const cancelDelete = () => {
    lv.closeDeleteModal();
    lv.setItemsToDelete([]);
  };

  const ActionsCell: Component<{ value: unknown; row: BoardProfile }> = (
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
            onSelect={() => props.onViewProfile?.(cellProps.row.id)}
            class="gap-2"
          >
            <Icon name="eye" size="sm" />
            <span>{t("common.actions.viewDetails")}</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => props.onViewProfile?.(cellProps.row.id)}
            class="gap-2"
          >
            <Icon name="pencil" size="sm" />
            <span>{t("common.actions.edit")}</span>
          </DropdownMenuItem>
          <Show when={!cellProps.row.is_system}>
            <DropdownMenuItem
              onSelect={() => openDeleteModal(cellProps.row)}
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
            <h1 class="text-4xl font-bold">{t("boardProfiles.title")}</h1>
            <p class="text-muted-foreground mt-2">
              {t("boardProfiles.description")}
            </p>
          </article>
          <nav class="flex items-center gap-4">
            {/* Architecture filter */}
            <select
              value={archFilter()}
              onChange={(e) => handleArchFilterChange(e.target.value)}
              class="px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary"
            >
              <option value="">
                {t("boardProfiles.filter.allArchitectures")}
              </option>
              <option value="x86_64">x86_64</option>
              <option value="aarch64">aarch64</option>
            </select>

            {/* Bulk delete button */}
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

            {/* Create button */}
            <button
              onClick={openCreateModal}
              class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
            >
              <Icon name="plus" size="sm" />
              <span>{t("boardProfiles.create")}</span>
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
              when={profiles().length > 0}
              fallback={
                <section class="flex flex-col items-center justify-center py-16 text-muted-foreground gap-3">
                  <Icon name="cpu" size="xl" class="opacity-40" />
                  <h3 class="text-lg font-medium">
                    {t("boardProfiles.empty")}
                  </h3>
                  <p class="text-sm">{t("boardProfiles.emptyDescription")}</p>
                </section>
              }
            >
              <Datagrid
                columns={[
                  {
                    key: "display_name",
                    label: t("boardProfiles.columns.name"),
                    sortable: true,
                    class: "font-medium",
                    render: (value, row) => (
                      <button
                        onClick={() =>
                          props.onViewProfile?.((row as BoardProfile).id)
                        }
                        class="text-left hover:text-primary hover:underline transition-colors"
                      >
                        {value as string}
                      </button>
                    ),
                  },
                  {
                    key: "arch",
                    label: t("boardProfiles.columns.arch"),
                    sortable: true,
                    class: "font-mono text-sm",
                  },
                  {
                    key: "config",
                    label: t("boardProfiles.columns.defconfig"),
                    render: (_value, row) =>
                      (row as BoardProfile).config?.kernel_defconfig || "—",
                    class: "font-mono text-sm text-muted-foreground",
                  },
                  {
                    key: "is_system",
                    label: t("boardProfiles.columns.system"),
                    sortable: true,
                    render: (value) =>
                      value ? (
                        <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary">
                          {t("boardProfiles.badges.system")}
                        </span>
                      ) : (
                        "—"
                      ),
                  },
                  {
                    key: "owner_id",
                    label: t("boardProfiles.columns.owner"),
                    sortable: true,
                    class: "font-mono text-xs text-muted-foreground",
                    render: (value) => (value as string) || "—",
                  },
                  {
                    key: "id",
                    label: t("boardProfiles.columns.actions"),
                    class: "text-right relative",
                    component: ActionsCell,
                  },
                ]}
                data={profiles()}
                rowKey="id"
                selectable={true}
                onSelectionChange={handleSelectionChange}
              />
            </Show>
          </Show>
        </section>
      </section>

      {/* Create Modal */}
      <Modal
        isOpen={lv.isModalOpen()}
        onClose={closeFormModal}
        title={t("boardProfiles.create")}
      >
        <BoardProfileForm
          onSubmit={handleFormSubmit}
          onCancel={closeFormModal}
          submitting={lv.isSubmitting()}
          error={formError()}
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={lv.deleteModalOpen()}
        onClose={cancelDelete}
        title={t("boardProfiles.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            <Show
              when={lv.itemsToDelete().length === 1}
              fallback={t("boardProfiles.delete.confirmMultiple", {
                count: lv.itemsToDelete().length.toString(),
              })}
            >
              {t("boardProfiles.delete.confirm", {
                name: lv.itemsToDelete()[0]?.display_name || "",
              })}
            </Show>
          </p>

          <Show when={lv.itemsToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto">
              {lv.itemsToDelete().map((p) => (
                <li class="py-1">{p.display_name}</li>
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
                  ? t("common.status.deleting")
                  : t("common.actions.delete")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
