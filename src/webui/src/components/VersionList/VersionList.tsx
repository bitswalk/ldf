import type { Component } from "solid-js";
import {
  createSignal,
  createEffect,
  onCleanup,
  For,
  Show,
  createMemo,
} from "solid-js";
import { Icon } from "../Icon";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationPrevious,
  PaginationNext,
  PaginationEllipsis,
} from "../Pagination";
import {
  type SourceVersion,
  type VersionSyncJob,
  type SourceType,
  listSourceVersions,
  triggerVersionSync,
  getSyncStatus,
  formatRelativeTime,
  isSyncInProgress,
} from "../../services/sourceVersions";

interface VersionListProps {
  sourceId: string;
  sourceType: SourceType;
  baseUrl?: string;
  urlTemplate?: string;
  onError?: (message: string) => void;
  onSuccess?: (message: string) => void;
  pollInterval?: number;
}

const PAGE_SIZE = 20;

export const VersionList: Component<VersionListProps> = (props) => {
  const [versions, setVersions] = createSignal<SourceVersion[]>([]);
  const [total, setTotal] = createSignal(0);
  const [currentPage, setCurrentPage] = createSignal(1);
  const [syncJob, setSyncJob] = createSignal<VersionSyncJob | null>(null);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [syncing, setSyncing] = createSignal(false);
  const [filterStable, setFilterStable] = createSignal(true);

  const pollInterval = () => props.pollInterval ?? 2000;

  const totalPages = createMemo(() => Math.ceil(total() / PAGE_SIZE));

  const filteredVersions = createMemo(() => {
    if (!filterStable()) return versions();
    return versions().filter((v) => v.is_stable);
  });

  const fetchVersions = async (page: number = 1) => {
    setLoading(true);
    const offset = (page - 1) * PAGE_SIZE;
    const result = await listSourceVersions(
      props.sourceId,
      props.sourceType,
      PAGE_SIZE,
      offset,
      filterStable(),
    );
    if (result.success) {
      setVersions(result.versions);
      setTotal(result.total);
      if (result.syncJob) {
        setSyncJob(result.syncJob);
      }
      setError(null);
    } else {
      if (result.error !== "not_found") {
        setError(result.message);
      }
    }
    setLoading(false);
  };

  const fetchSyncStatus = async () => {
    const result = await getSyncStatus(props.sourceId, props.sourceType);
    if (result.success && result.job) {
      setSyncJob(result.job);
      // If sync just completed, refresh versions
      if (result.job.status === "completed" || result.job.status === "failed") {
        fetchVersions(currentPage());
      }
    }
  };

  // Initial fetch
  createEffect(() => {
    fetchVersions(1);
    setCurrentPage(1);
  });

  // Polling for sync status
  createEffect(() => {
    const interval = setInterval(() => {
      if (isSyncInProgress(syncJob())) {
        fetchSyncStatus();
      }
    }, pollInterval());

    onCleanup(() => clearInterval(interval));
  });

  // Refetch when filter changes
  createEffect(() => {
    const _ = filterStable();
    fetchVersions(1);
    setCurrentPage(1);
  });

  const handlePageChange = (page: number) => {
    if (page < 1 || page > totalPages()) return;
    setCurrentPage(page);
    fetchVersions(page);
  };

  const handleSync = async () => {
    setSyncing(true);
    const result = await triggerVersionSync(props.sourceId, props.sourceType);
    if (result.success) {
      props.onSuccess?.("Version sync started");
      // Start polling for updates
      fetchSyncStatus();
    } else {
      props.onError?.(result.message);
    }
    setSyncing(false);
  };

  const handleCopyUrl = (version: SourceVersion) => {
    const url = version.download_url || buildVersionUrl(version.version);
    navigator.clipboard.writeText(url);
    props.onSuccess?.("URL copied to clipboard");
  };

  const buildVersionUrl = (version: string): string => {
    if (!props.baseUrl) return "";
    const template = props.urlTemplate || "{base_url}/{version}";
    return template
      .replace("{base_url}", props.baseUrl)
      .replace("{version}", version)
      .replace("{tag}", `v${version}`);
  };

  const getSyncStatusIcon = (job: VersionSyncJob | null): string => {
    if (!job) return "circle";
    switch (job.status) {
      case "pending":
        return "clock";
      case "running":
        return "spinner-gap";
      case "completed":
        return "check-circle";
      case "failed":
        return "x-circle";
      default:
        return "circle";
    }
  };

  const getSyncStatusColor = (job: VersionSyncJob | null): string => {
    if (!job) return "text-muted-foreground";
    switch (job.status) {
      case "pending":
        return "text-yellow-500";
      case "running":
        return "text-primary";
      case "completed":
        return "text-green-500";
      case "failed":
        return "text-red-500";
      default:
        return "text-muted-foreground";
    }
  };

  const formatDate = (dateString: string | undefined): string => {
    if (!dateString) return "-";
    const date = new Date(dateString);
    return date.toLocaleDateString();
  };

  // Generate visible page numbers with ellipsis logic
  const visiblePages = createMemo(() => {
    const total = totalPages();
    const current = currentPage();
    const pages: (number | "ellipsis")[] = [];

    if (total <= 7) {
      // Show all pages if 7 or fewer
      for (let i = 1; i <= total; i++) {
        pages.push(i);
      }
    } else {
      // Always show first page
      pages.push(1);

      if (current > 3) {
        pages.push("ellipsis");
      }

      // Show pages around current
      const start = Math.max(2, current - 1);
      const end = Math.min(total - 1, current + 1);

      for (let i = start; i <= end; i++) {
        pages.push(i);
      }

      if (current < total - 2) {
        pages.push("ellipsis");
      }

      // Always show last page
      pages.push(total);
    }

    return pages;
  });

  return (
    <div class="space-y-4">
      {/* Header with Sync button */}
      <div class="flex items-center justify-between flex-wrap gap-4">
        <div class="flex items-center gap-2">
          <Icon name="git-branch" size="lg" class="text-primary" />
          <h3 class="text-lg font-semibold">Available Versions</h3>
          <Show when={total() > 0}>
            <span class="text-sm text-muted-foreground">
              ({total()} {filterStable() ? "stable" : "total"})
            </span>
          </Show>
        </div>
        <div class="flex items-center gap-2">
          {/* Filter toggle */}
          <button
            onClick={() => setFilterStable(!filterStable())}
            class={`flex items-center gap-1 px-3 py-1.5 text-sm rounded-md transition-colors ${
              filterStable()
                ? "bg-green-500/20 text-green-500"
                : "bg-muted text-muted-foreground hover:bg-muted/80"
            }`}
          >
            <Icon name={filterStable() ? "check" : "circle"} size="sm" />
            <span>Stable only</span>
          </button>

          {/* Sync button */}
          <button
            onClick={handleSync}
            disabled={syncing() || isSyncInProgress(syncJob())}
            class="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <Show
              when={!syncing() && !isSyncInProgress(syncJob())}
              fallback={
                <Icon name="spinner-gap" size="sm" class="animate-spin" />
              }
            >
              <Icon name="arrows-clockwise" size="sm" />
            </Show>
            <span>
              {syncing()
                ? "Starting..."
                : isSyncInProgress(syncJob())
                  ? "Syncing..."
                  : "Sync Versions"}
            </span>
          </button>
        </div>
      </div>

      {/* Sync status */}
      <Show when={syncJob()}>
        <div
          class={`flex items-center gap-2 text-sm p-3 rounded-md ${
            syncJob()?.status === "failed"
              ? "bg-red-500/10"
              : syncJob()?.status === "completed"
                ? "bg-green-500/10"
                : "bg-muted"
          }`}
        >
          <Icon
            name={getSyncStatusIcon(syncJob())}
            size="md"
            class={`${getSyncStatusColor(syncJob())} ${
              isSyncInProgress(syncJob()) ? "animate-spin" : ""
            }`}
          />
          <div class="flex-1">
            <Show when={isSyncInProgress(syncJob())}>
              <span>Discovering versions from upstream...</span>
              <Show when={(syncJob()?.versions_found ?? 0) > 0}>
                <span class="ml-2 text-muted-foreground">
                  (found {syncJob()?.versions_found} so far)
                </span>
              </Show>
            </Show>
            <Show when={syncJob()?.status === "completed"}>
              <span class="text-green-500">
                Sync completed: found {syncJob()?.versions_found} versions
                <Show when={(syncJob()?.versions_new ?? 0) > 0}>
                  <span> ({syncJob()?.versions_new} new)</span>
                </Show>
              </span>
              <Show when={syncJob()?.completed_at}>
                <span class="ml-2 text-muted-foreground">
                  {formatRelativeTime(syncJob()!.completed_at!)}
                </span>
              </Show>
            </Show>
            <Show when={syncJob()?.status === "failed"}>
              <span class="text-red-500">
                Sync failed: {syncJob()?.error_message || "Unknown error"}
              </span>
            </Show>
          </div>
        </div>
      </Show>

      {/* Loading state */}
      <Show when={loading()}>
        <div class="flex items-center justify-center py-8">
          <Icon
            name="spinner-gap"
            size="xl"
            class="animate-spin text-primary"
          />
        </div>
      </Show>

      {/* Error state */}
      <Show when={error()}>
        <div class="bg-red-500/10 border border-red-500/20 rounded-md p-4">
          <div class="flex items-center gap-2 text-red-500">
            <Icon name="warning-circle" size="md" />
            <span>{error()}</span>
          </div>
        </div>
      </Show>

      {/* Empty state */}
      <Show when={!loading() && versions().length === 0 && !error()}>
        <div class="text-center py-8 text-muted-foreground">
          <Icon name="git-branch" size="2xl" class="mx-auto mb-2 opacity-50" />
          <p>No versions discovered yet</p>
          <p class="text-sm">
            Click "Sync Versions" to fetch available versions from upstream
          </p>
        </div>
      </Show>

      {/* Versions table */}
      <Show when={!loading() && filteredVersions().length > 0}>
        <div class="border border-border rounded-lg overflow-hidden">
          <table class="w-full">
            <thead class="bg-muted/50">
              <tr>
                <th class="text-left px-4 py-3 text-sm font-medium text-muted-foreground">
                  Version
                </th>
                <th class="text-left px-4 py-3 text-sm font-medium text-muted-foreground">
                  Release Date
                </th>
                <th class="text-left px-4 py-3 text-sm font-medium text-muted-foreground">
                  Type
                </th>
                <th class="text-right px-4 py-3 text-sm font-medium text-muted-foreground">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <For each={filteredVersions()}>
                {(version) => (
                  <tr class="hover:bg-muted/30 transition-colors">
                    <td class="px-4 py-3">
                      <div class="flex items-center gap-2">
                        <span class="font-mono font-medium">
                          {version.version}
                        </span>
                      </div>
                    </td>
                    <td class="px-4 py-3 text-sm text-muted-foreground">
                      {formatDate(version.release_date)}
                    </td>
                    <td class="px-4 py-3">
                      <span
                        class={`text-xs px-2 py-0.5 rounded-full ${
                          version.is_stable
                            ? "bg-green-500/20 text-green-500"
                            : "bg-yellow-500/20 text-yellow-500"
                        }`}
                      >
                        {version.is_stable ? "Stable" : "Pre-release"}
                      </span>
                    </td>
                    <td class="px-4 py-3 text-right">
                      <button
                        onClick={() => handleCopyUrl(version)}
                        class="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors"
                        title="Copy download URL"
                      >
                        <Icon name="copy" size="md" />
                      </button>
                      <Show when={version.download_url}>
                        <a
                          href={version.download_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          class="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors inline-block"
                          title="Open download URL"
                        >
                          <Icon name="arrow-square-out" size="md" />
                        </a>
                      </Show>
                    </td>
                  </tr>
                )}
              </For>
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <Show when={totalPages() > 1}>
          <div class="flex items-center justify-between pt-2">
            <p class="text-sm text-muted-foreground">
              Showing {(currentPage() - 1) * PAGE_SIZE + 1} to{" "}
              {Math.min(currentPage() * PAGE_SIZE, total())} of {total()}{" "}
              versions
            </p>
            <Pagination>
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    onClick={() => handlePageChange(currentPage() - 1)}
                    class={
                      currentPage() === 1
                        ? "pointer-events-none opacity-50"
                        : "cursor-pointer"
                    }
                  />
                </PaginationItem>

                <For each={visiblePages()}>
                  {(page) => (
                    <PaginationItem>
                      <Show
                        when={page !== "ellipsis"}
                        fallback={<PaginationEllipsis />}
                      >
                        <PaginationLink
                          onClick={() => handlePageChange(page as number)}
                          isActive={currentPage() === page}
                          class="cursor-pointer"
                        >
                          {page}
                        </PaginationLink>
                      </Show>
                    </PaginationItem>
                  )}
                </For>

                <PaginationItem>
                  <PaginationNext
                    onClick={() => handlePageChange(currentPage() + 1)}
                    class={
                      currentPage() === totalPages()
                        ? "pointer-events-none opacity-50"
                        : "cursor-pointer"
                    }
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </div>
        </Show>
      </Show>
    </div>
  );
};
