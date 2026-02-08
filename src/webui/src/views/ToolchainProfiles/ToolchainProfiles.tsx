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
import { ToolchainProfileForm } from "../../components/ToolchainProfileForm";
import { t } from "../../services/i18n";
import {
  listToolchainProfiles,
  createToolchainProfile,
  deleteToolchainProfile,
  type ToolchainProfile,
  type CreateToolchainProfileRequest,
} from "../../services/toolchainProfiles";
import type { UserInfo } from "../../services/auth";

interface ToolchainProfilesProps {
  isLoggedIn: boolean;
  user: UserInfo | null;
  onViewProfile?: (id: string) => void;
}

export const ToolchainProfiles: Component<ToolchainProfilesProps> = (props) => {
  const [profiles, setProfiles] = createSignal<ToolchainProfile[]>([]);
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [typeFilter, setTypeFilter] = createSignal<string>("");
  const [selectedProfiles, setSelectedProfiles] = createSignal<
    ToolchainProfile[]
  >([]);

  // Create modal state
  const [formModalOpen, setFormModalOpen] = createSignal(false);
  const [formSubmitting, setFormSubmitting] = createSignal(false);
  const [formError, setFormError] = createSignal<string | null>(null);

  // Delete modal state
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [profilesToDelete, setProfilesToDelete] = createSignal<
    ToolchainProfile[]
  >([]);
  const [deleting, setDeleting] = createSignal(false);

  const fetchProfiles = async () => {
    setIsLoading(true);
    setError(null);

    const filter = typeFilter() || undefined;
    const result = await listToolchainProfiles(filter);

    if (result.success) {
      setProfiles(result.profiles);
    } else {
      setError(result.message);
    }

    setIsLoading(false);
  };

  onMount(() => {
    if (props.isLoggedIn) {
      fetchProfiles();
    }
  });

  const handleTypeFilterChange = (value: string) => {
    setTypeFilter(value);
    fetchProfiles();
  };

  const handleSelectionChange = (selected: ToolchainProfile[]) => {
    setSelectedProfiles(selected);
  };

  const openCreateModal = () => {
    setFormError(null);
    setFormModalOpen(true);
  };

  const closeFormModal = () => {
    setFormModalOpen(false);
    setFormError(null);
  };

  const handleFormSubmit = async (data: CreateToolchainProfileRequest) => {
    setFormSubmitting(true);
    setFormError(null);

    const result = await createToolchainProfile(data);
    if (result.success) {
      closeFormModal();
      fetchProfiles();
    } else {
      setFormError(result.message);
    }

    setFormSubmitting(false);
  };

  const openDeleteModal = (profile: ToolchainProfile) => {
    setProfilesToDelete([profile]);
    setDeleteModalOpen(true);
  };

  const handleDeleteSelected = () => {
    const selected = selectedProfiles().filter((p) => !p.is_system);
    if (selected.length === 0) return;
    setProfilesToDelete(selected);
    setDeleteModalOpen(true);
  };

  const confirmDelete = async () => {
    const toDelete = profilesToDelete();
    if (toDelete.length === 0) return;

    setDeleting(true);

    for (const profile of toDelete) {
      const result = await deleteToolchainProfile(profile.id);
      if (!result.success) {
        setError(result.message);
        setDeleting(false);
        setDeleteModalOpen(false);
        setProfilesToDelete([]);
        return;
      }
    }

    const deletedIds = new Set(toDelete.map((p) => p.id));
    setProfiles((prev) => prev.filter((p) => !deletedIds.has(p.id)));
    setSelectedProfiles([]);
    setDeleting(false);
    setDeleteModalOpen(false);
    setProfilesToDelete([]);
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
    setProfilesToDelete([]);
  };

  const ActionsCell: Component<{ value: unknown; row: ToolchainProfile }> = (
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
            <h1 class="text-4xl font-bold">{t("toolchainProfiles.title")}</h1>
            <p class="text-muted-foreground mt-2">
              {t("toolchainProfiles.description")}
            </p>
          </article>
          <nav class="flex items-center gap-4">
            {/* Type filter */}
            <select
              value={typeFilter()}
              onChange={(e) => handleTypeFilterChange(e.target.value)}
              class="px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary"
            >
              <option value="">{t("toolchainProfiles.filter.allTypes")}</option>
              <option value="gcc">GCC</option>
              <option value="llvm">LLVM/Clang</option>
            </select>

            {/* Bulk delete button */}
            <button
              onClick={handleDeleteSelected}
              disabled={selectedProfiles().length === 0}
              class={`px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors ${
                selectedProfiles().length > 0
                  ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  : "bg-muted text-muted-foreground cursor-not-allowed"
              }`}
            >
              <Icon name="trash" size="sm" />
              <span>
                {t("common.actions.delete")} ({selectedProfiles().length})
              </span>
            </button>

            {/* Create button */}
            <button
              onClick={openCreateModal}
              class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
            >
              <Icon name="plus" size="sm" />
              <span>{t("toolchainProfiles.create")}</span>
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
              when={profiles().length > 0}
              fallback={
                <section class="flex flex-col items-center justify-center py-16 text-muted-foreground gap-3">
                  <Icon name="wrench" size="xl" class="opacity-40" />
                  <h3 class="text-lg font-medium">
                    {t("toolchainProfiles.empty")}
                  </h3>
                  <p class="text-sm">
                    {t("toolchainProfiles.emptyDescription")}
                  </p>
                </section>
              }
            >
              <Datagrid
                columns={[
                  {
                    key: "display_name",
                    label: t("toolchainProfiles.columns.name"),
                    sortable: true,
                    class: "font-medium",
                    render: (value, row) => (
                      <button
                        onClick={() =>
                          props.onViewProfile?.((row as ToolchainProfile).id)
                        }
                        class="text-left hover:text-primary hover:underline transition-colors"
                      >
                        {value as string}
                      </button>
                    ),
                  },
                  {
                    key: "type",
                    label: t("toolchainProfiles.columns.type"),
                    sortable: true,
                    class: "font-mono text-sm",
                    render: (value) =>
                      (value as string) === "llvm" ? "LLVM/Clang" : "GCC",
                  },
                  {
                    key: "config",
                    label: t("toolchainProfiles.columns.crossCompilePrefix"),
                    render: (_value, row) =>
                      (row as ToolchainProfile).config?.cross_compile_prefix ||
                      "\u2014",
                    class: "font-mono text-sm text-muted-foreground",
                  },
                  {
                    key: "is_system",
                    label: t("toolchainProfiles.columns.system"),
                    sortable: true,
                    render: (value) =>
                      value ? (
                        <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary">
                          {t("toolchainProfiles.badges.system")}
                        </span>
                      ) : (
                        "\u2014"
                      ),
                  },
                  {
                    key: "owner_id",
                    label: t("toolchainProfiles.columns.owner"),
                    sortable: true,
                    class: "font-mono text-xs text-muted-foreground",
                    render: (value) => (value as string) || "\u2014",
                  },
                  {
                    key: "id",
                    label: t("toolchainProfiles.columns.actions"),
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
        isOpen={formModalOpen()}
        onClose={closeFormModal}
        title={t("toolchainProfiles.create")}
      >
        <ToolchainProfileForm
          onSubmit={handleFormSubmit}
          onCancel={closeFormModal}
          submitting={formSubmitting()}
          error={formError()}
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title={t("toolchainProfiles.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            <Show
              when={profilesToDelete().length === 1}
              fallback={t("toolchainProfiles.delete.confirmMultiple", {
                count: profilesToDelete().length.toString(),
              })}
            >
              {t("toolchainProfiles.delete.confirm", {
                name: profilesToDelete()[0]?.display_name || "",
              })}
            </Show>
          </p>

          <Show when={profilesToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto">
              {profilesToDelete().map((p) => (
                <li class="py-1">{p.display_name}</li>
              ))}
            </ul>
          </Show>

          <nav class="flex justify-end gap-3">
            <button
              type="button"
              onClick={cancelDelete}
              disabled={deleting()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="button"
              onClick={confirmDelete}
              disabled={deleting()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={deleting()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {deleting()
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
