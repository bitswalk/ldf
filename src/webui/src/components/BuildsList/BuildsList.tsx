import type { Component } from "solid-js";
import { createSignal, onMount, Show, For } from "solid-js";
import { Icon } from "../Icon";
import { Label } from "../Label";
import type { LabelVariant } from "../Label";
import { Spinner } from "../Spinner";
import {
  listDistributionBuilds,
  type BuildJob,
  type BuildJobStatus,
  getStatusColor,
  getStatusDisplayText,
  getFormatDisplayText,
  formatBytes,
  formatDuration,
  isBuildActive,
} from "../../services/builds";
import { t } from "../../services/i18n";

const getStatusLabelVariant = (status: BuildJobStatus): LabelVariant => {
  const color = getStatusColor(status);
  switch (color) {
    case "success":
      return "success";
    case "danger":
      return "danger";
    case "warning":
      return "warning";
    case "primary":
      return "primary";
    default:
      return "muted";
  }
};

interface BuildsListProps {
  distributionId: string;
  onBuildClick?: (buildId: string) => void;
  limit?: number;
}

export const BuildsList: Component<BuildsListProps> = (props) => {
  const [builds, setBuilds] = createSignal<BuildJob[]>([]);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);

  const fetchBuilds = async () => {
    setLoading(true);
    setError(null);

    const result = await listDistributionBuilds(
      props.distributionId,
      props.limit || 5,
    );

    setLoading(false);

    if (result.success) {
      setBuilds(result.builds);
    } else {
      setError(result.message);
    }
  };

  onMount(() => {
    fetchBuilds();
  });

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const getStatusIcon = (status: BuildJobStatus): string => {
    switch (status) {
      case "completed":
        return "check-circle";
      case "failed":
        return "x-circle";
      case "cancelled":
        return "minus-circle";
      default:
        return "spinner-gap";
    }
  };

  return (
    <div class="space-y-3">
      <Show when={loading()}>
        <div class="flex items-center justify-center py-8">
          <Spinner size="md" />
        </div>
      </Show>

      <Show when={error()}>
        <div class="text-center py-4 text-muted-foreground">
          <Icon name="warning" size="lg" class="mx-auto mb-2 text-warning" />
          <p class="text-sm">{error()}</p>
          <button
            class="mt-2 text-sm text-primary hover:underline"
            onClick={fetchBuilds}
          >
            {t("common.actions.retry")}
          </button>
        </div>
      </Show>

      <Show when={!loading() && !error()}>
        <Show
          when={builds().length > 0}
          fallback={
            <div class="text-center py-6 text-muted-foreground">
              <Icon name="package" size="lg" class="mx-auto mb-2 opacity-50" />
              <p class="text-sm">{t("build.list.empty")}</p>
            </div>
          }
        >
          <div class="space-y-2">
            <For each={builds()}>
              {(build) => (
                <button
                  class="w-full flex items-center gap-3 p-3 rounded-md border border-border hover:bg-muted transition-colors text-left"
                  onClick={() => props.onBuildClick?.(build.id)}
                >
                  <div class="flex-shrink-0">
                    <Icon
                      name={getStatusIcon(build.status)}
                      size="md"
                      class={
                        build.status === "completed"
                          ? "text-green-500"
                          : build.status === "failed"
                            ? "text-red-500"
                            : build.status === "cancelled"
                              ? "text-yellow-500"
                              : "text-primary animate-spin"
                      }
                    />
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 flex-wrap">
                      <span class="font-medium truncate font-mono text-sm">
                        {build.id}
                      </span>
                      <Label variant={getStatusLabelVariant(build.status)}>
                        {getStatusDisplayText(build.status)}
                      </Label>
                      <Label>{build.target_arch}</Label>
                      <Label>{getFormatDisplayText(build.image_format)}</Label>
                    </div>
                    <div class="text-sm text-muted-foreground flex items-center gap-2 mt-1">
                      <span>{formatDate(build.created_at)}</span>
                      <Show when={build.completed_at && build.started_at}>
                        <span class="text-xs">
                          (
                          {formatDuration(
                            new Date(build.completed_at!).getTime() -
                              new Date(build.started_at!).getTime(),
                          )}
                          )
                        </span>
                      </Show>
                    </div>
                    <Show
                      when={
                        build.status === "completed" && build.artifact_size > 0
                      }
                    >
                      <div class="text-xs text-muted-foreground mt-1">
                        {formatBytes(build.artifact_size)}
                      </div>
                    </Show>
                    <Show when={build.error_message}>
                      <div class="text-xs text-destructive mt-1 truncate">
                        {build.error_message}
                      </div>
                    </Show>
                  </div>
                  <div class="flex-shrink-0">
                    <Show when={isBuildActive(build.status)}>
                      <div class="text-sm text-muted-foreground">
                        {build.progress_percent}%
                      </div>
                    </Show>
                    <Icon
                      name="caret-right"
                      size="sm"
                      class="text-muted-foreground"
                    />
                  </div>
                </button>
              )}
            </For>
          </div>
        </Show>
      </Show>
    </div>
  );
};
