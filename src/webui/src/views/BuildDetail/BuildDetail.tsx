import type { Component } from "solid-js";
import {
  createSignal,
  onMount,
  onCleanup,
  Show,
  For,
  createEffect,
} from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { Label } from "../../components/Label";
import type { LabelVariant } from "../../components/Label";
import {
  getBuild,
  getBuildLogs,
  streamBuildLogs,
  cancelBuild,
  retryBuild,
  type BuildJob,
  type BuildLog,
  type BuildStage,
  type BuildJobStatus,
  type BuildStageName,
  getStatusColor,
  getStatusDisplayText,
  getStageDisplayText,
  getArchDisplayText,
  getFormatDisplayText,
  isBuildActive,
  canRetryBuild,
  formatBytes,
  formatDuration,
} from "../../services/builds";
import { t } from "../../services/i18n";

interface BuildDetailProps {
  buildId: string;
  onBack: () => void;
}

export const BuildDetail: Component<BuildDetailProps> = (props) => {
  const [build, setBuild] = createSignal<BuildJob | null>(null);
  const [logs, setLogs] = createSignal<BuildLog[]>([]);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [selectedStage, setSelectedStage] = createSignal<string | null>(null);
  const [isStreaming, setIsStreaming] = createSignal(false);
  const [actionLoading, setActionLoading] = createSignal(false);

  let logContainerRef: HTMLDivElement | undefined;
  let cleanupStream: (() => void) | null = null;

  const fetchBuild = async () => {
    const result = await getBuild(props.buildId);

    if (result.success) {
      setBuild(result.build);
      return result.build;
    } else {
      setError(result.message);
      return null;
    }
  };

  const fetchLogs = async () => {
    const result = await getBuildLogs(
      props.buildId,
      selectedStage() || undefined,
      100,
    );

    if (result.success) {
      setLogs(result.logs);
    }
  };

  const startLogStream = () => {
    if (cleanupStream) {
      cleanupStream();
    }

    setIsStreaming(true);

    cleanupStream = streamBuildLogs(
      props.buildId,
      (log) => {
        setLogs((prev) => [...prev, log]);
        // Auto-scroll to bottom
        if (logContainerRef) {
          setTimeout(() => {
            logContainerRef!.scrollTop = logContainerRef!.scrollHeight;
          }, 10);
        }
      },
      (err) => {
        console.error("SSE error:", err);
        setIsStreaming(false);
      },
      () => {
        setIsStreaming(false);
        // Final refresh to get complete build data
        fetchBuild();
      },
      (statusUpdate) => {
        // Merge status fields into current build for real-time updates
        setBuild((prev) => (prev ? { ...prev, ...statusUpdate } : prev));
      },
    );
  };

  const stopLogStream = () => {
    if (cleanupStream) {
      cleanupStream();
      cleanupStream = null;
    }
    setIsStreaming(false);
  };

  onMount(async () => {
    setLoading(true);
    const buildData = await fetchBuild();
    await fetchLogs();
    setLoading(false);

    // Start streaming if build is active
    if (buildData && isBuildActive(buildData.status)) {
      startLogStream();
    }
  });

  onCleanup(() => {
    stopLogStream();
  });

  // Watch for build completion to stop streaming
  createEffect(() => {
    const b = build();
    if (b && !isBuildActive(b.status) && isStreaming()) {
      stopLogStream();
    }
  });

  const handleCancel = async () => {
    setActionLoading(true);
    const result = await cancelBuild(props.buildId);
    setActionLoading(false);

    if (result.success) {
      await fetchBuild();
    }
  };

  const handleRetry = async () => {
    setActionLoading(true);
    const result = await retryBuild(props.buildId);
    setActionLoading(false);

    if (result.success) {
      await fetchBuild();
      setLogs([]);
      startLogStream();
    }
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const getStageIcon = (status: string): string => {
    switch (status) {
      case "completed":
        return "check-circle";
      case "running":
        return "spinner-gap";
      case "failed":
        return "x-circle";
      default:
        return "circle";
    }
  };

  const getStageIconClass = (status: string): string => {
    switch (status) {
      case "completed":
        return "text-green-500";
      case "running":
        return "text-primary animate-spin";
      case "failed":
        return "text-red-500";
      default:
        return "text-muted-foreground";
    }
  };

  const getLogLevelClass = (level: string): string => {
    switch (level) {
      case "error":
        return "text-red-500";
      case "warn":
        return "text-yellow-500";
      case "info":
        return "text-foreground";
      default:
        return "text-muted-foreground";
    }
  };

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

  return (
    <section class="h-full w-full relative">
      <section class="h-full flex flex-col p-8 gap-6 overflow-auto">
        {/* Header */}
        <header class="flex items-center gap-4">
          <button
            onClick={props.onBack}
            class="p-2 rounded-md hover:bg-muted transition-colors"
            title={t("build.detail.title")}
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <div class="flex items-center gap-3">
              <h1 class="text-4xl font-bold">{t("build.detail.title")}</h1>
              <Show when={build()}>
                <Label variant={getStatusLabelVariant(build()!.status)}>
                  {getStatusDisplayText(build()!.status)}
                </Label>
                <Label>{build()!.target_arch}</Label>
                <Label>{getFormatDisplayText(build()!.image_format)}</Label>
              </Show>
            </div>
            <p class="text-muted-foreground mt-1">{props.buildId}</p>
          </div>
          <Show when={build()}>
            <div class="flex items-center gap-2">
              <Show when={isBuildActive(build()!.status)}>
                <button
                  onClick={handleCancel}
                  disabled={actionLoading()}
                  class="flex items-center gap-2 px-4 py-2 border border-destructive/50 text-destructive rounded-md hover:bg-destructive/10 transition-colors disabled:opacity-50"
                >
                  <Show
                    when={actionLoading()}
                    fallback={<Icon name="x" size="sm" />}
                  >
                    <Spinner size="sm" />
                  </Show>
                  {t("build.detail.actions.cancel")}
                </button>
              </Show>
              <Show when={canRetryBuild(build()!.status)}>
                <button
                  onClick={handleRetry}
                  disabled={actionLoading()}
                  class="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  <Show
                    when={actionLoading()}
                    fallback={<Icon name="arrow-clockwise" size="sm" />}
                  >
                    <Spinner size="sm" />
                  </Show>
                  {t("build.detail.actions.retry")}
                </button>
              </Show>
            </div>
          </Show>
        </header>

        {/* Loading State */}
        <Show when={loading()}>
          <div class="flex items-center justify-center py-12">
            <Spinner size="lg" />
          </div>
        </Show>

        {/* Error State */}
        <Show when={error()}>
          <div class="text-center py-12">
            <Icon
              name="warning"
              size="xl"
              class="mx-auto mb-4 text-destructive"
            />
            <p class="text-lg font-medium">{t("common.errors.loadFailed")}</p>
            <p class="text-muted-foreground mt-1">{error()}</p>
            <button
              onClick={() => {
                setError(null);
                setLoading(true);
                fetchBuild().then(() => setLoading(false));
              }}
              class="mt-4 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              {t("common.actions.retry")}
            </button>
          </div>
        </Show>

        {/* Build Details */}
        <Show when={!loading() && !error() && build()}>
          <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Left Column: Build Info + Stages */}
            <div class="lg:col-span-1 space-y-6">
              {/* Build Info */}
              <Card header={{ title: t("build.detail.info.title") }}>
                <div class="space-y-1">
                  <article class="flex items-center justify-between py-2">
                    <span class="text-muted-foreground text-sm">
                      {t("build.detail.info.status")}
                    </span>
                    <Label variant={getStatusLabelVariant(build()!.status)}>
                      {getStatusDisplayText(build()!.status)}
                    </Label>
                  </article>

                  <article class="flex items-center justify-between py-2">
                    <span class="text-muted-foreground text-sm">
                      {t("build.detail.info.architecture")}
                    </span>
                    <span class="text-sm font-medium">
                      {build()!.target_arch}
                    </span>
                  </article>

                  <article class="flex items-center justify-between py-2">
                    <span class="text-muted-foreground text-sm">
                      {t("build.detail.info.format")}
                    </span>
                    <span class="text-sm font-medium">
                      {getFormatDisplayText(build()!.image_format)}
                    </span>
                  </article>

                  <article class="flex items-center justify-between py-2">
                    <span class="text-muted-foreground text-sm">
                      {t("build.detail.info.progress")}
                    </span>
                    <span class="text-sm font-medium">
                      {build()!.progress_percent}%
                    </span>
                  </article>

                  <div class="border-t border-border pt-2 mt-2">
                    <article class="flex items-center justify-between py-2">
                      <span class="text-muted-foreground text-sm">
                        {t("build.detail.info.created")}
                      </span>
                      <span class="text-sm">
                        {formatDate(build()!.created_at)}
                      </span>
                    </article>

                    <Show when={build()!.started_at}>
                      <article class="flex items-center justify-between py-2">
                        <span class="text-muted-foreground text-sm">
                          {t("build.detail.info.started")}
                        </span>
                        <span class="text-sm">
                          {formatDate(build()!.started_at!)}
                        </span>
                      </article>
                    </Show>

                    <Show when={build()!.completed_at}>
                      <article class="flex items-center justify-between py-2">
                        <span class="text-muted-foreground text-sm">
                          {t("build.detail.info.completed")}
                        </span>
                        <span class="text-sm">
                          {formatDate(build()!.completed_at!)}
                        </span>
                      </article>
                    </Show>

                    <Show when={build()!.completed_at && build()!.started_at}>
                      <article class="flex items-center justify-between py-2">
                        <span class="text-muted-foreground text-sm">
                          {t("build.detail.info.duration")}
                        </span>
                        <span class="text-sm font-medium">
                          {formatDuration(
                            new Date(build()!.completed_at!).getTime() -
                              new Date(build()!.started_at!).getTime(),
                          )}
                        </span>
                      </article>
                    </Show>
                  </div>

                  <Show when={build()!.artifact_size > 0}>
                    <article class="flex items-center justify-between py-2">
                      <span class="text-muted-foreground text-sm">
                        {t("build.detail.info.artifactSize")}
                      </span>
                      <span class="text-sm font-medium">
                        {formatBytes(build()!.artifact_size)}
                      </span>
                    </article>
                  </Show>

                  <Show when={build()!.error_message}>
                    <div class="p-3 bg-destructive/10 border border-destructive/20 rounded-md mt-2">
                      <p class="text-sm text-destructive font-medium">
                        {t("build.detail.info.error")}
                      </p>
                      <p class="text-sm text-destructive mt-1">
                        {build()!.error_message}
                      </p>
                    </div>
                  </Show>
                </div>
              </Card>

              {/* Stages */}
              <Card header={{ title: t("build.detail.stages.title") }}>
                <div class="space-y-2">
                  <Show
                    when={build()!.stages && build()!.stages!.length > 0}
                    fallback={
                      <p class="text-muted-foreground text-sm py-4 text-center">
                        {t("build.detail.stages.empty")}
                      </p>
                    }
                  >
                    <For each={build()!.stages}>
                      {(stage) => (
                        <button
                          class={`w-full flex items-center gap-3 p-3 rounded-md border transition-colors text-left ${
                            selectedStage() === stage.name
                              ? "border-primary bg-primary/10"
                              : "border-border hover:bg-muted"
                          }`}
                          onClick={() => {
                            setSelectedStage(
                              selectedStage() === stage.name
                                ? null
                                : stage.name,
                            );
                            fetchLogs();
                          }}
                        >
                          <Icon
                            name={getStageIcon(stage.status)}
                            size="sm"
                            class={getStageIconClass(stage.status)}
                          />
                          <div class="flex-1">
                            <div class="font-medium text-sm">
                              {getStageDisplayText(
                                stage.name as BuildStageName,
                              )}
                            </div>
                            <Show when={stage.duration_ms > 0}>
                              <div class="text-xs text-muted-foreground">
                                {formatDuration(stage.duration_ms)}
                              </div>
                            </Show>
                          </div>
                          <Show when={stage.status === "running"}>
                            <span class="text-xs text-muted-foreground">
                              {stage.progress_percent}%
                            </span>
                          </Show>
                        </button>
                      )}
                    </For>
                  </Show>
                </div>
              </Card>
            </div>

            {/* Right Column: Logs */}
            <div class="lg:col-span-2">
              <Card
                header={{
                  title: t("build.detail.logs.title"),
                  action: (
                    <div class="flex items-center gap-2">
                      <Show when={isStreaming()}>
                        <span class="flex items-center gap-1 text-xs text-primary">
                          <span class="w-2 h-2 bg-primary rounded-full animate-pulse" />
                          {t("build.detail.logs.streaming")}
                        </span>
                      </Show>
                      <Show when={selectedStage()}>
                        <button
                          onClick={() => {
                            setSelectedStage(null);
                            fetchLogs();
                          }}
                          class="text-xs text-muted-foreground hover:text-foreground"
                        >
                          {t("build.detail.logs.clearFilter")}
                        </button>
                      </Show>
                    </div>
                  ),
                }}
              >
                <div
                  ref={logContainerRef}
                  class="h-[500px] overflow-y-auto font-mono text-sm bg-muted/50 rounded-md p-4"
                >
                  <Show
                    when={logs().length > 0}
                    fallback={
                      <div class="text-center py-8 text-muted-foreground">
                        <Icon
                          name="file-text"
                          size="lg"
                          class="mx-auto mb-2 opacity-50"
                        />
                        <p>{t("build.detail.logs.empty")}</p>
                      </div>
                    }
                  >
                    <div class="space-y-1">
                      <For each={logs()}>
                        {(log) => (
                          <div class="flex gap-2">
                            <span class="text-muted-foreground flex-shrink-0 w-20">
                              {new Date(log.created_at).toLocaleTimeString()}
                            </span>
                            <span
                              class={`flex-shrink-0 w-12 uppercase text-xs ${getLogLevelClass(log.level)}`}
                            >
                              [{log.level}]
                            </span>
                            <span class={getLogLevelClass(log.level)}>
                              {log.message}
                            </span>
                          </div>
                        )}
                      </For>
                    </div>
                  </Show>
                </div>
              </Card>
            </div>
          </div>
        </Show>
      </section>
    </section>
  );
};
