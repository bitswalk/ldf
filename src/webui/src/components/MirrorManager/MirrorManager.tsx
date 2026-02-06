import type { Component } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
import { Icon } from "../Icon";
import { Modal } from "../Modal";
import { Spinner } from "../Spinner";
import { t } from "../../services/i18n";
import {
  listMirrors,
  createMirror,
  updateMirror,
  deleteMirror,
  type Mirror,
  type CreateMirrorRequest,
  type UpdateMirrorRequest,
} from "../../services/mirrors";

export const MirrorManager: Component = () => {
  const [mirrors, setMirrors] = createSignal<Mirror[]>([]);
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  // Form modal state
  const [formModalOpen, setFormModalOpen] = createSignal(false);
  const [editingMirror, setEditingMirror] = createSignal<Mirror | null>(null);
  const [formSubmitting, setFormSubmitting] = createSignal(false);
  const [formError, setFormError] = createSignal<string | null>(null);

  // Form fields
  const [formName, setFormName] = createSignal("");
  const [formUrlPrefix, setFormUrlPrefix] = createSignal("");
  const [formMirrorUrl, setFormMirrorUrl] = createSignal("");
  const [formPriority, setFormPriority] = createSignal(0);
  const [formEnabled, setFormEnabled] = createSignal(true);

  // Delete modal state
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [mirrorToDelete, setMirrorToDelete] = createSignal<Mirror | null>(
    null,
  );
  const [deleting, setDeleting] = createSignal(false);

  const fetchMirrors = async () => {
    setIsLoading(true);
    setError(null);

    const result = await listMirrors();
    if (result.success) {
      setMirrors(result.mirrors);
    } else {
      setError(result.message);
    }

    setIsLoading(false);
  };

  onMount(() => {
    fetchMirrors();
  });

  const openCreateModal = () => {
    setEditingMirror(null);
    setFormName("");
    setFormUrlPrefix("");
    setFormMirrorUrl("");
    setFormPriority(0);
    setFormEnabled(true);
    setFormError(null);
    setFormModalOpen(true);
  };

  const openEditModal = (mirror: Mirror) => {
    setEditingMirror(mirror);
    setFormName(mirror.name);
    setFormUrlPrefix(mirror.url_prefix);
    setFormMirrorUrl(mirror.mirror_url);
    setFormPriority(mirror.priority);
    setFormEnabled(mirror.enabled);
    setFormError(null);
    setFormModalOpen(true);
  };

  const closeFormModal = () => {
    setFormModalOpen(false);
    setEditingMirror(null);
    setFormError(null);
  };

  const handleFormSubmit = async (e: SubmitEvent) => {
    e.preventDefault();
    setFormSubmitting(true);
    setFormError(null);

    const editing = editingMirror();
    if (editing) {
      const data: UpdateMirrorRequest = {
        name: formName(),
        url_prefix: formUrlPrefix(),
        mirror_url: formMirrorUrl(),
        priority: formPriority(),
        enabled: formEnabled(),
      };
      const result = await updateMirror(editing.id, data);
      if (result.success) {
        closeFormModal();
        fetchMirrors();
      } else {
        setFormError(result.message);
      }
    } else {
      const data: CreateMirrorRequest = {
        name: formName(),
        url_prefix: formUrlPrefix(),
        mirror_url: formMirrorUrl(),
        priority: formPriority(),
        enabled: formEnabled(),
      };
      const result = await createMirror(data);
      if (result.success) {
        closeFormModal();
        fetchMirrors();
      } else {
        setFormError(result.message);
      }
    }

    setFormSubmitting(false);
  };

  const handleToggleEnabled = async (mirror: Mirror) => {
    const result = await updateMirror(mirror.id, {
      enabled: !mirror.enabled,
    });
    if (result.success) {
      fetchMirrors();
    } else {
      setError(result.message);
    }
  };

  const openDeleteModal = (mirror: Mirror) => {
    setMirrorToDelete(mirror);
    setDeleteModalOpen(true);
  };

  const confirmDelete = async () => {
    const mirror = mirrorToDelete();
    if (!mirror) return;

    setDeleting(true);

    const result = await deleteMirror(mirror.id);
    if (result.success) {
      setDeleteModalOpen(false);
      setMirrorToDelete(null);
      fetchMirrors();
    } else {
      setError(result.message);
      setDeleteModalOpen(false);
      setMirrorToDelete(null);
    }

    setDeleting(false);
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
    setMirrorToDelete(null);
  };

  return (
    <div class="space-y-4">
      <Show when={error()}>
        <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm flex items-center justify-between">
          <span>{error()}</span>
          <button
            onClick={fetchMirrors}
            class="px-3 py-1 text-sm border border-destructive/30 rounded hover:bg-destructive/10 transition-colors"
          >
            {t("common.actions.retry")}
          </button>
        </aside>
      </Show>

      <Show
        when={!isLoading()}
        fallback={
          <div class="flex items-center justify-center py-8">
            <Spinner size="lg" />
          </div>
        }
      >
        {/* Mirror table */}
        <Show
          when={mirrors().length > 0}
          fallback={
            <div class="text-center py-8 text-muted-foreground">
              {t("settings.mirrors.empty")}
            </div>
          }
        >
          <div class="border border-border rounded-md overflow-hidden">
            <table class="w-full text-sm">
              <thead class="bg-muted/50">
                <tr>
                  <th class="px-4 py-2 text-left font-medium">
                    {t("settings.mirrors.columns.name")}
                  </th>
                  <th class="px-4 py-2 text-left font-medium">
                    {t("settings.mirrors.columns.urlPrefix")}
                  </th>
                  <th class="px-4 py-2 text-left font-medium">
                    {t("settings.mirrors.columns.mirrorUrl")}
                  </th>
                  <th class="px-4 py-2 text-left font-medium">
                    {t("settings.mirrors.columns.priority")}
                  </th>
                  <th class="px-4 py-2 text-left font-medium">
                    {t("settings.mirrors.columns.enabled")}
                  </th>
                  <th class="px-4 py-2 text-right font-medium">
                    {t("settings.mirrors.columns.actions")}
                  </th>
                </tr>
              </thead>
              <tbody>
                <For each={mirrors()}>
                  {(mirror) => (
                    <tr class="border-t border-border">
                      <td class="px-4 py-2 font-medium">{mirror.name}</td>
                      <td class="px-4 py-2 font-mono text-xs text-muted-foreground max-w-48 truncate">
                        {mirror.url_prefix}
                      </td>
                      <td class="px-4 py-2 font-mono text-xs text-muted-foreground max-w-48 truncate">
                        {mirror.mirror_url}
                      </td>
                      <td class="px-4 py-2 font-mono">
                        {mirror.priority}
                      </td>
                      <td class="px-4 py-2">
                        <button
                          type="button"
                          role="switch"
                          aria-checked={mirror.enabled}
                          onClick={() => handleToggleEnabled(mirror)}
                          class={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                            mirror.enabled ? "bg-primary" : "bg-muted"
                          }`}
                        >
                          <span
                            class={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-background shadow-lg ring-0 transition-transform ${
                              mirror.enabled
                                ? "translate-x-4"
                                : "translate-x-0"
                            }`}
                          />
                        </button>
                      </td>
                      <td class="px-4 py-2 text-right">
                        <section class="flex items-center justify-end gap-1">
                          <button
                            onClick={() => openEditModal(mirror)}
                            class="p-1.5 rounded-md hover:bg-muted transition-colors"
                            title={t("common.actions.edit")}
                          >
                            <Icon
                              name="pencil-simple"
                              size="sm"
                              class="text-muted-foreground hover:text-foreground"
                            />
                          </button>
                          <button
                            onClick={() => openDeleteModal(mirror)}
                            class="p-1.5 rounded-md hover:bg-destructive/10 transition-colors"
                            title={t("common.actions.delete")}
                          >
                            <Icon
                              name="trash"
                              size="sm"
                              class="text-muted-foreground hover:text-destructive"
                            />
                          </button>
                        </section>
                      </td>
                    </tr>
                  )}
                </For>
              </tbody>
            </table>
          </div>
        </Show>

        {/* Add mirror button */}
        <div class="pt-4">
          <button
            onClick={openCreateModal}
            class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors flex items-center gap-2 text-sm"
          >
            <Icon name="plus" size="sm" />
            <span>{t("settings.mirrors.add")}</span>
          </button>
        </div>
      </Show>

      {/* Create/Edit Modal */}
      <Modal
        isOpen={formModalOpen()}
        onClose={closeFormModal}
        title={
          editingMirror()
            ? t("settings.mirrors.editMirror")
            : t("settings.mirrors.addMirror")
        }
      >
        <form onSubmit={handleFormSubmit} class="flex flex-col gap-5">
          <Show when={formError()}>
            <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
              {formError()}
            </aside>
          </Show>

          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium" for="mirror-name">
              {t("settings.mirrors.form.name")}
            </label>
            <input
              id="mirror-name"
              type="text"
              value={formName()}
              onInput={(e) => setFormName(e.currentTarget.value)}
              required
              placeholder="kernel.org mirror"
              class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            />
          </div>

          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium" for="mirror-url-prefix">
              {t("settings.mirrors.form.urlPrefix")}
            </label>
            <input
              id="mirror-url-prefix"
              type="text"
              value={formUrlPrefix()}
              onInput={(e) => setFormUrlPrefix(e.currentTarget.value)}
              required
              placeholder="https://cdn.kernel.org/"
              class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono text-sm"
            />
            <p class="text-xs text-muted-foreground">
              {t("settings.mirrors.form.urlPrefixDescription")}
            </p>
          </div>

          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium" for="mirror-mirror-url">
              {t("settings.mirrors.form.mirrorUrl")}
            </label>
            <input
              id="mirror-mirror-url"
              type="text"
              value={formMirrorUrl()}
              onInput={(e) => setFormMirrorUrl(e.currentTarget.value)}
              required
              placeholder="https://mirror.example.com/kernel/"
              class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono text-sm"
            />
            <p class="text-xs text-muted-foreground">
              {t("settings.mirrors.form.mirrorUrlDescription")}
            </p>
          </div>

          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium" for="mirror-priority">
              {t("settings.mirrors.form.priority")}
            </label>
            <input
              id="mirror-priority"
              type="number"
              value={formPriority()}
              onInput={(e) =>
                setFormPriority(parseInt(e.currentTarget.value, 10) || 0)
              }
              class="w-24 px-3 py-2 rounded-md border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono"
            />
            <p class="text-xs text-muted-foreground">
              {t("settings.mirrors.form.priorityDescription")}
            </p>
          </div>

          <div class="flex items-center gap-3">
            <label class="text-sm font-medium" for="mirror-enabled">
              {t("settings.mirrors.form.enabled")}
            </label>
            <button
              id="mirror-enabled"
              type="button"
              role="switch"
              aria-checked={formEnabled()}
              onClick={() => setFormEnabled(!formEnabled())}
              class={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                formEnabled() ? "bg-primary" : "bg-muted"
              }`}
            >
              <span
                class={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-background shadow-lg ring-0 transition-transform ${
                  formEnabled() ? "translate-x-4" : "translate-x-0"
                }`}
              />
            </button>
          </div>

          <nav class="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={closeFormModal}
              disabled={formSubmitting()}
              class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {t("common.actions.cancel")}
            </button>
            <button
              type="submit"
              disabled={
                formSubmitting() ||
                !formName() ||
                !formUrlPrefix() ||
                !formMirrorUrl()
              }
              class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={formSubmitting()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {editingMirror()
                  ? t("common.actions.save")
                  : t("common.actions.create")}
              </span>
            </button>
          </nav>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title={t("settings.mirrors.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            {t("settings.mirrors.delete.confirm", {
              name: mirrorToDelete()?.name || "",
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
    </div>
  );
};
