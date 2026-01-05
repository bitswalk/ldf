import type { Component as SolidComponent } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import { ComponentForm } from "../../components/ComponentForm";
import {
  getComponent,
  updateComponent,
  deleteComponent,
  getCategoryDisplayName,
  getVersionRuleLabel,
  type Component,
  type CreateComponentRequest,
  type UpdateComponentRequest,
} from "../../services/components";
import { listSources, type Source } from "../../services/sources";
import { t } from "../../services/i18n";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface ComponentDetailsProps {
  componentId: string;
  onBack: () => void;
  onDeleted?: () => void;
  user?: UserInfo | null;
}

export const ComponentDetails: SolidComponent<ComponentDetailsProps> = (
  props,
) => {
  const [component, setComponent] = createSignal<Component | null>(null);
  const [relatedSources, setRelatedSources] = createSignal<Source[]>([]);
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

  const fetchComponent = async () => {
    setLoading(true);
    setError(null);

    const result = await getComponent(props.componentId);

    if (result.success) {
      setComponent(result.component);
    } else {
      setError(result.message);
    }
    setLoading(false);
  };

  const fetchRelatedSources = async () => {
    const result = await listSources();
    if (result.success) {
      // Filter sources that reference this component
      const filtered = result.sources.filter((s) =>
        s.component_ids.includes(props.componentId),
      );
      setRelatedSources(filtered);
    }
  };

  onMount(() => {
    fetchComponent();
    fetchRelatedSources();
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
    setEditModalOpen(true);
  };

  const handleEditSubmit = async (formData: CreateComponentRequest) => {
    setIsSubmitting(true);
    setError(null);

    const updateReq: UpdateComponentRequest = {
      name: formData.name,
      category: formData.category,
      display_name: formData.display_name,
      description: formData.description,
      artifact_pattern: formData.artifact_pattern,
      default_url_template: formData.default_url_template,
      github_normalized_template: formData.github_normalized_template,
      is_optional: formData.is_optional,
      default_version: formData.default_version,
      default_version_rule: formData.default_version_rule,
    };

    const result = await updateComponent(props.componentId, updateReq);

    setIsSubmitting(false);

    if (result.success) {
      setEditModalOpen(false);
      fetchComponent();
      showNotification("success", t("components.detail.updateSuccess"));
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

    const result = await deleteComponent(props.componentId);

    setIsDeleting(false);

    if (result.success) {
      setDeleteModalOpen(false);
      showNotification("success", t("components.detail.deleteSuccess"));
      props.onDeleted?.();
      props.onBack();
    } else {
      setError(result.message);
    }
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
  };

  const getCategoryColor = (category: string): string => {
    const colorMap: Record<string, string> = {
      core: "bg-blue-500/10 text-blue-500",
      bootloader: "bg-orange-500/10 text-orange-500",
      init: "bg-green-500/10 text-green-500",
      runtime: "bg-purple-500/10 text-purple-500",
      security: "bg-red-500/10 text-red-500",
      desktop: "bg-pink-500/10 text-pink-500",
    };
    return colorMap[category] || "bg-muted text-muted-foreground";
  };

  return (
    <section class="h-full w-full relative">
      <section class="h-full flex flex-col p-8 gap-6">
        {/* Header */}
        <header class="flex items-center gap-4">
          <button
            onClick={props.onBack}
            class="p-2 rounded-md hover:bg-muted transition-colors"
            title={t("components.detail.back")}
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <div class="flex items-center gap-3">
              <h1 class="text-4xl font-bold">
                {component()?.display_name || t("components.detail.title")}
              </h1>
              <Show when={component()}>
                <span
                  class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${getCategoryColor(component()!.category)}`}
                >
                  {getCategoryDisplayName(component()!.category)}
                </span>
                <span
                  class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                    component()!.is_optional
                      ? "bg-muted text-muted-foreground"
                      : "bg-primary/10 text-primary"
                  }`}
                >
                  {component()!.is_optional
                    ? t("components.table.optional")
                    : t("components.table.required")}
                </span>
              </Show>
            </div>
            <p class="text-muted-foreground mt-1 font-mono text-sm">
              {component()?.name || t("components.detail.subtitle")}
            </p>
          </div>
          <Show when={isAdmin()}>
            <div class="flex items-center gap-2">
              <button
                onClick={handleEdit}
                class="flex items-center gap-2 px-4 py-2 border border-border rounded-md hover:bg-muted transition-colors"
              >
                <Icon name="pencil" size="sm" />
                <span>{t("common.actions.edit")}</span>
              </button>
              <button
                onClick={handleDeleteClick}
                class="flex items-center gap-2 px-4 py-2 border border-destructive/50 text-destructive rounded-md hover:bg-destructive/10 transition-colors"
              >
                <Icon name="trash" size="sm" />
                <span>{t("common.actions.delete")}</span>
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
        <Show when={!loading() && component()}>
          <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Component Info - Left column */}
            <div class="lg:col-span-1 space-y-6">
              <Card header={{ title: t("components.detail.componentDetails") }}>
                <div class="space-y-4">
                  <div>
                    <span class="text-sm text-muted-foreground">
                      {t("components.detail.displayName")}
                    </span>
                    <p class="font-medium">{component()!.display_name}</p>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">
                      {t("components.detail.slug")}
                    </span>
                    <p class="font-mono text-sm">{component()!.name}</p>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">
                      {t("components.detail.identifier")}
                    </span>
                    <p class="font-mono text-xs text-muted-foreground break-all">
                      {component()!.id}
                    </p>
                  </div>

                  <div>
                    <span class="text-sm text-muted-foreground">
                      {t("components.detail.category")}
                    </span>
                    <p class="font-medium">
                      {getCategoryDisplayName(component()!.category)}
                    </p>
                  </div>

                  <Show when={component()!.description}>
                    <div>
                      <span class="text-sm text-muted-foreground">
                        {t("components.detail.description")}
                      </span>
                      <p class="text-sm">{component()!.description}</p>
                    </div>
                  </Show>

                  <div class="flex items-center justify-between">
                    <span class="text-sm text-muted-foreground">
                      {t("components.detail.type")}
                    </span>
                    <span
                      class={`flex items-center gap-2 ${
                        component()!.is_optional
                          ? "text-muted-foreground"
                          : "text-primary"
                      }`}
                    >
                      <Icon
                        name={
                          component()!.is_optional ? "circle" : "check-circle"
                        }
                        size="sm"
                      />
                      {component()!.is_optional
                        ? t("components.table.optional")
                        : t("components.table.required")}
                    </span>
                  </div>

                  {/* Version Info */}
                  <div class="border-t border-border pt-4 mt-4 space-y-3">
                    <div class="flex items-center justify-between">
                      <span class="text-sm text-muted-foreground">
                        {t("components.form.versionRule.label")}
                      </span>
                      <span class="text-sm font-medium">
                        {getVersionRuleLabel(component()!.default_version_rule)}
                      </span>
                    </div>
                    <Show
                      when={
                        component()!.default_version_rule === "pinned" &&
                        component()!.default_version
                      }
                    >
                      <div class="flex items-center justify-between">
                        <span class="text-sm text-muted-foreground">
                          {t("components.form.defaultVersion.label")}
                        </span>
                        <span class="font-mono text-sm">
                          {component()!.default_version}
                        </span>
                      </div>
                    </Show>
                  </div>

                  <div class="border-t border-border pt-4 mt-4 space-y-2">
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">
                        {t("components.detail.created")}
                      </span>
                      <span class="font-mono text-xs">
                        {formatDate(component()!.created_at)}
                      </span>
                    </div>
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">
                        {t("components.detail.updated")}
                      </span>
                      <span class="font-mono text-xs">
                        {formatDate(component()!.updated_at)}
                      </span>
                    </div>
                  </div>
                </div>
              </Card>
            </div>

            {/* Templates & Sources - Right columns */}
            <div class="lg:col-span-2 space-y-6">
              {/* URL Templates */}
              <Card header={{ title: t("components.detail.urlTemplates") }}>
                <div class="space-y-4">
                  <Show when={component()!.artifact_pattern}>
                    <div>
                      <span class="text-sm text-muted-foreground">
                        {t("components.detail.artifactPattern")}
                      </span>
                      <p class="font-mono text-sm bg-muted/50 p-2 rounded mt-1 break-all">
                        {component()!.artifact_pattern}
                      </p>
                    </div>
                  </Show>

                  <Show when={component()!.default_url_template}>
                    <div>
                      <span class="text-sm text-muted-foreground">
                        {t("components.detail.defaultUrlTemplate")}
                      </span>
                      <p class="font-mono text-sm bg-muted/50 p-2 rounded mt-1 break-all">
                        {component()!.default_url_template}
                      </p>
                    </div>
                  </Show>

                  <Show when={component()!.github_normalized_template}>
                    <div>
                      <span class="text-sm text-muted-foreground">
                        {t("components.detail.githubTemplate")}
                      </span>
                      <p class="font-mono text-sm bg-muted/50 p-2 rounded mt-1 break-all">
                        {component()!.github_normalized_template}
                      </p>
                    </div>
                  </Show>

                  <Show
                    when={
                      !component()!.artifact_pattern &&
                      !component()!.default_url_template &&
                      !component()!.github_normalized_template
                    }
                  >
                    <p class="text-muted-foreground text-sm italic">
                      {t("components.detail.noTemplates")}
                    </p>
                  </Show>
                </div>
              </Card>

              {/* Related Sources */}
              <Card header={{ title: t("components.detail.relatedSources") }}>
                <Show
                  when={relatedSources().length > 0}
                  fallback={
                    <p class="text-muted-foreground text-sm italic">
                      {t("components.detail.noSources")}
                    </p>
                  }
                >
                  <div class="space-y-3">
                    {relatedSources().map((source) => (
                      <div class="flex items-center justify-between p-3 bg-muted/30 rounded-md">
                        <div class="flex-1 min-w-0">
                          <div class="flex items-center gap-2">
                            <span class="font-medium truncate">
                              {source.name}
                            </span>
                            <span
                              class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                                source.is_system
                                  ? "bg-primary/10 text-primary"
                                  : "bg-muted text-muted-foreground"
                              }`}
                            >
                              {source.is_system
                                ? t("sources.type.system")
                                : t("sources.type.user")}
                            </span>
                          </div>
                          <p class="text-xs text-muted-foreground font-mono truncate mt-1">
                            {source.url}
                          </p>
                        </div>
                        <div class="flex items-center gap-2 ml-4">
                          <span
                            class={`flex items-center gap-1 text-sm ${
                              source.enabled
                                ? "text-green-500"
                                : "text-muted-foreground"
                            }`}
                          >
                            <Icon
                              name={
                                source.enabled ? "check-circle" : "x-circle"
                              }
                              size="sm"
                            />
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                </Show>
              </Card>
            </div>
          </div>
        </Show>
      </section>

      {/* Edit Modal */}
      <Modal
        isOpen={editModalOpen()}
        onClose={handleEditCancel}
        title={t("components.create.editModalTitle")}
      >
        <ComponentForm
          key={component()?.id || "edit"}
          onSubmit={handleEditSubmit}
          onCancel={handleEditCancel}
          initialData={component() || undefined}
          isSubmitting={isSubmitting()}
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title={t("components.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("components.detail.deleteWarning", {
              name: component()?.display_name || "",
            })}
          </p>

          <Show when={relatedSources().length > 0}>
            <aside class="p-3 bg-amber-500/10 border border-amber-500/20 rounded-md text-amber-600 text-sm">
              <div class="flex items-start gap-2">
                <Icon name="warning" size="sm" class="mt-0.5 flex-shrink-0" />
                <span>
                  {t("components.detail.deleteSourcesWarning", {
                    count: relatedSources().length,
                  })}
                </span>
              </div>
            </aside>
          </Show>

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
                  ? t("components.delete.deleting")
                  : t("common.actions.delete")}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
