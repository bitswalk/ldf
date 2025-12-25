// Downloads service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";

export type DownloadJobStatus =
  | "pending"
  | "verifying"
  | "downloading"
  | "completed"
  | "failed"
  | "cancelled";

export interface DownloadJob {
  id: string;
  distribution_id: string;
  component_id: string;
  source_id: string;
  source_type: string;
  resolved_url: string;
  version: string;
  status: DownloadJobStatus;
  progress_bytes: number;
  total_bytes: number;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  artifact_path?: string;
  checksum?: string;
  error_message?: string;
  retry_count: number;
  max_retries: number;
  component_name?: string;
  progress: number;
}

export interface StartDownloadsRequest {
  components?: string[];
}

export type ListResult =
  | { success: true; jobs: DownloadJob[] }
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

export type StartResult =
  | { success: true; jobs: DownloadJob[] }
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

export type GetResult =
  | { success: true; job: DownloadJob }
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

// Start downloads for a distribution
export async function startDownloads(
  distributionId: string,
  components?: string[]
): Promise<StartResult> {
  const url = getApiUrl(`/distributions/${distributionId}/downloads`);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const body: StartDownloadsRequest = {};
    if (components && components.length > 0) {
      body.components = components;
    }

    const response = await fetch(url, {
      method: "POST",
      headers: getAuthHeaders(),
      body: JSON.stringify(body),
    });

    if (response.ok) {
      const data = await response.json();
      const jobs: DownloadJob[] = data.jobs || [];
      return { success: true, jobs };
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

    return {
      success: false,
      error: "internal_error",
      message: "Failed to start downloads",
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

// List downloads for a distribution
export async function listDownloads(
  distributionId: string
): Promise<ListResult> {
  const url = getApiUrl(`/distributions/${distributionId}/downloads`);

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
      const jobs: DownloadJob[] = data.jobs || [];
      return { success: true, jobs };
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
      message: "Failed to fetch downloads",
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

// Get a specific download job
export async function getDownload(jobId: string): Promise<GetResult> {
  const url = getApiUrl(`/downloads/${jobId}`);

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
      const job = await response.json();
      return { success: true, job };
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
        message: "Download job not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch download job",
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

// Cancel a download job
export async function cancelDownload(jobId: string): Promise<ActionResult> {
  const url = getApiUrl(`/downloads/${jobId}/cancel`);

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
        message: "Download job not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to cancel download",
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

// Retry a failed download job
export async function retryDownload(jobId: string): Promise<GetResult> {
  const url = getApiUrl(`/downloads/${jobId}/retry`);

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
      const job = await response.json();
      return { success: true, job };
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
        message: "Download job not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to retry download",
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

// List active downloads (admin only)
export async function listActiveDownloads(): Promise<ListResult> {
  const url = getApiUrl("/downloads/active");

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
      const jobs: DownloadJob[] = data.jobs || [];
      return { success: true, jobs };
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
      message: "Failed to fetch active downloads",
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

// Get status display text
export function getStatusDisplayText(status: DownloadJobStatus): string {
  const texts: Record<DownloadJobStatus, string> = {
    pending: "Pending",
    verifying: "Verifying",
    downloading: "Downloading",
    completed: "Completed",
    failed: "Failed",
    cancelled: "Cancelled",
  };
  return texts[status] || status;
}

// Get status color class
export function getStatusColor(
  status: DownloadJobStatus
): "default" | "primary" | "success" | "warning" | "danger" {
  const colors: Record<
    DownloadJobStatus,
    "default" | "primary" | "success" | "warning" | "danger"
  > = {
    pending: "default",
    verifying: "primary",
    downloading: "primary",
    completed: "success",
    failed: "danger",
    cancelled: "warning",
  };
  return colors[status] || "default";
}

// Check if job is active (can be cancelled)
export function isJobActive(status: DownloadJobStatus): boolean {
  return status === "pending" || status === "verifying" || status === "downloading";
}

// Check if job can be retried
export function canRetryJob(status: DownloadJobStatus): boolean {
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
