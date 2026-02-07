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
import { t } from "../../services/i18n";
import {
  type DownloadJob,
  type DownloadJobStatus,
  listDownloads,
  cancelDownload,
  retryDownload,
  startDownloads,
  flushDownloads,
  getStatusDisplayText,
  getStatusColor,
  isJobActive,
  canRetryJob,
  formatBytes,
} from "../../services/downloads";

export interface DownloadActions {
  start: () => void;
  flush: () => void;
  starting: () => boolean;
  flushing: () => boolean;
  hasActiveJobs: () => boolean;
  hasJobs: () => boolean;
}

interface DownloadStatusProps {
  distributionId: string;
  onError?: (message: string) => void;
  onSuccess?: (message: string) => void;
  onActions?: (actions: DownloadActions) => void;
  pollInterval?: number;
}

export const DownloadStatus: Component<DownloadStatusProps> = (props) => {
  const [jobs, setJobs] = createSignal<DownloadJob[]>([]);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [starting, setStarting] = createSignal(false);
  const [actionInProgress, setActionInProgress] = createSignal<string | null>(
    null,
  );
  const [flushing, setFlushing] = createSignal(false);

  const pollInterval = () => props.pollInterval ?? 3000;

  const hasActiveJobs = createMemo(() =>
    jobs().some((job) => isJobActive(job.status)),
  );

  const fetchJobs = async () => {
    const result = await listDownloads(props.distributionId);
    if (result.success) {
      setJobs(result.jobs);
      setError(null);
    } else {
      if (result.error !== "not_found") {
        setError(result.message);
      }
    }
    setLoading(false);
  };

  // Initial fetch and polling
  createEffect(() => {
    fetchJobs();

    const interval = setInterval(() => {
      if (hasActiveJobs() || jobs().length === 0) {
        fetchJobs();
      }
    }, pollInterval());

    onCleanup(() => clearInterval(interval));
  });

  const handleStartDownloads = async () => {
    setStarting(true);
    const result = await startDownloads(props.distributionId);
    if (result.success) {
      setJobs(result.jobs);
      props.onSuccess?.(t("common.downloads.started"));
    } else {
      props.onError?.(result.message);
    }
    setStarting(false);
  };

  const handleCancel = async (jobId: string) => {
    setActionInProgress(jobId);
    const result = await cancelDownload(jobId);
    if (result.success) {
      await fetchJobs();
      props.onSuccess?.(t("common.downloads.cancelled"));
    } else {
      props.onError?.(result.message);
    }
    setActionInProgress(null);
  };

  const handleRetry = async (jobId: string) => {
    setActionInProgress(jobId);
    const result = await retryDownload(jobId);
    if (result.success) {
      await fetchJobs();
      props.onSuccess?.(t("common.downloads.retried"));
    } else {
      props.onError?.(result.message);
    }
    setActionInProgress(null);
  };

  const handleFlush = async () => {
    setFlushing(true);
    const result = await flushDownloads(props.distributionId);
    if (result.success) {
      setJobs([]);
      props.onSuccess?.(t("common.downloads.historyCleared"));
    } else {
      props.onError?.(result.message);
    }
    setFlushing(false);
  };

  // Expose actions to parent for Card header rendering
  props.onActions?.({
    start: handleStartDownloads,
    flush: handleFlush,
    starting,
    flushing,
    hasActiveJobs,
    hasJobs: () => jobs().length > 0,
  });

  const getStatusIcon = (status: DownloadJobStatus): string => {
    switch (status) {
      case "pending":
        return "clock";
      case "verifying":
        return "magnifying-glass";
      case "downloading":
        return "cloud-arrow-down";
      case "completed":
        return "check-circle";
      case "failed":
        return "x-circle";
      case "cancelled":
        return "warning-circle";
      default:
        return "circle";
    }
  };

  const getStatusColorClass = (status: DownloadJobStatus): string => {
    const color = getStatusColor(status);
    switch (color) {
      case "primary":
        return "text-primary";
      case "success":
        return "text-green-500";
      case "danger":
        return "text-red-500";
      case "warning":
        return "text-yellow-500";
      default:
        return "text-muted-foreground";
    }
  };

  const getProgressPercentage = (job: DownloadJob): number => {
    if (job.total_bytes === 0) return 0;
    return Math.round((job.progress_bytes / job.total_bytes) * 100);
  };

  return (
    <div class="space-y-3">
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
      <Show when={!loading() && jobs().length === 0 && !error()}>
        <div class="text-center py-8 text-muted-foreground">
          <Icon
            name="cloud-arrow-down"
            size="2xl"
            class="mx-auto mb-2 opacity-50"
          />
          <p>{t("common.downloads.noDownloads")}</p>
          <p class="text-sm">{t("common.downloads.emptyHint")}</p>
        </div>
      </Show>

      {/* Jobs list */}
      <Show when={!loading() && jobs().length > 0}>
        <div class="space-y-3">
          <For each={jobs()}>
            {(job) => (
              <div class="border border-border rounded-lg p-4">
                <div class="flex items-start justify-between gap-4">
                  {/* Job info */}
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-1">
                      <Icon
                        name={getStatusIcon(job.status)}
                        size="md"
                        class={`${getStatusColorClass(job.status)} ${
                          isJobActive(job.status) ? "animate-pulse" : ""
                        }`}
                      />
                      <span class="font-medium truncate">
                        {job.component_name || job.component_id}
                      </span>
                      <span
                        class={`text-xs px-2 py-0.5 rounded-full ${
                          job.status === "completed"
                            ? "bg-green-500/20 text-green-500"
                            : job.status === "failed"
                              ? "bg-red-500/20 text-red-500"
                              : job.status === "cancelled"
                                ? "bg-yellow-500/20 text-yellow-500"
                                : isJobActive(job.status)
                                  ? "bg-primary/20 text-primary"
                                  : "bg-muted text-muted-foreground"
                        }`}
                      >
                        {getStatusDisplayText(job.status)}
                      </span>
                    </div>

                    {/* Version and URL */}
                    <div class="text-sm text-muted-foreground mb-2">
                      <span>Version: {job.version}</span>
                      <Show when={job.resolved_url}>
                        <span class="mx-2">|</span>
                        <span class="truncate" title={job.resolved_url}>
                          {job.resolved_url.length > 50
                            ? job.resolved_url.substring(0, 50) + "..."
                            : job.resolved_url}
                        </span>
                      </Show>
                    </div>

                    {/* Progress bar for active downloads */}
                    <Show
                      when={job.status === "downloading" && job.total_bytes > 0}
                    >
                      <div class="space-y-1">
                        <div class="h-2 bg-muted rounded-full overflow-hidden">
                          <div
                            class="h-full bg-primary transition-all duration-300"
                            style={{ width: `${getProgressPercentage(job)}%` }}
                          />
                        </div>
                        <div class="flex justify-between text-xs text-muted-foreground">
                          <span>
                            {formatBytes(job.progress_bytes)} /{" "}
                            {formatBytes(job.total_bytes)}
                          </span>
                          <span>{getProgressPercentage(job)}%</span>
                        </div>
                      </div>
                    </Show>

                    {/* Indeterminate progress for verifying/pending */}
                    <Show
                      when={
                        job.status === "verifying" ||
                        (job.status === "downloading" && job.total_bytes === 0)
                      }
                    >
                      <div class="h-2 bg-muted rounded-full overflow-hidden">
                        <div class="h-full bg-primary w-1/3 animate-pulse" />
                      </div>
                    </Show>

                    {/* Error message */}
                    <Show when={job.error_message}>
                      <div class="mt-2 text-sm text-red-500 flex items-start gap-1">
                        <Icon
                          name="warning"
                          size="sm"
                          class="mt-0.5 flex-shrink-0"
                        />
                        <span>{job.error_message}</span>
                      </div>
                    </Show>

                    {/* Completed info */}
                    <Show
                      when={job.status === "completed" && job.artifact_path}
                    >
                      <div class="mt-2 text-sm text-green-500 flex items-center gap-1">
                        <Icon name="check" size="sm" />
                        <span class="truncate" title={job.artifact_path}>
                          Saved: {job.artifact_path}
                        </span>
                      </div>
                    </Show>

                    {/* Retry count */}
                    <Show when={job.retry_count > 0}>
                      <div class="mt-1 text-xs text-muted-foreground">
                        Retry {job.retry_count}/{job.max_retries}
                      </div>
                    </Show>
                  </div>

                  {/* Actions */}
                  <div class="flex items-center gap-2">
                    <Show when={isJobActive(job.status)}>
                      <button
                        onClick={() => handleCancel(job.id)}
                        disabled={actionInProgress() === job.id}
                        class="p-2 text-muted-foreground hover:text-red-500 hover:bg-red-500/10 rounded-md transition-colors disabled:opacity-50"
                        title="Cancel download"
                      >
                        <Show
                          when={actionInProgress() !== job.id}
                          fallback={
                            <Icon
                              name="spinner-gap"
                              size="md"
                              class="animate-spin"
                            />
                          }
                        >
                          <Icon name="x-circle" size="md" />
                        </Show>
                      </button>
                    </Show>

                    <Show when={canRetryJob(job.status)}>
                      <button
                        onClick={() => handleRetry(job.id)}
                        disabled={actionInProgress() === job.id}
                        class="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors disabled:opacity-50"
                        title="Retry download"
                      >
                        <Show
                          when={actionInProgress() !== job.id}
                          fallback={
                            <Icon
                              name="spinner-gap"
                              size="md"
                              class="animate-spin"
                            />
                          }
                        >
                          <Icon name="arrow-counter-clockwise" size="md" />
                        </Show>
                      </button>
                    </Show>
                  </div>
                </div>
              </div>
            )}
          </For>
        </div>
      </Show>
    </div>
  );
};
