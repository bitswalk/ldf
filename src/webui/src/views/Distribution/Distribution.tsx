import type { Component, JSX } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
import { Card } from "../../components/Card";
import { Datagrid } from "../../components/Datagrid";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Modal } from "../../components/Modal";
import { DistributionForm } from "../../components/DistributionForm";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "../../components/DropdownMenu";
import { SummaryToggle } from "../../components/Summary";
import {
  listDistributions,
  deleteDistribution,
  createDistribution,
  updateDistribution,
  getDeletionPreview,
  type Distribution as DistributionType,
  type DistributionStatus,
  type DistributionVisibility,
  type DistributionConfig,
  type DeletionPreview,
} from "../../services/distribution";
import { t } from "../../services/i18n";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface DistributionProps {
  isLoggedIn?: boolean;
  user?: UserInfo | null;
  onViewDistribution?: (id: string) => void;
}

export const Distribution: Component<DistributionProps> = (props) => {
  const [isModalOpen, setIsModalOpen] = createSignal(false);
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [distributions, setDistributions] = createSignal<DistributionType[]>(
    [],
  );
  const [selectedDistributions, setSelectedDistributions] = createSignal<
    DistributionType[]
  >([]);
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [distributionsToDelete, setDistributionsToDelete] = createSignal<
    DistributionType[]
  >([]);
  const [isDeleting, setIsDeleting] = createSignal(false);
  const [showOnlyMine, setShowOnlyMine] = createSignal(false);
  const [deletionPreviews, setDeletionPreviews] = createSignal<
    DeletionPreview[]
  >([]);
  const [loadingPreviews, setLoadingPreviews] = createSignal(false);

  const fetchDistributions = async () => {
    setIsLoading(true);
    setError(null);

    const result = await listDistributions();

    setIsLoading(false);

    if (result.success) {
      setDistributions(result.distributions);
    } else {
      setError(result.message);
    }
  };

  onMount(() => {
    if (props.isLoggedIn) {
      fetchDistributions();
    }
  });

  const [isSubmitting, setIsSubmitting] = createSignal(false);

  const handleCreateDistribution = () => {
    setIsModalOpen(true);
  };

  const handleFormSubmit = async (formData: {
    name: string;
    core: DistributionConfig["core"];
    system: DistributionConfig["system"];
    security: DistributionConfig["security"];
    runtime: DistributionConfig["runtime"];
    target: DistributionConfig["target"];
  }) => {
    setIsSubmitting(true);
    setError(null);

    const result = await createDistribution({
      name: formData.name,
      config: {
        core: formData.core,
        system: formData.system,
        security: formData.security,
        runtime: formData.runtime,
        target: formData.target,
      },
    });

    setIsSubmitting(false);

    if (result.success) {
      setIsModalOpen(false);
      fetchDistributions();
    } else {
      setError(result.message);
    }
  };

  const handleFormCancel = () => {
    setIsModalOpen(false);
  };

  const handleEditDistribution = (_id: string) => {
    // TODO: Implement edit distribution
  };

  const openDeleteModal = async (dists: DistributionType[]) => {
    setDistributionsToDelete(dists);
    setDeleteModalOpen(true);
    setLoadingPreviews(true);
    setDeletionPreviews([]);

    // Fetch deletion previews for all distributions
    const previews: DeletionPreview[] = [];
    for (const dist of dists) {
      const result = await getDeletionPreview(dist.id);
      if (result.success) {
        previews.push(result.preview);
      }
    }
    setDeletionPreviews(previews);
    setLoadingPreviews(false);
  };

  const handleDeleteDistribution = (id: string) => {
    const dist = distributions().find((d) => d.id === id);
    if (dist) {
      openDeleteModal([dist]);
    }
  };

  const handleSelectionChange = (selected: DistributionType[]) => {
    setSelectedDistributions(selected);
  };

  const handleDeleteSelected = () => {
    const selected = selectedDistributions();
    if (selected.length === 0) return;
    openDeleteModal(selected);
  };

  const confirmDelete = async () => {
    const toDelete = distributionsToDelete();
    if (toDelete.length === 0) return;

    setIsDeleting(true);
    setError(null);

    for (const dist of toDelete) {
      const result = await deleteDistribution(dist.id);
      if (!result.success) {
        setError(result.message);
        setIsDeleting(false);
        return;
      }
    }

    const deletedIds = new Set(toDelete.map((d) => d.id));
    setDistributions((prev) => prev.filter((d) => !deletedIds.has(d.id)));
    setSelectedDistributions([]);
    setIsDeleting(false);
    setDeleteModalOpen(false);
    setDistributionsToDelete([]);
  };

  const cancelDelete = () => {
    setDeleteModalOpen(false);
    setDistributionsToDelete([]);
    setDeletionPreviews([]);
  };

  const formatDate = (
    value: DistributionType[keyof DistributionType],
    _row: DistributionType,
  ): JSX.Element => {
    const dateString = value as string;
    const date = new Date(dateString);
    return <span>{date.toLocaleDateString()}</span>;
  };

  const isAdmin = () => props.user?.role === "root";

  const filteredDistributions = () => {
    if (showOnlyMine() && props.user?.id) {
      return distributions().filter((d) => d.owner_id === props.user?.id);
    }
    return distributions();
  };

  const renderVisibility = (
    value: DistributionType[keyof DistributionType],
    _row: DistributionType,
  ): JSX.Element => {
    const visibility = value as DistributionVisibility;
    const isPublic = visibility === "public";
    return (
      <span
        class={`flex items-center gap-2 ${isPublic ? "text-primary" : "text-muted-foreground"}`}
      >
        <Icon name={isPublic ? "globe" : "lock"} size="sm" />
        <span>{t(`common.visibility.${visibility}`)}</span>
      </span>
    );
  };

  const handleToggleVisibility = async (dist: DistributionType) => {
    const newVisibility: DistributionVisibility =
      dist.visibility === "public" ? "private" : "public";

    const result = await updateDistribution(dist.id, {
      visibility: newVisibility,
    });

    if (result.success) {
      setDistributions((prev) =>
        prev.map((d) =>
          d.id === dist.id ? { ...d, visibility: newVisibility } : d,
        ),
      );
    } else {
      setError(result.message);
    }
  };

  const renderStatus = (
    value: DistributionType[keyof DistributionType],
    _row: DistributionType,
  ): JSX.Element => {
    const status = value as DistributionStatus;
    const getStatusColor = () => {
      switch (status) {
        case "ready":
          return "text-primary";
        case "pending":
        case "downloading":
        case "validating":
        case "building":
          return "text-muted-foreground";
        case "failed":
          return "text-destructive";
        default:
          return "text-muted-foreground";
      }
    };

    const isInProgress =
      status === "pending" ||
      status === "downloading" ||
      status === "validating" ||
      status === "building";

    return (
      <span class={`flex items-center gap-2 ${getStatusColor()}`}>
        <Show
          when={isInProgress}
          fallback={
            <Icon
              name={
                status === "ready"
                  ? "check-circle"
                  : status === "failed"
                    ? "x-circle"
                    : "circle"
              }
              size="sm"
            />
          }
        >
          <Spinner size="sm" />
        </Show>
        <span>{t(`distribution.status.${status}`)}</span>
      </span>
    );
  };

  const ActionsCell: Component<{ value: any; row: DistributionType }> = (
    cellProps,
  ) => {
    const canToggleVisibility = () => {
      // User can toggle if they own it OR if they're an admin
      return props.user?.id === cellProps.row.owner_id || isAdmin();
    };

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
            onSelect={() => props.onViewDistribution?.(cellProps.row.id)}
            class="gap-2"
          >
            <Icon name="eye" size="sm" />
            <span>{t("distribution.actions.viewDetails")}</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => handleEditDistribution(cellProps.row.id)}
            class="gap-2"
          >
            <Icon name="pencil" size="sm" />
            <span>{t("distribution.actions.edit")}</span>
          </DropdownMenuItem>
          <Show when={canToggleVisibility()}>
            <DropdownMenuItem
              onSelect={() => handleToggleVisibility(cellProps.row)}
              class="gap-2"
            >
              <Icon
                name={cellProps.row.visibility === "public" ? "lock" : "globe"}
                size="sm"
              />
              <span>
                {cellProps.row.visibility === "public"
                  ? t("common.visibility.makePrivate")
                  : t("common.visibility.makePublic")}
              </span>
            </DropdownMenuItem>
          </Show>
          <DropdownMenuItem
            onSelect={() => handleDeleteDistribution(cellProps.row.id)}
            class="gap-2 text-destructive focus:text-destructive"
          >
            <Icon name="trash" size="sm" />
            <span>{t("distribution.actions.delete")}</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  };

  return (
    <section class="h-full w-full relative">
      <Show
        when={props.isLoggedIn}
        fallback={
          <section class="h-full flex flex-col items-center justify-center text-center p-8">
            <h1 class="text-4xl font-bold mb-4">
              {t("distribution.welcome.title")}
            </h1>
            <p class="text-lg text-muted-foreground mb-8">
              {t("distribution.welcome.subtitle")}
            </p>
            <Card
              borderStyle="dashed"
              header={{ title: t("distribution.create.cardTitle") }}
            >
              <button onClick={handleCreateDistribution} class="cursor-pointer">
                <Icon
                  name="plus"
                  size="2xl"
                  class="text-muted-foreground hover:text-primary transition-colors"
                />
              </button>
            </Card>
          </section>
        }
      >
        <section class="h-full flex flex-col p-8 gap-6">
          <header class="flex items-center justify-between">
            <article>
              <h1 class="text-4xl font-bold">{t("distribution.title")}</h1>
              <p class="text-muted-foreground mt-2">
                {t("distribution.subtitle")}
              </p>
            </article>
            <nav class="flex items-center gap-4">
              <Show when={isAdmin()}>
                <label class="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer select-none">
                  <span>{t("distribution.filter.showOnlyMine")}</span>
                  <SummaryToggle
                    checked={showOnlyMine()}
                    onChange={setShowOnlyMine}
                  />
                </label>
              </Show>
              <button
                onClick={handleDeleteSelected}
                disabled={selectedDistributions().length === 0}
                class={`px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors ${
                  selectedDistributions().length > 0
                    ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    : "bg-muted text-muted-foreground cursor-not-allowed"
                }`}
              >
                <Icon name="trash" size="sm" />
                <span>
                  {t("common.actions.delete")} ({selectedDistributions().length}
                  )
                </span>
              </button>
              <button
                onClick={handleCreateDistribution}
                class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
              >
                <Icon name="plus" size="sm" />
                <span>{t("distribution.create.button")}</span>
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
              <Datagrid
                columns={[
                  {
                    key: "name",
                    label: t("distribution.table.columns.name"),
                    sortable: true,
                    class: "font-medium",
                    render: (value, row) => (
                      <button
                        onClick={() =>
                          props.onViewDistribution?.(
                            (row as DistributionType).id,
                          )
                        }
                        class="text-left hover:text-primary hover:underline transition-colors"
                      >
                        {value as string}
                      </button>
                    ),
                  },
                  {
                    key: "config",
                    label: t("distribution.table.columns.kernelVersion"),
                    sortable: true,
                    class: "font-mono",
                    render: (_value, row) =>
                      (row as DistributionType).config?.core?.kernel?.version ||
                      t("distribution.table.noValue"),
                  },
                  {
                    key: "status",
                    label: t("distribution.table.columns.status"),
                    sortable: true,
                    render: renderStatus,
                  },
                  {
                    key: "visibility",
                    label: t("distribution.table.columns.visibility"),
                    sortable: true,
                    render: renderVisibility,
                  },
                  {
                    key: "owner_id",
                    label: t("distribution.table.columns.owner"),
                    sortable: true,
                    class: "font-mono text-xs",
                    render: (value, _row) =>
                      (value as string) || t("distribution.table.noValue"),
                  },
                  {
                    key: "created_at",
                    label: t("distribution.table.columns.created"),
                    sortable: true,
                    class: "font-mono",
                    render: formatDate,
                  },
                  {
                    key: "id",
                    label: t("distribution.table.columns.actions"),
                    class: "text-right relative",
                    component: ActionsCell,
                  },
                ]}
                data={filteredDistributions()}
                rowKey="id"
                selectable={true}
                onSelectionChange={handleSelectionChange}
              />
            </Show>
          </section>

          <Show when={selectedDistributions().length > 0}>
            <footer class="flex justify-end pt-4">
              <button
                onClick={handleDeleteSelected}
                class="px-4 py-2 rounded-md font-medium flex items-center gap-2 transition-colors bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                <Icon name="trash" size="sm" />
                <span>
                  {t("distribution.actions.deleteSelected", {
                    count: selectedDistributions().length,
                  })}
                </span>
              </button>
            </footer>
          </Show>
        </section>
      </Show>

      <Modal
        isOpen={isModalOpen()}
        onClose={handleFormCancel}
        title={t("distribution.create.modalTitle")}
      >
        <DistributionForm
          onSubmit={handleFormSubmit}
          onCancel={handleFormCancel}
        />
      </Modal>

      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title={t("distribution.delete.title")}
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            <Show
              when={distributionsToDelete().length === 1}
              fallback={
                <>
                  {t("distribution.delete.confirmMultiple", {
                    count: distributionsToDelete().length,
                  })}
                </>
              }
            >
              {t("distribution.delete.confirmSingle", {
                name: distributionsToDelete()[0]?.name || "",
              })}
            </Show>
          </p>

          <Show when={distributionsToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto">
              {distributionsToDelete().map((dist) => (
                <li class="py-1">{dist.name}</li>
              ))}
            </ul>
          </Show>

          {/* Deletion Preview */}
          <Show when={loadingPreviews()}>
            <div class="flex items-center justify-center py-4">
              <Spinner size="md" />
            </div>
          </Show>

          <Show when={!loadingPreviews() && deletionPreviews().length > 0}>
            {(() => {
              const previews = deletionPreviews();
              const totalDownloadJobs = previews.reduce(
                (sum, p) => sum + p.download_jobs.count,
                0,
              );
              const totalArtifacts = previews.reduce(
                (sum, p) => sum + p.artifacts.count,
                0,
              );
              const totalUserSources = previews.reduce(
                (sum, p) => sum + p.user_sources.count,
                0,
              );
              const hasRelatedItems =
                totalDownloadJobs > 0 ||
                totalArtifacts > 0 ||
                totalUserSources > 0;

              return (
                <Show when={hasRelatedItems}>
                  <div class="rounded-md border border-amber-500/30 bg-amber-500/10 p-4">
                    <div class="flex items-start gap-3">
                      <Icon
                        name="warning"
                        size="md"
                        class="text-amber-500 mt-0.5"
                      />
                      <div class="flex-1">
                        <h4 class="font-medium text-amber-500 mb-2">
                          {t("distribution.delete.cascadeWarning")}
                        </h4>
                        <ul class="space-y-2 text-sm">
                          <Show when={totalDownloadJobs > 0}>
                            <li class="flex items-center gap-2">
                              <Icon
                                name="download"
                                size="sm"
                                class="text-muted-foreground"
                              />
                              <span>
                                {t("distribution.delete.downloadJobs", {
                                  count: totalDownloadJobs.toString(),
                                })}
                              </span>
                            </li>
                          </Show>
                          <Show when={totalArtifacts > 0}>
                            <li class="flex items-center gap-2">
                              <Icon
                                name="file"
                                size="sm"
                                class="text-muted-foreground"
                              />
                              <span>
                                {t("distribution.delete.artifacts", {
                                  count: totalArtifacts.toString(),
                                })}
                              </span>
                            </li>
                          </Show>
                          <Show when={totalUserSources > 0}>
                            <li class="flex items-center gap-2">
                              <Icon
                                name="database"
                                size="sm"
                                class="text-muted-foreground"
                              />
                              <span>
                                {t("distribution.delete.userSources", {
                                  count: totalUserSources.toString(),
                                })}
                              </span>
                            </li>
                            <Show
                              when={previews.some(
                                (p) =>
                                  p.user_sources.sources &&
                                  p.user_sources.sources.length > 0,
                              )}
                            >
                              <ul class="ml-6 text-xs text-muted-foreground space-y-1">
                                <For
                                  each={previews.flatMap(
                                    (p) => p.user_sources.sources || [],
                                  )}
                                >
                                  {(source) => (
                                    <li class="flex items-center gap-1">
                                      <span class="w-1 h-1 rounded-full bg-muted-foreground" />
                                      <span>{source.name}</span>
                                    </li>
                                  )}
                                </For>
                              </ul>
                            </Show>
                          </Show>
                        </ul>
                      </div>
                    </div>
                  </div>
                </Show>
              );
            })()}
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
              disabled={isDeleting() || loadingPreviews()}
              class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Show when={isDeleting()}>
                <Spinner size="sm" />
              </Show>
              <span>
                {isDeleting()
                  ? t("distribution.delete.deleting")
                  : distributionsToDelete().length > 1
                    ? t("distribution.actions.deleteCount", {
                        count: distributionsToDelete().length,
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
