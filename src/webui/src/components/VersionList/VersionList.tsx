import type { Component, JSX } from "solid-js";
import {
  createSignal,
  createEffect,
  createMemo,
  onCleanup,
  Show,
} from "solid-js";
import { Icon } from "../Icon";
import { Datagrid } from "../Datagrid";
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
  type VersionType,
  listSourceVersions,
  triggerVersionSync,
  getSyncStatus,
  clearSourceVersions,
  getSourceVersionTypes,
  formatRelativeTime,
  isSyncInProgress,
} from "../../services/sourceVersions";
import {
  parseVersionFilter,
  matchesFilter,
  type FilterPattern,
} from "../../utils/globFilter";

interface VersionListProps {
  sourceId: string;
  baseUrl?: string;
  urlTemplate?: string;
  versionFilter?: string;
  onError?: (message: string) => void;
  onSuccess?: (message: string) => void;
  pollInterval?: number;
}

const PAGE_SIZE = 20;

// Format version type label for display
function formatVersionTypeLabel(type: string): string {
  if (type === "linux-next") return "Linux-next";
  return type.charAt(0).toUpperCase() + type.slice(1);
}

export const VersionList: Component<VersionListProps> = (props) => {
  const [versions, setVersions] = createSignal<SourceVersion[]>([]);
  const [total, setTotal] = createSignal(0);
  const [currentPage, setCurrentPage] = createSignal(1);
  const [syncJob, setSyncJob] = createSignal<VersionSyncJob | null>(null);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [syncing, setSyncing] = createSignal(false);
  const [clearing, setClearing] = createSignal(false);
  const [versionTypeFilter, setVersionTypeFilter] = createSignal<string>("all");
  const [availableVersionTypes, setAvailableVersionTypes] = createSignal<
    string[]
  >([]);
  const [dropdownOpen, setDropdownOpen] = createSignal(false);
  const [showFilteredOut, setShowFilteredOut] = createSignal(false);

  // Build version type options from available types
  const versionTypeOptions = createMemo(() => {
    const types = availableVersionTypes();
    const options: { value: string; label: string }[] = [
      { value: "all", label: "All versions" },
    ];
    for (const type of types) {
      options.push({ value: type, label: formatVersionTypeLabel(type) });
    }
    return options;
  });

  const pollInterval = () => props.pollInterval ?? 2000;

  // Parse filter patterns
  const filterPatterns = createMemo<FilterPattern[]>(() => {
    if (!props.versionFilter) return [];
    return parseVersionFilter(props.versionFilter);
  });

  const hasVersionFilter = () => filterPatterns().length > 0;

  // Apply version filter to versions
  const filteredVersions = createMemo(() => {
    const allVersions = versions();
    const patterns = filterPatterns();

    if (patterns.length === 0 || showFilteredOut()) {
      return allVersions;
    }

    return allVersions.filter((v) => matchesFilter(v.version, patterns));
  });

  // Count filtered out versions
  const filteredOutCount = createMemo(() => {
    const allVersions = versions();
    const patterns = filterPatterns();

    if (patterns.length === 0) return 0;

    return allVersions.filter((v) => !matchesFilter(v.version, patterns))
      .length;
  });

  // Check if a version is filtered out
  const isFilteredOut = (version: string): boolean => {
    const patterns = filterPatterns();
    if (patterns.length === 0) return false;
    return !matchesFilter(version, patterns);
  };

  const totalPages = () => Math.ceil(total() / PAGE_SIZE);

  const fetchVersions = async (page: number = 1) => {
    setLoading(true);
    const offset = (page - 1) * PAGE_SIZE;
    const filter = versionTypeFilter();
    const result = await listSourceVersions(
      props.sourceId,
      PAGE_SIZE,
      offset,
      filter === "all" ? undefined : filter,
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
    const result = await getSyncStatus(props.sourceId);
    if (result.success && result.job) {
      setSyncJob(result.job);
      if (result.job.status === "completed" || result.job.status === "failed") {
        fetchVersions(currentPage());
        // Refresh version types after sync completes
        fetchVersionTypes();
      }
    }
  };

  const fetchVersionTypes = async () => {
    const result = await getSourceVersionTypes(props.sourceId);
    if (result.success) {
      setAvailableVersionTypes(result.types);
    }
  };

  // Initial fetch
  createEffect(() => {
    fetchVersions(1);
    fetchVersionTypes();
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
    const _ = versionTypeFilter();
    fetchVersions(1);
    setCurrentPage(1);
  });

  // Close dropdown when clicking outside
  createEffect(() => {
    if (dropdownOpen()) {
      const handleClickOutside = (e: MouseEvent) => {
        const target = e.target as HTMLElement;
        if (!target.closest("[data-version-filter-dropdown]")) {
          setDropdownOpen(false);
        }
      };
      document.addEventListener("click", handleClickOutside);
      onCleanup(() =>
        document.removeEventListener("click", handleClickOutside),
      );
    }
  });

  const handlePageChange = (page: number) => {
    if (page < 1 || page > totalPages()) return;
    setCurrentPage(page);
    fetchVersions(page);
  };

  const handleSync = async () => {
    setSyncing(true);
    const result = await triggerVersionSync(props.sourceId);
    if (result.success) {
      props.onSuccess?.("Version sync started");
      fetchSyncStatus();
    } else {
      props.onError?.(result.message);
    }
    setSyncing(false);
  };

  const handleClear = async () => {
    setClearing(true);
    const result = await clearSourceVersions(props.sourceId);
    if (result.success) {
      props.onSuccess?.("Version cache cleared");
      setVersions([]);
      setTotal(0);
      setSyncJob(null);
      setCurrentPage(1);
    } else {
      props.onError?.(result.message);
    }
    setClearing(false);
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

  const getVersionTypeColor = (
    type: VersionType | undefined | string,
  ): string => {
    switch (type) {
      case "mainline":
        return "bg-blue-500/20 text-blue-500";
      case "longterm":
        return "bg-purple-500/20 text-purple-500";
      case "linux-next":
        return "bg-orange-500/20 text-orange-500";
      case "stable":
        return "bg-green-500/20 text-green-500";
      default:
        // Handle empty string, undefined, or any unknown type as stable
        return "bg-green-500/20 text-green-500";
    }
  };

  const getVersionTypeLabel = (
    type: VersionType | undefined | string,
  ): string => {
    if (!type || type === "") return "Stable";
    if (type === "linux-next") return "Linux-next";
    return type.charAt(0).toUpperCase() + type.slice(1);
  };

  const renderVersionType = (
    versionType: VersionType | undefined | string,
  ): JSX.Element => {
    return (
      <span
        class={`text-xs px-2 py-0.5 rounded-full ${getVersionTypeColor(versionType)}`}
      >
        {getVersionTypeLabel(versionType)}
      </span>
    );
  };

  const ActionsCell: Component<{ value: any; row: SourceVersion }> = (
    cellProps,
  ) => {
    return (
      <div class="flex justify-end gap-1">
        <button
          onClick={() => handleCopyUrl(cellProps.row)}
          class="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors"
          title="Copy download URL"
        >
          <Icon name="copy" size="md" />
        </button>
        <Show when={cellProps.row.download_url}>
          <a
            href={cellProps.row.download_url}
            target="_blank"
            rel="noopener noreferrer"
            class="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors inline-block"
            title="Open download URL"
          >
            <Icon name="arrow-square-out" size="md" />
          </a>
        </Show>
      </div>
    );
  };

  // Generate visible page numbers with ellipsis logic
  const visiblePages = () => {
    const total = totalPages();
    const current = currentPage();
    const pages: (number | "ellipsis")[] = [];

    if (total <= 7) {
      for (let i = 1; i <= total; i++) {
        pages.push(i);
      }
    } else {
      pages.push(1);

      if (current > 3) {
        pages.push("ellipsis");
      }

      const start = Math.max(2, current - 1);
      const end = Math.min(total - 1, current + 1);

      for (let i = start; i <= end; i++) {
        pages.push(i);
      }

      if (current < total - 2) {
        pages.push("ellipsis");
      }

      pages.push(total);
    }

    return pages;
  };

  return (
    <div class="space-y-4">
      {/* Header with Sync button */}
      <div class="flex items-center justify-between flex-wrap gap-4">
        <div class="flex items-center gap-2">
          <Icon name="git-branch" size="lg" class="text-primary" />
          <h3 class="text-lg font-semibold">Available Versions</h3>
          <Show when={total() > 0}>
            <span class="text-sm text-muted-foreground">({total()})</span>
          </Show>
        </div>
        <div class="flex items-center gap-2">
          {/* Version filter indicator */}
          <Show when={hasVersionFilter()}>
            <div class="flex items-center gap-2">
              <span class="text-xs text-muted-foreground">
                Filter:{" "}
                <code class="px-1 py-0.5 bg-muted rounded font-mono">
                  {props.versionFilter}
                </code>
              </span>
              <Show when={filteredOutCount() > 0}>
                <button
                  onClick={() => setShowFilteredOut(!showFilteredOut())}
                  class={`flex items-center gap-1 px-2 py-1 text-xs rounded-md transition-colors ${
                    showFilteredOut()
                      ? "bg-orange-500/20 text-orange-500"
                      : "bg-muted text-muted-foreground hover:bg-muted/80"
                  }`}
                  title={
                    showFilteredOut()
                      ? "Hide filtered versions"
                      : "Show filtered versions"
                  }
                >
                  <Icon
                    name={showFilteredOut() ? "eye" : "eye-slash"}
                    size="xs"
                  />
                  <span>{filteredOutCount()} hidden</span>
                </button>
              </Show>
            </div>
          </Show>

          {/* Version type filter dropdown - only show if there are multiple types */}
          <Show when={versionTypeOptions().length > 1}>
            <div class="relative" data-version-filter-dropdown>
              <button
                onClick={() => setDropdownOpen(!dropdownOpen())}
                class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md bg-muted hover:bg-muted/80 transition-colors"
              >
                <Icon name="funnel" size="sm" />
                <span>
                  {versionTypeOptions().find(
                    (o) => o.value === versionTypeFilter(),
                  )?.label ?? "All versions"}
                </span>
                <Icon name="caret-down" size="sm" />
              </button>
              <Show when={dropdownOpen()}>
                <div class="absolute right-0 top-full mt-1 z-50 min-w-[160px] bg-popover border border-border rounded-md shadow-lg">
                  {versionTypeOptions().map((option) => (
                    <button
                      onClick={() => {
                        setVersionTypeFilter(option.value);
                        setDropdownOpen(false);
                      }}
                      class={`w-full text-left px-3 py-2 text-sm hover:bg-muted transition-colors first:rounded-t-md last:rounded-b-md ${
                        versionTypeFilter() === option.value
                          ? "bg-primary/10 text-primary"
                          : ""
                      }`}
                    >
                      {option.label}
                    </button>
                  ))}
                </div>
              </Show>
            </div>
          </Show>

          {/* Clear button */}
          <Show when={total() > 0}>
            <button
              onClick={handleClear}
              disabled={clearing() || isSyncInProgress(syncJob())}
              class="flex items-center gap-2 px-3 py-2 border border-border text-muted-foreground rounded-md hover:bg-muted hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              title="Clear version cache"
            >
              <Show
                when={!clearing()}
                fallback={
                  <Icon name="spinner-gap" size="sm" class="animate-spin" />
                }
              >
                <Icon name="trash" size="sm" />
              </Show>
              <span>{clearing() ? "Clearing..." : "Clear"}</span>
            </button>
          </Show>

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

      {/* Versions table using Datagrid */}
      <Show when={!loading() && filteredVersions().length > 0}>
        <div class="border border-border rounded-lg overflow-hidden">
          <Datagrid
            columns={[
              {
                key: "version",
                label: "Version",
                sortable: true,
                class: "font-mono font-medium",
                render: (value) => {
                  const version = value as string;
                  const filtered = isFilteredOut(version);
                  return (
                    <span
                      class={
                        filtered ? "line-through text-muted-foreground" : ""
                      }
                    >
                      {version}
                      {filtered && (
                        <span class="ml-2 text-xs text-orange-500">
                          (filtered)
                        </span>
                      )}
                    </span>
                  );
                },
              },
              {
                key: "release_date",
                label: "Release Date",
                sortable: true,
                class: "text-sm text-muted-foreground",
                render: (value) => formatDate(value as string | undefined),
              },
              {
                key: "version_type",
                label: "Type",
                sortable: true,
                render: (value) =>
                  renderVersionType(value as VersionType | undefined | string),
              },
              {
                key: "id",
                label: "Actions",
                class: "text-right",
                component: ActionsCell,
              },
            ]}
            data={filteredVersions()}
            rowKey="id"
          />
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

                {visiblePages().map((page) => (
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
                ))}

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
