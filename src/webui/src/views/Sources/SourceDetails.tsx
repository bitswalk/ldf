import type { Component } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import { SourceForm } from "../../components/SourceForm";
import { VersionList } from "../../components/VersionList";
import {
  getSource,
  type SourceType,
} from "../../services/sourceVersions";
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
import { listComponents, type Component as ComponentType } from "../../services/components";

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
  const [source, setSource] = createSignal<SourceDefault | UserSource | null>(null);
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
    if (!componentId) return "Not assigned";
    const component = components().find((c) => c.id === componentId);
    return component?.display_name || componentId;
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
      component_id: formData.component_id,
      retrieval_method: formData.retrieval_method,
      url_template: formData.url_template,
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
      showNotification("success", "Source updated successfully");
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
      showNotification("success", "Source deleted successfully");
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
      component_id: src.component_id,
      retrieval_method: src.retrieval_method,
      url_template: src.url_template,
      priority: src.priority,
      enabled: src.enabled,
      is_system: isDefaultSource(),
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
            title="Back to sources"
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <div class="flex items-center gap-3">
              <h1 class="text-4xl font-bold">
                {source()?.name || "Source Details"}
              </h1>
              <Show when={source()}>
                <span
                  class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                    isDefaultSource()
                      ? "bg-primary/10 text-primary"
                      : "bg-muted text-muted-foreground"
                  }`}
                >
                  {isDefaultSource() ? "System" : "User"}
                </span>
              </Show>
            </div>
            <p class="text-muted-foreground mt-1">
              <Show when={source()?.component_id}>
                <span class="flex items-center gap-2">
                  <Icon name="cube" size="sm" />
                  {getComponentName(source()?.component_id)}
                  <Show when={getComponentCategory(source()?.component_id)}>
                    <span class="text-xs px-2 py-0.5 bg-muted rounded">
                      {getComponentCategory(source()?.component_id)}
                    </span>
                  </Show>
                </span>
              </Show>
              <Show when={!source()?.component_id}>
                View source details and sync available versions
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
                <span>Edit</span>
              </button>
              <button
                onClick={handleDeleteClick}
                class="flex items-center gap-2 px-4 py-2 border border-destructive/50 text-destructive rounded-md hover:bg-destructive/10 transition-colors"
              >
                <Icon name="trash" size="sm" />
                <span>Delete</span>
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
              <Card header={{ title: "Source Details" }}>
                <div class="space-y-4">
                  <div>
                    <span class="text-sm text-muted-foreground">Name</span>
                    <p class="font-medium">{source()!.name}</p>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">URL</span>
                    <p class="font-mono text-sm break-all">{source()!.url}</p>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">Component</span>
                    <p class="font-medium">
                      {getComponentName(source()!.component_id)}
                    </p>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">
                      Retrieval Method
                    </span>
                    <p class="font-medium capitalize">
                      {source()!.retrieval_method || "release"}
                    </p>
                  </div>

                  <Show when={source()!.url_template}>
                    <div>
                      <span class="text-sm text-muted-foreground">
                        URL Template
                      </span>
                      <p class="font-mono text-sm break-all">
                        {source()!.url_template}
                      </p>
                    </div>
                  </Show>

                  <div class="flex items-center justify-between">
                    <span class="text-sm text-muted-foreground">Priority</span>
                    <span class="font-mono">{source()!.priority}</span>
                  </div>

                  <div class="flex items-center justify-between">
                    <span class="text-sm text-muted-foreground">Status</span>
                    <span
                      class={`flex items-center gap-2 ${source()!.enabled ? "text-green-500" : "text-muted-foreground"}`}
                    >
                      <Icon
                        name={source()!.enabled ? "check-circle" : "x-circle"}
                        size="sm"
                      />
                      {source()!.enabled ? "Enabled" : "Disabled"}
                    </span>
                  </div>

                  <div class="border-t border-border pt-4 mt-4 space-y-2">
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">Created</span>
                      <span class="font-mono text-xs">
                        {formatDate(source()!.created_at)}
                      </span>
                    </div>
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">Updated</span>
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
              <Card header={{ title: "Available Versions" }}>
                <VersionList
                  sourceId={props.sourceId}
                  sourceType={props.sourceType}
                  baseUrl={source()!.url}
                  urlTemplate={source()!.url_template}
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
        title="Edit Source"
      >
        <SourceForm
          key={source()?.id || "edit"}
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
        title="Confirm Deletion"
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            Are you sure you want to delete{" "}
            <span class="text-foreground font-medium">
              "{source()?.name}"
            </span>
            ? This will also remove all cached versions. This action cannot be
            undone.
          </p>

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
              <span>{isDeleting() ? "Deleting..." : "Delete"}</span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
