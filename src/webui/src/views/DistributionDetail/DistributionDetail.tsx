import type { Component } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Card } from "../../components/Card";
import { Spinner } from "../../components/Spinner";
import { Icon } from "../../components/Icon";
import { DownloadStatus } from "../../components/DownloadStatus";
import {
  getDistribution,
  type Distribution,
  type DistributionStatus,
} from "../../services/distribution";

interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface DistributionDetailProps {
  distributionId: string;
  onBack: () => void;
  user?: UserInfo | null;
}

export const DistributionDetail: Component<DistributionDetailProps> = (
  props
) => {
  const [distribution, setDistribution] = createSignal<Distribution | null>(
    null
  );
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [notification, setNotification] = createSignal<{
    type: "success" | "error";
    message: string;
  } | null>(null);

  const fetchDistribution = async () => {
    setLoading(true);
    setError(null);

    const result = await getDistribution(props.distributionId);

    setLoading(false);

    if (result.success) {
      setDistribution(result.distribution);
    } else {
      setError(result.message);
    }
  };

  onMount(() => {
    fetchDistribution();
  });

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const getStatusIcon = (status: DistributionStatus): string => {
    switch (status) {
      case "ready":
        return "check-circle";
      case "pending":
      case "downloading":
      case "validating":
        return "spinner-gap";
      case "failed":
        return "x-circle";
      case "deleted":
        return "trash";
      default:
        return "circle";
    }
  };

  const getStatusColor = (status: DistributionStatus): string => {
    switch (status) {
      case "ready":
        return "text-green-500";
      case "pending":
      case "downloading":
      case "validating":
        return "text-primary";
      case "failed":
        return "text-red-500";
      case "deleted":
        return "text-muted-foreground";
      default:
        return "text-muted-foreground";
    }
  };

  const handleDownloadSuccess = (message: string) => {
    setNotification({ type: "success", message });
    setTimeout(() => setNotification(null), 3000);
  };

  const handleDownloadError = (message: string) => {
    setNotification({ type: "error", message });
    setTimeout(() => setNotification(null), 5000);
  };

  return (
    <section class="h-full w-full relative">
      <section class="h-full flex flex-col p-8 gap-6">
        {/* Header */}
        <header class="flex items-center gap-4">
          <button
            onClick={props.onBack}
            class="p-2 rounded-md hover:bg-muted transition-colors"
            title="Back to distributions"
          >
            <Icon name="arrow-left" size="lg" />
          </button>
          <div class="flex-1">
            <h1 class="text-4xl font-bold">
              {distribution()?.name || "Distribution Details"}
            </h1>
            <p class="text-muted-foreground mt-1">
              Manage distribution configuration and downloads
            </p>
          </div>
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
        <Show when={!loading() && distribution()}>
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Distribution Info */}
            <Card header={{ title: "Distribution Info" }}>
              <div class="space-y-4">
                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">Status</span>
                  <span
                    class={`flex items-center gap-2 ${getStatusColor(distribution()!.status)}`}
                  >
                    <Icon
                      name={getStatusIcon(distribution()!.status)}
                      size="sm"
                      class={
                        ["pending", "downloading", "validating"].includes(
                          distribution()!.status
                        )
                          ? "animate-spin"
                          : ""
                      }
                    />
                    <span class="capitalize">{distribution()!.status}</span>
                  </span>
                </div>

                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">Visibility</span>
                  <span class="flex items-center gap-2">
                    <Icon
                      name={
                        distribution()!.visibility === "public"
                          ? "globe"
                          : "lock"
                      }
                      size="sm"
                    />
                    <span class="capitalize">
                      {distribution()!.visibility}
                    </span>
                  </span>
                </div>

                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">Kernel Version</span>
                  <span class="font-mono">{distribution()!.version}</span>
                </div>

                <Show when={distribution()!.size_bytes > 0}>
                  <div class="flex items-center justify-between">
                    <span class="text-muted-foreground">Size</span>
                    <span class="font-mono">
                      {formatBytes(distribution()!.size_bytes)}
                    </span>
                  </div>
                </Show>

                <Show when={distribution()!.checksum}>
                  <div class="flex items-center justify-between">
                    <span class="text-muted-foreground">Checksum</span>
                    <span
                      class="font-mono text-xs truncate max-w-[200px]"
                      title={distribution()!.checksum}
                    >
                      {distribution()!.checksum}
                    </span>
                  </div>
                </Show>

                <div class="border-t border-border pt-4 mt-4 space-y-2">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">Created</span>
                    <span class="font-mono">
                      {formatDate(distribution()!.created_at)}
                    </span>
                  </div>
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-muted-foreground">Updated</span>
                    <span class="font-mono">
                      {formatDate(distribution()!.updated_at)}
                    </span>
                  </div>
                  <Show when={distribution()!.owner_id}>
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">Owner ID</span>
                      <span class="font-mono text-xs">
                        {distribution()!.owner_id}
                      </span>
                    </div>
                  </Show>
                </div>

                <Show when={distribution()!.error_message}>
                  <div class="border-t border-border pt-4 mt-4">
                    <div class="text-sm text-red-500">
                      <div class="flex items-center gap-1 mb-1">
                        <Icon name="warning" size="sm" />
                        <span class="font-medium">Error</span>
                      </div>
                      <p>{distribution()!.error_message}</p>
                    </div>
                  </div>
                </Show>
              </div>
            </Card>

            {/* Quick Actions */}
            <Card header={{ title: "Quick Actions" }}>
              <div class="space-y-3">
                <button
                  class="w-full flex items-center gap-3 p-3 rounded-md border border-border hover:bg-muted transition-colors text-left"
                  onClick={() => fetchDistribution()}
                >
                  <Icon name="arrow-clockwise" size="md" class="text-primary" />
                  <div>
                    <div class="font-medium">Refresh Status</div>
                    <div class="text-sm text-muted-foreground">
                      Update distribution info
                    </div>
                  </div>
                </button>

                <button class="w-full flex items-center gap-3 p-3 rounded-md border border-border hover:bg-muted transition-colors text-left opacity-50 cursor-not-allowed">
                  <Icon name="pencil" size="md" class="text-primary" />
                  <div>
                    <div class="font-medium">Edit Configuration</div>
                    <div class="text-sm text-muted-foreground">
                      Modify distribution settings
                    </div>
                  </div>
                </button>

                <button class="w-full flex items-center gap-3 p-3 rounded-md border border-border hover:bg-muted transition-colors text-left opacity-50 cursor-not-allowed">
                  <Icon name="export" size="md" class="text-primary" />
                  <div>
                    <div class="font-medium">Export Distribution</div>
                    <div class="text-sm text-muted-foreground">
                      Download as ISO or archive
                    </div>
                  </div>
                </button>
              </div>
            </Card>
          </div>

          {/* Download Status Section */}
          <Card header={{ title: "Component Downloads" }}>
            <DownloadStatus
              distributionId={props.distributionId}
              onSuccess={handleDownloadSuccess}
              onError={handleDownloadError}
              pollInterval={3000}
            />
          </Card>
        </Show>
      </section>
    </section>
  );
};
