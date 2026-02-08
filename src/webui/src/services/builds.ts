// Builds service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";

export type BuildJobStatus =
  | "pending"
  | "resolving"
  | "preparing"
  | "compiling"
  | "assembling"
  | "packaging"
  | "completed"
  | "failed"
  | "cancelled";

export type BuildStageName =
  | "resolve"
  | "download"
  | "prepare"
  | "compile"
  | "assemble"
  | "package";

export type TargetArch = "x86_64" | "aarch64";

export type ImageFormat = "raw" | "qcow2" | "iso";

export interface BuildStage {
  id: number;
  build_id: string;
  name: BuildStageName;
  status: string;
  progress_percent: number;
  started_at?: string;
  completed_at?: string;
  duration_ms: number;
  error_message?: string;
  log_path?: string;
}

export interface BuildJob {
  id: string;
  distribution_id: string;
  owner_id: string;
  status: BuildJobStatus;
  current_stage: string;
  target_arch: TargetArch;
  image_format: ImageFormat;
  progress_percent: number;
  workspace_path?: string;
  artifact_path?: string;
  artifact_checksum?: string;
  artifact_size: number;
  error_message?: string;
  error_stage?: string;
  retry_count: number;
  max_retries: number;
  config_snapshot?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  stages?: BuildStage[];
}

export interface BuildLog {
  id: number;
  build_id: string;
  stage: string;
  level: string;
  message: string;
  created_at: string;
}

export interface StartBuildRequest {
  arch: TargetArch;
  format: ImageFormat;
  clear_cache?: boolean;
}

export type StartResult =
  | { success: true; build: BuildJob }
  | {
      success: false;
      error:
        | "unauthorized"
        | "forbidden"
        | "not_found"
        | "bad_request"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type GetResult =
  | { success: true; build: BuildJob }
  | {
      success: false;
      error:
        | "unauthorized"
        | "forbidden"
        | "not_found"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type ListResult =
  | { success: true; builds: BuildJob[]; count: number }
  | {
      success: false;
      error:
        | "unauthorized"
        | "forbidden"
        | "not_found"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type LogsResult =
  | { success: true; logs: BuildLog[]; count: number }
  | {
      success: false;
      error:
        | "unauthorized"
        | "forbidden"
        | "not_found"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type ActionResult =
  | { success: true }
  | {
      success: false;
      error:
        | "unauthorized"
        | "forbidden"
        | "not_found"
        | "bad_request"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

function getApiUrl(path: string): string | null {
  const serverUrl = getServerUrl();
  if (!serverUrl) return null;
  return `${serverUrl}/v1${path}`;
}

function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = getAuthToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  return headers;
}

// Start a build for a distribution
export async function startBuild(
  distributionId: string,
  arch: TargetArch,
  format: ImageFormat,
  clearCache: boolean = false,
): Promise<StartResult> {
  const url = getApiUrl(`/distributions/${distributionId}/build`);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const body: StartBuildRequest = { arch, format, clear_cache: clearCache };

    const response = await fetch(url, {
      method: "POST",
      headers: getAuthHeaders(),
      body: JSON.stringify(body),
    });

    if (response.ok) {
      const build = await response.json();
      return { success: true, build };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Write access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Distribution not found",
      };
    }

    if (response.status === 400) {
      const data = await response.json().catch(() => ({}));
      return {
        success: false,
        error: "bad_request",
        message: data.message || "Invalid request",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to start build",
    };
  } catch (err) {
    return {
      success: false,
      error: "network_error",
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

// Get a specific build job
export async function getBuild(buildId: string): Promise<GetResult> {
  const url = getApiUrl(`/builds/${buildId}`);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "GET",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      const build = await response.json();
      return { success: true, build };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Access denied",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Build not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch build",
    };
  } catch (err) {
    return {
      success: false,
      error: "network_error",
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

// List builds for a distribution
export async function listDistributionBuilds(
  distributionId: string,
  limit?: number,
  offset?: number,
): Promise<ListResult> {
  let path = `/distributions/${distributionId}/builds`;
  const params = new URLSearchParams();
  if (limit !== undefined) params.set("limit", String(limit));
  if (offset !== undefined) params.set("offset", String(offset));
  if (params.toString()) path += `?${params.toString()}`;

  const url = getApiUrl(path);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "GET",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      const data = await response.json();
      return {
        success: true,
        builds: data.builds || [],
        count: data.total || data.count || 0,
      };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Access denied",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Distribution not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch builds",
    };
  } catch (err) {
    return {
      success: false,
      error: "network_error",
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

// Clear all builds for a distribution
export async function clearDistributionBuilds(
  distributionId: string,
): Promise<ActionResult> {
  const url = getApiUrl(`/distributions/${distributionId}/builds`);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "DELETE",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      return { success: true };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Access denied",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Distribution not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to clear builds",
    };
  } catch (err) {
    return {
      success: false,
      error: "network_error",
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

// Get build logs
export async function getBuildLogs(
  buildId: string,
  stage?: string,
  limit?: number,
  offset?: number,
): Promise<LogsResult> {
  let path = `/builds/${buildId}/logs`;
  const params = new URLSearchParams();
  if (stage) params.set("stage", stage);
  if (limit !== undefined) params.set("limit", String(limit));
  if (offset !== undefined) params.set("offset", String(offset));
  if (params.toString()) path += `?${params.toString()}`;

  const url = getApiUrl(path);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "GET",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      const data = await response.json();
      return {
        success: true,
        logs: data.logs || [],
        count: data.count || 0,
      };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Access denied",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Build not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch build logs",
    };
  } catch (err) {
    return {
      success: false,
      error: "network_error",
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

// Stream build logs via SSE
export function streamBuildLogs(
  buildId: string,
  onLog: (log: BuildLog) => void,
  onError?: (error: Error) => void,
  onComplete?: () => void,
  onStatus?: (status: Partial<BuildJob>) => void,
): () => void {
  const serverUrl = getServerUrl();
  if (!serverUrl) {
    onError?.(new Error("Server connection not configured"));
    return () => {};
  }

  const token = getAuthToken();
  let url = `${serverUrl}/v1/builds/${buildId}/logs/stream`;
  if (token) {
    url += `?token=${encodeURIComponent(token)}`;
  }

  const eventSource = new EventSource(url);

  eventSource.onmessage = (event) => {
    try {
      const log = JSON.parse(event.data) as BuildLog;
      onLog(log);
    } catch (err) {
      console.error("Failed to parse SSE log:", err);
    }
  };

  eventSource.addEventListener("status", (event) => {
    try {
      const status = JSON.parse(
        (event as MessageEvent).data,
      ) as Partial<BuildJob>;
      onStatus?.(status);
    } catch (err) {
      console.error("Failed to parse SSE status:", err);
    }
  });

  eventSource.addEventListener("done", (event) => {
    try {
      const status = JSON.parse(
        (event as MessageEvent).data,
      ) as Partial<BuildJob>;
      onStatus?.(status);
    } catch {
      // done event may have minimal payload
    }
    eventSource.close();
    onComplete?.();
  });

  eventSource.onerror = () => {
    if (eventSource.readyState === EventSource.CLOSED) {
      onComplete?.();
    } else {
      onError?.(new Error("SSE connection error"));
    }
    eventSource.close();
  };

  // Return cleanup function
  return () => {
    eventSource.close();
  };
}

// Cancel a build
export async function cancelBuild(buildId: string): Promise<ActionResult> {
  const url = getApiUrl(`/builds/${buildId}/cancel`);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      return { success: true };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Write access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Build not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to cancel build",
    };
  } catch (err) {
    return {
      success: false,
      error: "network_error",
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

// Retry a failed build
export async function retryBuild(buildId: string): Promise<ActionResult> {
  const url = getApiUrl(`/builds/${buildId}/retry`);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      return { success: true };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Write access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Build not found",
      };
    }

    if (response.status === 400) {
      const data = await response.json().catch(() => ({}));
      return {
        success: false,
        error: "bad_request",
        message: data.message || "Cannot retry this build",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to retry build",
    };
  } catch (err) {
    return {
      success: false,
      error: "network_error",
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

// List active builds (admin only)
export async function listActiveBuilds(): Promise<ListResult> {
  const url = getApiUrl("/builds/active");

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "GET",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      const data = await response.json();
      return {
        success: true,
        builds: data.builds || [],
        count: data.count || 0,
      };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Admin access required",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch active builds",
    };
  } catch (err) {
    return {
      success: false,
      error: "network_error",
      message:
        err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

// Helper functions

// Get status display text
export function getStatusDisplayText(status: BuildJobStatus): string {
  const texts: Record<BuildJobStatus, string> = {
    pending: "Pending",
    resolving: "Resolving",
    preparing: "Preparing",
    compiling: "Compiling",
    assembling: "Assembling",
    packaging: "Packaging",
    completed: "Completed",
    failed: "Failed",
    cancelled: "Cancelled",
  };
  return texts[status] || status;
}

// Get status color
export function getStatusColor(
  status: BuildJobStatus,
): "default" | "primary" | "success" | "warning" | "danger" {
  const colors: Record<
    BuildJobStatus,
    "default" | "primary" | "success" | "warning" | "danger"
  > = {
    pending: "default",
    resolving: "primary",
    preparing: "primary",
    compiling: "primary",
    assembling: "primary",
    packaging: "primary",
    completed: "success",
    failed: "danger",
    cancelled: "warning",
  };
  return colors[status] || "default";
}

// Get stage display text
export function getStageDisplayText(stage: BuildStageName): string {
  const texts: Record<BuildStageName, string> = {
    resolve: "Resolve",
    download: "Download",
    prepare: "Prepare",
    compile: "Compile",
    assemble: "Assemble",
    package: "Package",
  };
  return texts[stage] || stage;
}

// Get architecture display text
export function getArchDisplayText(arch: TargetArch): string {
  const texts: Record<TargetArch, string> = {
    x86_64: "x86_64 (AMD64)",
    aarch64: "AArch64 (ARM64)",
  };
  return texts[arch] || arch;
}

// Get image format display text
export function getFormatDisplayText(format: ImageFormat): string {
  const texts: Record<ImageFormat, string> = {
    raw: "Raw Disk Image",
    qcow2: "QCOW2 (QEMU)",
    iso: "ISO (Bootable)",
  };
  return texts[format] || format;
}

// Check if build is active (can be cancelled)
export function isBuildActive(status: BuildJobStatus): boolean {
  return (
    status === "pending" ||
    status === "resolving" ||
    status === "preparing" ||
    status === "compiling" ||
    status === "assembling" ||
    status === "packaging"
  );
}

// Check if build can be retried
export function canRetryBuild(status: BuildJobStatus): boolean {
  return status === "failed" || status === "cancelled";
}

// Format bytes to human readable
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

// Format duration in milliseconds to human readable
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
}
