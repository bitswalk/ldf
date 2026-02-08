import type { Component } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Datagrid } from "../../components/Datagrid";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import { ToolchainProfileForm } from "../../components/ToolchainProfileForm";
import { t } from "../../services/i18n";
import {
  listToolchainProfiles,
  createToolchainProfile,
  updateToolchainProfile,
  deleteToolchainProfile,
  type ToolchainProfile,
  type CreateToolchainProfileRequest,
  type UpdateToolchainProfileRequest,
} from "../../services/toolchainProfiles";
import type { UserInfo } from "../../services/auth";

interface ToolchainProfilesProps {
  isLoggedIn: boolean;
  user: UserInfo | null;
}

export const ToolchainProfiles: Component<ToolchainProfilesProps> = (props) => {
  const [profiles, setProfiles] = createSignal<ToolchainProfile[]>([]);
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [typeFilter, setTypeFilter] = createSignal<string>("");

  // Create/edit modal state
  const [formModalOpen, setFormModalOpen] = createSignal(false);
  const [editingProfile, setEditingProfile] = createSignal<
    ToolchainProfile | undefined
  >(undefined);
  const [formSubmitting, setFormSubmitting] = createSignal(false);
  const [formError, setFormError] = createSignal<string | null>(null);

  // Delete modal state
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [profileToDelete, setProfileToDelete] =
    createSignal<ToolchainProfile | null>(null);
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

  const openCreateModal = () => {
    setEditingProfile(undefined);
    setFormError(null);
    setFormModalOpen(true);
  };

  const openEditModal = (profile: ToolchainProfile) => {
    setEditingProfile(profile);
    setFormError(null);
    setFormModalOpen(true);
  };

  const closeFormModal = () => {
    setFormModalOpen(false);
    setEditingProfile(undefined);
    setFormError(null);
  };

  const handleFormSubmit = async (
    data: CreateToolchainProfileRequest | UpdateToolchainProfileRequest,
  ) => {
    setFormSubmitting(true);
    setFormError(null);

    const editing = editingProfile();
    if (editing) {
      const result = await updateToolchainProfile(
        editing.id,
        data as UpdateToolchainProfileRequest,
      );
      if (result.success) {
        closeFormModal();
        fetchProfiles();
      } else {
        setFormError(result.message);
      }
    } else {
      const result = await createToolchainProfile(
        data as CreateToolchainProfileRequest,
      );
      if (result.success) {
        closeFormModal();
        fetchProfiles();
      } else {
        setFormError(result.message);
      }
    }

    setFormSubmitting(false);
  };

  const openDeleteModal = (profile: ToolchainProfile) => {
    setProfileToDelete(profile);
    setDeleteModalOpen(true);
  };

  const confirmDelete = async () => {
    const profile = profileToDelete();
    if (!profile) return;

    setDeleting(true);

    const result = await deleteToolchainProfile(profile.id);
    if (result.success) {
      setDeleteModalOpen(false);
      setProfileToDelete(null);
      fetchProfiles();
    } else {
      setError(result.message);
      setDeleteModalOpen(false);
      setProfileToDelete(null);
    }

    setDeleting(false);
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
    setProfileToDelete(null);
  };

  const ActionsCell: Component<{ value: unknown; row: ToolchainProfile }> = (
    cellProps,
  ) => {
    return (
      <section class="flex items-center justify-end gap-1">
        <button
          onClick={() => openEditModal(cellProps.row)}
          class="p-1.5 rounded-md hover:bg-muted transition-colors"
          title={t("common.actions.edit")}
        >
          <Icon
            name="pencil-simple"
            size="sm"
            class="text-muted-foreground hover:text-foreground"
          />
        </button>
        <Show when={!cellProps.row.is_system}>
          <button
            onClick={() => openDeleteModal(cellProps.row)}
            class="p-1.5 rounded-md hover:bg-destructive/10 transition-colors"
            title={t("common.actions.delete")}
          >
            <Icon
              name="trash"
              size="sm"
              class="text-muted-foreground hover:text-destructive"
            />
          </button>
        </Show>
      </section>
    );
  };

  return (
    <section class="h-full flex flex-col">
      {/* Header */}
      <header class="shrink-0 px-6 py-4 border-b border-border">
        <section class="flex items-center justify-between">
          <div>
            <h1 class="text-2xl font-bold">
              {t("toolchainProfiles.title")}
            </h1>
            <p class="text-sm text-muted-foreground mt-1">
              {t("toolchainProfiles.description")}
            </p>
          </div>
          <div class="flex items-center gap-3">
            {/* Type filter */}
            <select
              value={typeFilter()}
              onChange={(e) => handleTypeFilterChange(e.target.value)}
              class="px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary"
            >
              <option value="">
                {t("toolchainProfiles.filter.allTypes")}
              </option>
              <option value="gcc">GCC</option>
              <option value="llvm">LLVM/Clang</option>
            </select>

            {/* Create button */}
            <button
              onClick={openCreateModal}
              class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors flex items-center gap-2"
            >
              <Icon name="plus" size="sm" />
              <span>{t("toolchainProfiles.create")}</span>
            </button>
          </div>
        </section>
      </header>

      {/* Content */}
      <section class="flex-1 overflow-auto p-6">
        <Show when={error()}>
          <aside class="p-3 mb-4 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm flex items-center justify-between">
            <span>{error()}</span>
            <button
              onClick={fetchProfiles}
              class="px-3 py-1 text-sm border border-destructive/30 rounded hover:bg-destructive/10 transition-colors"
            >
              {t("common.actions.retry")}
            </button>
          </aside>
        </Show>

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
                  label: t("toolchainProfiles.columns.displayName"),
                  sortable: true,
                  class: "font-medium",
                },
                {
                  key: "name",
                  label: t("toolchainProfiles.columns.name"),
                  sortable: true,
                  class: "font-mono text-sm text-muted-foreground",
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
                  label: t(
                    "toolchainProfiles.columns.crossCompilePrefix",
                  ),
                  render: (_value, row) =>
                    (row as ToolchainProfile).config
                      ?.cross_compile_prefix || "\u2014",
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
                  class: "text-right",
                  component: ActionsCell,
                },
              ]}
              data={profiles()}
              rowKey="id"
            />
          </Show>
        </Show>
      </section>

      {/* Create/Edit Modal */}
      <Modal
        isOpen={formModalOpen()}
        onClose={closeFormModal}
        title={
          editingProfile()
            ? t("toolchainProfiles.edit")
            : t("toolchainProfiles.create")
        }
      >
        <ToolchainProfileForm
          profile={editingProfile()}
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
            {t("toolchainProfiles.delete.confirm", {
              name: profileToDelete()?.display_name || "",
            })}
          </p>

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
