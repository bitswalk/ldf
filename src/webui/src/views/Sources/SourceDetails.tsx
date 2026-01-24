import type { Component } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import { SourceForm } from "../../components/SourceForm";
import { VersionList } from "../../components/VersionList";
import { getSource, type SourceType } from "../../services/sourceVersions";
import {
  updateSource,
  updateDefaultSource,
  deleteSource,
  deleteDefaultSource,
  type CreateSourceRequest,
  type UpdateSourceRequest,
  type SourceDefault,
  type UserSource,
} from "../../services/sources";
import {
  listComponents,
  type Component as ComponentType,
} from "../../services/components";
import { t } from "../../services/i18n";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface SourceDetailsProps {
  sourceId: string;
  sourceType: SourceType;
  onBack: () => void;
  onDeleted?: () => void;
  user?: UserInfo | null;
}

export const SourceDetails: Component<SourceDetailsProps> = (props) => {
  const [source, setSource] = createSignal<SourceDefault | UserSource | null>(
    null,
  );
  const [components, setComponents] = createSignal<ComponentType[]>([]);
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

  const isAdmin = () => props.user?.role === "root";
  const isDefaultSource = () => props.sourceType === "default";

  const canEdit = () => {
    if (isDefaultSource()) {
      return isAdmin();
    }
    const src = source() as UserSource | null;
    return src && (props.user?.id === src.owner_id || isAdmin());
  };

  const canDelete = () => canEdit();

  const fetchSource = async () => {
    setLoading(true);
    setError(null);

    const result = await getSource(props.sourceId, props.sourceType);

    if (result.success) {
      setSource(result.source);
    } else {
      setError(result.message);
    }
    setLoading(false);
  };

  const fetchComponents = async () => {
    const result = await listComponents();
    if (result.success) {
      setComponents(result.components);
    }
  };

  onMount(() => {
    fetchSource();
    fetchComponents();
  });

  const getComponentName = (componentId: string | undefined): string => {
    if (!componentId) return t("sources.detail.notAssigned");
    const component = components().find((c) => c.id === componentId);
    return component?.display_name || componentId;
  };

  const getComponentNames = (componentIds: string[] | undefined): string[] => {
    if (!componentIds || componentIds.length === 0) return [];
    return componentIds.map((id) => {
      const component = components().find((c) => c.id === id);
      return component?.display_name || id;
    });
  };

  const getComponentCategory = (componentId: string | undefined): string => {
    if (!componentId) return "";
    const component = components().find((c) => c.id === componentId);
    return component?.category || "";
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const showNotification = (type: "success" | "error", message: string) => {
    setNotification({ type, message });
    setTimeout(() => setNotification(null), type === "success" ? 3000 : 5000);
  };

  const handleEdit = () => {
    setEditModalOpen(true);
  };

  const handleEditSubmit = async (formData: CreateSourceRequest) => {
    setIsSubmitting(true);
    setError(null);

    const updateReq: UpdateSourceRequest = {
      name: formData.name,
      url: formData.url,
      component_ids: formData.component_ids,
      retrieval_method: formData.retrieval_method,
      url_template: formData.url_template,
      forge_type: formData.forge_type,
      version_filter: formData.version_filter,
      priority: formData.priority,
      enabled: formData.enabled,
    };

    const result = isDefaultSource()
      ? await updateDefaultSource(props.sourceId, updateReq)
      : await updateSource(props.sourceId, updateReq);

    setIsSubmitting(false);

    if (result.success) {
      setEditModalOpen(false);
      fetchSource();
      showNotification("success", t("sources.detail.updateSuccess"));
    } else {
      setError(result.message);
    }
  };

  const handleEditCancel = () => {
    setEditModalOpen(false);
  };

  const handleDeleteClick = () => {
    setDeleteModalOpen(true);
  };

  const confirmDelete = async () => {
    setIsDeleting(true);
    setError(null);

    const result = isDefaultSource()
      ? await deleteDefaultSource(props.sourceId)
      : await deleteSource(props.sourceId);

    setIsDeleting(false);

    if (result.success) {
      setDeleteModalOpen(false);
      showNotification("success", t("sources.detail.deleteSuccess"));
      props.onDeleted?.();
      props.onBack();
    } else {
      setError(result.message);
    }
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
  };

  const handleVersionSuccess = (message: string) => {
    showNotification("success", message);
  };

  const handleVersionError = (message: string) => {
    showNotification("error", message);
  };

  // Convert source to form-compatible format
  const getFormInitialData = () => {
    const src = source();
    if (!src) return undefined;
    return {
      id: src.id,
      name: src.name,
      url: src.url,
      component_ids: src.component_ids,
      retrieval_method: src.retrieval_method,
      url_template: src.url_template,
      forge_type: src.forge_type,
      version_filter: src.version_filter,
      priority: src.priority,
      enabled: src.enabled,
      is_system: isDefaultSource(),
      created_at: src.created_at,
      updated_at: src.updated_at,
    };
  };

  return (
    <section class="h-full w-full relative">
      <section class="h-full flex flex-col p-8 gap-6">
        {/* Header */}
        <header class="flex items-center gap-4">
          <button
            onClick={props.onBack}
            class="p-2 rounded-md hover:bg-muted transition-colors"
            title={t("sources.detail.back")}
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <div class="flex items-center gap-3">
              <h1 class="text-4xl font-bold">
                {source()?.name || t("sources.detail.title")}
              </h1>
              <Show when={source()}>
                <span
                  class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                    isDefaultSource()
                      ? "bg-primary/10 text-primary"
                      : "bg-muted text-muted-foreground"
                  }`}
                >
                  {isDefaultSource()
                    ? t("sources.type.system")
                    : t("sources.type.user")}
                </span>
              </Show>
            </div>
            <p class="text-muted-foreground mt-1">
              <Show
                when={
                  source()?.component_ids && source()!.component_ids.length > 0
                }
              >
                <span class="flex items-center gap-2 flex-wrap">
                  <Icon name="cube" size="sm" />
                  {getComponentNames(source()?.component_ids).join(", ")}
                </span>
              </Show>
              <Show
                when={
                  !source()?.component_ids ||
                  source()!.component_ids.length === 0
                }
              >
                {t("sources.detail.subtitle")}
              </Show>
            </p>
          </div>
          <Show when={canEdit()}>
            <div class="flex items-center gap-2">
              <button
                onClick={handleEdit}
                class="flex items-center gap-2 px-4 py-2 border border-border rounded-md hover:bg-muted transition-colors"
              >
                <Icon name="pencil" size="sm" />
                <span>{t("sources.actions.edit")}</span>
              </button>
              <button
                onClick={handleDeleteClick}
                class="flex items-center gap-2 px-4 py-2 border border-destructive/50 text-destructive rounded-md hover:bg-destructive/10 transition-colors"
              >
                <Icon name="trash" size="sm" />
                <span>{t("sources.actions.delete")}</span>
              </button>
            </div>
          </Show>
        </header>

        {/* Notification */}
        <Show when={notification()}>
          <div
            class={`p-3 rounded-md ${
              notification()?.type === "success"
                ? "bg-green-500/10 border border-green-500/20 text-green-500"
                : "bg-red-500/10 border border-red-500/20 text-red-500"
            }`}
          >
            <div class="flex items-center gap-2">
              <Icon
                name={
                  notification()?.type === "success"
                    ? "check-circle"
                    : "warning-circle"
                }
                size="md"
              />
              <span>{notification()?.message}</span>
            </div>
          </div>
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
        <Show when={!loading() && source()}>
          <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Source Info - Left column */}
            <div class="lg:col-span-1 space-y-6">
              <Card header={{ title: t("sources.detail.sourceDetails") }}>
                <div class="space-y-4">
                  <div>
                    <span class="text-sm text-muted-foreground">
                      {t("sources.detail.name")}
                    </span>
                    <p class="font-medium">{source()!.name}</p>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">
                      {t("sources.detail.url")}
                    </span>
                    <p class="font-mono text-sm break-all">{source()!.url}</p>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">
                      {t("sources.detail.component")}
                    </span>
                    <Show
                      when={
                        source()!.component_ids &&
                        source()!.component_ids.length > 0
                      }
                    >
                      <div class="flex flex-wrap gap-1 mt-1">
                        {getComponentNames(source()!.component_ids).map(
                          (name) => (
                            <span class="inline-flex items-center px-2 py-0.5 bg-primary/10 text-primary text-xs rounded-full">
                              {name}
                            </span>
                          ),
                        )}
                      </div>
                    </Show>
                    <Show
                      when={
                        !source()!.component_ids ||
                        source()!.component_ids.length === 0
                      }
                    >
                      <p class="font-medium text-muted-foreground">
                        {t("sources.detail.notAssigned")}
                      </p>
                    </Show>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">
                      {t("sources.detail.retrievalMethod")}
                    </span>
                    <p class="font-medium capitalize">
                      {source()!.retrieval_method || "release"}
                    </p>
                  </div>

                  <Show when={source()!.url_template}>
                    <div>
                      <span class="text-sm text-muted-foreground">
                        {t("sources.detail.urlTemplate")}
                      </span>
                      <p class="font-mono text-sm break-all">
                        {source()!.url_template}
                      </p>
                    </div>
                  </Show>

                  <div class="flex items-center justify-between">
                    <span class="text-sm text-muted-foreground">
                      {t("sources.detail.priority")}
                    </span>
                    <span class="font-mono">{source()!.priority}</span>
                  </div>

                  <div class="flex items-center justify-between">
                    <span class="text-sm text-muted-foreground">
                      {t("sources.detail.status")}
                    </span>
                    <span
                      class={`flex items-center gap-2 ${source()!.enabled ? "text-green-500" : "text-muted-foreground"}`}
                    >
                      <Icon
                        name={source()!.enabled ? "check-circle" : "x-circle"}
                        size="sm"
                      />
                      {source()!.enabled
                        ? t("common.status.enabled")
                        : t("common.status.disabled")}
                    </span>
                  </div>

                  <div class="border-t border-border pt-4 mt-4 space-y-2">
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">
                        {t("sources.detail.created")}
                      </span>
                      <span class="font-mono text-xs">
                        {formatDate(source()!.created_at)}
                      </span>
                    </div>
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">
                        {t("sources.detail.updated")}
                      </span>
                      <span class="font-mono text-xs">
                        {formatDate(source()!.updated_at)}
                      </span>
                    </div>
                  </div>
                </div>
              </Card>
            </div>

            {/* Versions - Right columns */}
            <div class="lg:col-span-2">
              <Card header={{ title: t("sources.detail.availableVersions") }}>
                <VersionList
                  sourceId={props.sourceId}
                  sourceType={props.sourceType}
                  baseUrl={source()!.url}
                  urlTemplate={source()!.url_template}
                  versionFilter={source()!.version_filter}
                  onSuccess={handleVersionSuccess}
                  onError={handleVersionError}
                />
              </Card>
            </div>
          </div>
        </Show>
      </section>

      {/* Edit Modal */}
      <Modal
        isOpen={editModalOpen()}
        onClose={handleEditCancel}
        title={t("sources.create.editModalTitle")}
      >
        <SourceForm
          onSubmit={handleEditSubmit}
          onCancel={handleEditCancel}
          initialData={getFormInitialData()}
          isSubmitting={isSubmitting()}
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title={t("sources.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("sources.detail.deleteWarning", { name: source()?.name || "" })}
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
                  ? t("sources.delete.deleting")
                  : t("common.actions.delete")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
