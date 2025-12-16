import type { Component, JSX } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
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
  type Distribution as DistributionType,
  type DistributionStatus,
  type DistributionVisibility,
  type DistributionConfig,
} from "../../services/distributionService";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface DistributionProps {
  isLoggedIn?: boolean;
  user?: UserInfo | null;
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

  const handleEditDistribution = (id: number) => {
    console.log("Edit distribution:", id);
  };

  const openDeleteModal = (dists: DistributionType[]) => {
    setDistributionsToDelete(dists);
    setDeleteModalOpen(true);
  };

  const handleDeleteDistribution = (id: number) => {
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
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString();
  };

  const isAdmin = () => props.user?.role === "root";

  const filteredDistributions = () => {
    if (showOnlyMine() && props.user?.id) {
      return distributions().filter((d) => d.owner_id === props.user?.id);
    }
    return distributions();
  };

  const renderVisibility = (
    visibility: DistributionVisibility,
  ): JSX.Element => {
    const isPublic = visibility === "public";
    return (
      <span
        class={`flex items-center gap-2 ${isPublic ? "text-primary" : "text-muted-foreground"}`}
      >
        <Icon name={isPublic ? "globe" : "lock"} size="sm" />
        <span class="capitalize">{visibility}</span>
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

  const renderStatus = (status: DistributionStatus): JSX.Element => {
    const getStatusColor = () => {
      switch (status) {
        case "ready":
          return "text-primary";
        case "pending":
        case "downloading":
        case "validating":
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
      status === "validating";

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
        <span class="capitalize">{status}</span>
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
            onSelect={() => handleEditDistribution(cellProps.row.id)}
            class="gap-2"
          >
            <Icon name="pencil" size="sm" />
            <span>Edit</span>
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
                Make{" "}
                {cellProps.row.visibility === "public" ? "Private" : "Public"}
              </span>
            </DropdownMenuItem>
          </Show>
          <DropdownMenuItem
            onSelect={() => handleDeleteDistribution(cellProps.row.id)}
            class="gap-2 text-destructive focus:text-destructive"
          >
            <Icon name="trash" size="sm" />
            <span>Delete</span>
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
              Welcome to Linux Distribution Factory
            </h1>
            <p class="text-lg text-muted-foreground mb-8">
              No distributions configured yet. Get started by creating your
              first custom Linux distribution.
            </p>
            <Card
              borderStyle="dashed"
              header={{ title: "Create new distribution" }}
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
              <h1 class="text-4xl font-bold">Distributions</h1>
              <p class="text-muted-foreground mt-2">
                Manage your custom Linux distributions
              </p>
            </article>
            <nav class="flex items-center gap-4">
              <Show when={isAdmin()}>
                <label class="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer select-none">
                  <span>Show only mine</span>
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
                <span>Delete ({selectedDistributions().length})</span>
              </button>
              <button
                onClick={handleCreateDistribution}
                class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium flex items-center gap-2"
              >
                <Icon name="plus" size="sm" />
                <span>New Distribution</span>
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
                    label: "Name",
                    sortable: true,
                    class: "font-medium",
                  },
                  {
                    key: "version",
                    label: "Kernel Version",
                    sortable: true,
                    class: "font-mono",
                  },
                  {
                    key: "status",
                    label: "Status",
                    sortable: true,
                    render: renderStatus,
                  },
                  {
                    key: "visibility",
                    label: "Visibility",
                    sortable: true,
                    render: renderVisibility,
                  },
                  {
                    key: "owner_id",
                    label: "Owner",
                    sortable: true,
                    class: "font-mono text-xs",
                    render: (ownerId: string) => ownerId || "—",
                  },
                  {
                    key: "created_at",
                    label: "Created",
                    sortable: true,
                    class: "font-mono",
                    render: formatDate,
                  },
                  {
                    key: "id",
                    label: "Actions",
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
                <span>Delete Selected ({selectedDistributions().length})</span>
              </button>
            </footer>
          </Show>
        </section>
      </Show>

      <Modal
        isOpen={isModalOpen()}
        onClose={handleFormCancel}
        title="Create New Distribution"
      >
        <DistributionForm
          onSubmit={handleFormSubmit}
          onCancel={handleFormCancel}
        />
      </Modal>

      <Modal
        isOpen={deleteModalOpen()}
        onClose={cancelDelete}
        title="Confirm Deletion"
      >
        <section class="flex flex-col gap-6">
          <p class="text-muted-foreground">
            Are you sure you want to delete{" "}
            <Show
              when={distributionsToDelete().length === 1}
              fallback={
                <span class="text-foreground font-medium">
                  {distributionsToDelete().length} distributions
                </span>
              }
            >
              <span class="text-foreground font-medium">
                "{distributionsToDelete()[0]?.name}"
              </span>
            </Show>
            ? This action cannot be undone.
          </p>

          <Show when={distributionsToDelete().length > 1}>
            <ul class="text-sm text-muted-foreground bg-muted/50 rounded-md p-3 max-h-32 overflow-y-auto">
              {distributionsToDelete().map((dist) => (
                <li class="py-1">{dist.name}</li>
              ))}
            </ul>
          </Show>

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
              <span>
                {isDeleting()
                  ? "Deleting..."
                  : `Delete${distributionsToDelete().length > 1 ? ` (${distributionsToDelete().length})` : ""}`}
              </span>
            </button>
          </nav>
        </section>
      </Modal>
    </section>
  );
};
