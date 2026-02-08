import type { Component } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Notification } from "../../components/Notification";
import { Modal } from "../../components/Modal";
import { BoardProfileForm } from "../../components/BoardProfileForm";
import {
  getBoardProfile,
  updateBoardProfile,
  deleteBoardProfile,
  type BoardProfile,
  type UpdateBoardProfileRequest,
  type CreateBoardProfileRequest,
} from "../../services/boardProfiles";
import { t } from "../../services/i18n";
import { isAdmin } from "../../utils/auth";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface BoardProfileDetailProps {
  profileId: string;
  onBack: () => void;
  onDeleted?: () => void;
  user?: UserInfo | null;
}

export const BoardProfileDetail: Component<BoardProfileDetailProps> = (
  props,
) => {
  const [profile, setProfile] = createSignal<BoardProfile | null>(null);
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

    const result = await getBoardProfile(props.profileId);

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
    data: CreateBoardProfileRequest | UpdateBoardProfileRequest,
  ) => {
    setIsSubmitting(true);
    setFormError(null);

    const result = await updateBoardProfile(
      props.profileId,
      data as UpdateBoardProfileRequest,
    );

    setIsSubmitting(false);

    if (result.success) {
      setEditModalOpen(false);
      fetchProfile();
      showNotification("success", t("boardProfiles.detail.updateSuccess"));
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

    const result = await deleteBoardProfile(props.profileId);

    setIsDeleting(false);

    if (result.success) {
      setDeleteModalOpen(false);
      showNotification("success", t("boardProfiles.detail.deleteSuccess"));
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
            title={t("boardProfiles.detail.back")}
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <div class="flex items-center gap-3">
              <h1 class="text-4xl font-bold">
                {profile()?.display_name || t("boardProfiles.detail.title")}
              </h1>
              <Show when={profile()}>
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary">
                  {profile()!.arch}
                </span>
                <Show when={profile()!.is_system}>
                  <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground">
                    {t("boardProfiles.badges.system")}
                  </span>
                </Show>
              </Show>
            </div>
            <p class="text-muted-foreground mt-1 font-mono text-sm">
              {profile()?.name || t("boardProfiles.detail.subtitle")}
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
                title: t("boardProfiles.detail.profileDetails"),
              }}
            >
              <div class="space-y-4">
                <div>
                  <span class="text-sm text-muted-foreground">
                    {t("boardProfiles.detail.displayName")}
                  </span>
                  <p class="font-medium">{profile()!.display_name}</p>
                </div>

                <div>
                  <span class="text-sm text-muted-foreground">
                    {t("boardProfiles.detail.name")}
                  </span>
                  <p class="font-mono text-sm">{profile()!.name}</p>
                </div>

                <div>
                  <span class="text-sm text-muted-foreground">
                    {t("boardProfiles.detail.description")}
                  </span>
                  <p class="text-sm">
                    {profile()!.description ||
                      t("boardProfiles.detail.noDescription")}
                  </p>
                </div>

                <div class="flex items-center justify-between">
                  <span class="text-sm text-muted-foreground">
                    {t("boardProfiles.detail.arch")}
                  </span>
                  <span class="font-mono text-sm">{profile()!.arch}</span>
                </div>

                <div class="flex items-center justify-between">
                  <span class="text-sm text-muted-foreground">
                    {t("boardProfiles.detail.system")}
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
                      {t("boardProfiles.detail.owner")}
                    </span>
                    <span class="font-mono text-xs">
                      {profile()!.owner_id}
                    </span>
                  </div>
                </Show>

                <div class="border-t border-border pt-4 mt-4 space-y-2">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">
                      {t("boardProfiles.detail.created")}
                    </span>
                    <span class="font-mono text-xs">
                      {formatDate(profile()!.created_at)}
                    </span>
                  </div>
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">
                      {t("boardProfiles.detail.updated")}
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
                title: t("boardProfiles.detail.configuration"),
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
                      {t("boardProfiles.detail.noConfig")}
                    </p>
                  </div>
                }
              >
                <div class="space-y-1">
                  <ConfigRow
                    label={t("boardProfiles.detail.kernelDefconfig")}
                    value={config()!.kernel_defconfig}
                  />
                  <ConfigRow
                    label={t("boardProfiles.detail.kernelCmdline")}
                    value={config()!.kernel_cmdline}
                  />
                </div>

                <Show
                  when={
                    config()!.device_trees &&
                    config()!.device_trees!.length > 0
                  }
                >
                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      {t("boardProfiles.detail.deviceTrees")}
                    </h4>
                    <div class="space-y-2">
                      <For each={config()!.device_trees}>
                        {(dt) => (
                          <div class="text-sm font-mono bg-muted/50 p-2 rounded">
                            <p>{dt.source}</p>
                            <Show
                              when={dt.overlays && dt.overlays.length > 0}
                            >
                              <p class="text-xs text-muted-foreground mt-1">
                                Overlays: {dt.overlays!.join(", ")}
                              </p>
                            </Show>
                          </div>
                        )}
                      </For>
                    </div>
                  </div>
                </Show>

                <Show when={config()!.boot_params}>
                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      {t("boardProfiles.detail.bootParams")}
                    </h4>
                    <div class="space-y-1">
                      <Show when={config()!.boot_params!.bootloader_override}>
                        <ConfigRow
                          label="Bootloader Override"
                          value={config()!.boot_params!.bootloader_override}
                        />
                      </Show>
                      <Show when={config()!.boot_params!.uboot_board}>
                        <ConfigRow
                          label="U-Boot Board"
                          value={config()!.boot_params!.uboot_board}
                        />
                      </Show>
                      <Show when={config()!.boot_params!.config_txt}>
                        <div class="mt-2">
                          <span class="text-sm text-muted-foreground">
                            config.txt
                          </span>
                          <pre class="text-xs font-mono bg-muted/50 p-2 rounded mt-1 whitespace-pre-wrap">
                            {config()!.boot_params!.config_txt}
                          </pre>
                        </div>
                      </Show>
                    </div>
                  </div>
                </Show>

                <Show
                  when={
                    config()!.firmware && config()!.firmware!.length > 0
                  }
                >
                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      {t("boardProfiles.detail.firmware")}
                    </h4>
                    <div class="space-y-2">
                      <For each={config()!.firmware}>
                        {(fw) => (
                          <div class="text-sm bg-muted/50 p-2 rounded">
                            <p class="font-medium">{fw.name}</p>
                            <Show when={fw.description}>
                              <p class="text-xs text-muted-foreground mt-1">
                                {fw.description}
                              </p>
                            </Show>
                            <Show when={fw.path}>
                              <p class="text-xs font-mono text-muted-foreground mt-1">
                                {fw.path}
                              </p>
                            </Show>
                          </div>
                        )}
                      </For>
                    </div>
                  </div>
                </Show>

                <Show
                  when={
                    config()!.kernel_overlay &&
                    Object.keys(config()!.kernel_overlay!).length > 0
                  }
                >
                  <div class="border-t border-border pt-4 mt-4">
                    <h4 class="text-sm font-semibold text-muted-foreground mb-3">
                      Kernel Overlay
                    </h4>
                    <div class="space-y-1">
                      <For
                        each={Object.entries(config()!.kernel_overlay!)}
                      >
                        {([key, value]) => (
                          <ConfigRow label={key} value={value} />
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
        title={t("boardProfiles.edit")}
      >
        <Show when={profile()}>
          <BoardProfileForm
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
        title={t("boardProfiles.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("boardProfiles.delete.confirm", {
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
