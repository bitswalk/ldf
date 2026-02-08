import type { Component } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Notification } from "../../components/Notification";
import { Modal } from "../../components/Modal";
import { ToolchainProfileForm } from "../../components/ToolchainProfileForm";
import {
  getToolchainProfile,
  updateToolchainProfile,
  deleteToolchainProfile,
  type ToolchainProfile,
  type UpdateToolchainProfileRequest,
  type CreateToolchainProfileRequest,
} from "../../services/toolchainProfiles";
import { t } from "../../services/i18n";
import { isAdmin } from "../../utils/auth";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface ToolchainProfileDetailProps {
  profileId: string;
  onBack: () => void;
  onDeleted?: () => void;
  user?: UserInfo | null;
}

export const ToolchainProfileDetail: Component<
  ToolchainProfileDetailProps
> = (props) => {
  const [profile, setProfile] = createSignal<ToolchainProfile | null>(null);
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
  const [formError, setFormError] = createSignal<string | null>(null);

  const admin = () => isAdmin(props.user);

  const fetchProfile = async () => {
    setLoading(true);
    setError(null);

    const result = await getToolchainProfile(props.profileId);

    if (result.success) {
      setProfile(result.profile);
    } else {
      setError(result.message);
    }
    setLoading(false);
  };

  onMount(() => {
    fetchProfile();
  });

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const showNotification = (type: "success" | "error", message: string) => {
    setNotification({ type, message });
    setTimeout(() => setNotification(null), type === "success" ? 3000 : 5000);
  };

  const handleEdit = () => {
    setFormError(null);
    setEditModalOpen(true);
  };

  const handleEditSubmit = async (
    data: CreateToolchainProfileRequest | UpdateToolchainProfileRequest,
  ) => {
    setIsSubmitting(true);
    setFormError(null);

    const result = await updateToolchainProfile(
      props.profileId,
      data as UpdateToolchainProfileRequest,
    );

    setIsSubmitting(false);

    if (result.success) {
      setEditModalOpen(false);
      fetchProfile();
      showNotification(
        "success",
        t("toolchainProfiles.detail.updateSuccess"),
      );
    } else {
      setFormError(result.message);
    }
  };

  const handleEditCancel = () => {
    setEditModalOpen(false);
    setFormError(null);
  };

  const handleDeleteClick = () => {
    setDeleteModalOpen(true);
  };

  const confirmDelete = async () => {
    setIsDeleting(true);
    setError(null);

    const result = await deleteToolchainProfile(props.profileId);

    setIsDeleting(false);

    if (result.success) {
      setDeleteModalOpen(false);
      showNotification(
        "success",
        t("toolchainProfiles.detail.deleteSuccess"),
      );
      props.onDeleted?.();
      props.onBack();
    } else {
      setError(result.message);
    }
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
  };

  const config = () => profile()?.config;

  const getTypeLabel = (type: string): string => {
    return type === "llvm" ? "LLVM/Clang" : "GCC";
  };

  const ConfigRow: Component<{
    label: string;
    value: string | undefined;
  }> = (rowProps) => (
    <article class="flex items-center justify-between py-2">
      <span class="text-muted-foreground text-sm">{rowProps.label}</span>
      <span class="text-sm font-medium font-mono">
        {rowProps.value || "\u2014"}
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
            class="size-10 flex items-center justify-center rounded-md hover:bg-muted transition-colors"
            title={t("toolchainProfiles.detail.back")}
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <div class="flex items-center gap-3">
              <h1 class="text-4xl font-bold">
                {profile()?.display_name ||
                  t("toolchainProfiles.detail.title")}
              </h1>
              <Show when={profile()}>
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary">
                  {getTypeLabel(profile()!.type)}
                </span>
                <Show when={profile()!.is_system}>
                  <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground">
                    {t("toolchainProfiles.badges.system")}
                  </span>
                </Show>
              </Show>
            </div>
            <p class="text-muted-foreground mt-1 font-mono text-sm">
              {profile()?.name || t("toolchainProfiles.detail.subtitle")}
            </p>
          </div>
          <Show when={admin()}>
            <div class="flex items-center gap-2">
              <button
                onClick={handleEdit}
                class="flex items-center gap-2 px-4 py-2 border border-border rounded-md hover:bg-muted transition-colors"
              >
                <Icon name="pencil" size="sm" />
                <span>{t("common.actions.edit")}</span>
              </button>
              <Show when={!profile()?.is_system}>
                <button
                  onClick={handleDeleteClick}
                  class="flex items-center gap-2 px-4 py-2 border border-destructive/50 text-destructive rounded-md hover:bg-destructive/10 transition-colors"
                >
                  <Icon name="trash" size="sm" />
                  <span>{t("common.actions.delete")}</span>
                </button>
              </Show>
            </div>
          </Show>
        </header>

        {/* Notification */}
        <Show when={notification()}>
          <Notification type={notification()!.type} message={notification()!.message} />
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
        <Show when={!loading() && profile()}>
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Profile Details */}
            <Card
              header={{
                title: t("toolchainProfiles.detail.profileDetails"),
              }}
            >
              <div class="space-y-4">
                <div>
                  <span class="text-sm text-muted-foreground">
                    {t("toolchainProfiles.detail.displayName")}
                  </span>
                  <p class="font-medium">{profile()!.display_name}</p>
                </div>

                <div>
                  <span class="text-sm text-muted-foreground">
                    {t("toolchainProfiles.detail.name")}
                  </span>
                  <p class="font-mono text-sm">{profile()!.name}</p>
                </div>

                <div>
                  <span class="text-sm text-muted-foreground">
                    {t("toolchainProfiles.detail.description")}
                  </span>
                  <p class="text-sm">
                    {profile()!.description ||
                      t("toolchainProfiles.detail.noDescription")}
                  </p>
                </div>

                <div class="flex items-center justify-between">
                  <span class="text-sm text-muted-foreground">
                    {t("toolchainProfiles.detail.type")}
                  </span>
                  <span class="text-sm font-medium">
                    {getTypeLabel(profile()!.type)}
                  </span>
                </div>

                <div class="flex items-center justify-between">
                  <span class="text-sm text-muted-foreground">
                    {t("toolchainProfiles.detail.system")}
                  </span>
                  <Icon
                    name={
                      profile()!.is_system ? "check-circle" : "x-circle"
                    }
                    size="sm"
                    class={
                      profile()!.is_system
                        ? "text-primary"
                        : "text-muted-foreground/50"
                    }
                  />
                </div>

                <Show when={profile()!.owner_id}>
                  <div class="flex items-center justify-between">
                    <span class="text-sm text-muted-foreground">
                      {t("toolchainProfiles.detail.owner")}
                    </span>
                    <span class="font-mono text-xs">
                      {profile()!.owner_id}
                    </span>
                  </div>
                </Show>

                <div class="border-t border-border pt-4 mt-4 space-y-2">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">
                      {t("toolchainProfiles.detail.created")}
                    </span>
                    <span class="font-mono text-xs">
                      {formatDate(profile()!.created_at)}
                    </span>
                  </div>
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">
                      {t("toolchainProfiles.detail.updated")}
                    </span>
                    <span class="font-mono text-xs">
                      {formatDate(profile()!.updated_at)}
                    </span>
                  </div>
                </div>
              </div>
            </Card>

            {/* Configuration */}
            <Card
              header={{
                title: t("toolchainProfiles.detail.configuration"),
              }}
            >
              <Show
                when={config()}
                fallback={
                  <div class="text-center py-8 text-muted-foreground">
                    <Icon
                      name="gear"
                      size="xl"
                      class="mx-auto mb-2 opacity-50"
                    />
                    <p class="text-sm">
                      {t("toolchainProfiles.detail.noConfig")}
                    </p>
                  </div>
                }
              >
                <div class="space-y-1">
                  <ConfigRow
                    label={t("toolchainProfiles.detail.crossCompilePrefix")}
                    value={config()!.cross_compile_prefix}
                  />
                  <ConfigRow
                    label={t("toolchainProfiles.detail.compilerFlags")}
                    value={config()!.compiler_flags}
                  />
                </div>

                <Show
                  when={
                    config()!.extra_env &&
                    Object.keys(config()!.extra_env!).length > 0
                  }
                >
                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      {t("toolchainProfiles.detail.extraEnv")}
                    </h4>
                    <div class="space-y-1">
                      <For each={Object.entries(config()!.extra_env!)}>
                        {([key, value]) => (
                          <div class="text-sm font-mono bg-muted/50 p-2 rounded">
                            {key}={value}
                          </div>
                        )}
                      </For>
                    </div>
                  </div>
                </Show>
              </Show>
            </Card>
          </div>
        </Show>
      </section>

      {/* Edit Modal */}
      <Modal
        isOpen={editModalOpen()}
        onClose={handleEditCancel}
        title={t("toolchainProfiles.edit")}
      >
        <Show when={profile()}>
          <ToolchainProfileForm
            profile={profile()!}
            onSubmit={handleEditSubmit}
            onCancel={handleEditCancel}
            submitting={isSubmitting()}
            error={formError()}
          />
        </Show>
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
              name: profile()?.display_name || "",
            })}
          </p>

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
              disabled={isDeleting()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={isDeleting()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {isDeleting()
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
