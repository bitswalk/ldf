// Source Versions service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";
import type { Source } from "./sources";

export type VersionType = "mainline" | "stable" | "longterm" | "linux-next";

export interface SourceVersion {
  id: string;
  source_id: string;
  source_type: string;
  version: string;
  version_type: VersionType;
  release_date?: string;
  download_url?: string;
  checksum?: string;
  checksum_type?: string;
  file_size?: number;
  is_stable: boolean;
  discovered_at: string;
}

export interface VersionSyncJob {
  id: string;
  source_id: string;
  source_type: string;
  status: "pending" | "running" | "completed" | "failed";
  versions_found: number;
  versions_new: number;
  started_at?: string;
  completed_at?: string;
  error_message?: string;
  created_at: string;
}

export type ListVersionsResult =
  | {
      success: true;
      versions: SourceVersion[];
      total: number;
      syncJob?: VersionSyncJob;
    }
  | {
      success: false;
      error:
        | "not_found"
        | "forbidden"
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type SyncResult =
  | { success: true; jobId: string; message: string }
  | {
      success: false;
      error:
        | "conflict"
        | "not_found"
        | "forbidden"
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type SyncStatusResult =
  | { success: true; job: VersionSyncJob | null }
  | {
      success: false;
      error:
        | "not_found"
        | "forbidden"
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type GetSourceResult =
  | { success: true; source: Source }
  | {
      success: false;
      error:
        | "not_found"
        | "forbidden"
        | "unauthorized"
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

// Get a single source by ID
export async function getSource(sourceId: string): Promise<GetSourceResult> {
  const url = getApiUrl(`/sources/${sourceId}`);

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
      const source = await response.json();
      return { success: true, source };
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
        message: "Source not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch source",
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

// List versions for a source
export async function listSourceVersions(
  sourceId: string,
  limit: number = 50,
  offset: number = 0,
  versionTypeFilter?: string,
): Promise<ListVersionsResult> {
  const params = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
  });
  if (versionTypeFilter && versionTypeFilter !== "all") {
    params.set("version_type", versionTypeFilter);
  }
  const url = getApiUrl(`/sources/${sourceId}/versions?${params.toString()}`);

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
        versions: data.versions || [],
        total: data.total || 0,
        syncJob: data.sync_job,
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
        message: "Source not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch versions",
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

// Trigger a version sync for a source
export async function triggerVersionSync(
  sourceId: string,
): Promise<SyncResult> {
  const url = getApiUrl(`/sources/${sourceId}/sync`);

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

    if (response.ok || response.status === 202) {
      const data = await response.json();
      return {
        success: true,
        jobId: data.job_id,
        message: data.message || "Sync started",
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
        message: "Source not found",
      };
    }

    if (response.status === 409) {
      return {
        success: false,
        error: "conflict",
        message: "A sync is already in progress",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to start sync",
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

// Get sync status for a source
export async function getSyncStatus(
  sourceId: string,
): Promise<SyncStatusResult> {
  const url = getApiUrl(`/sources/${sourceId}/sync/status`);

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
        job: data.job || null,
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
        message: "Source not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to get sync status",
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

// Helper to format version for display
export function formatVersion(version: SourceVersion): string {
  return version.version;
}

// Helper to format relative time
export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins} minute${diffMins > 1 ? "s" : ""} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? "s" : ""} ago`;
  if (diffDays < 30) return `${diffDays} day${diffDays > 1 ? "s" : ""} ago`;

  return date.toLocaleDateString();
}

// Helper to check if sync is in progress
export function isSyncInProgress(
  job: VersionSyncJob | null | undefined,
): boolean {
  if (!job) return false;
  return job.status === "pending" || job.status === "running";
}

export type ClearVersionsResult =
  | { success: true; message: string }
  | {
      success: false;
      error:
        | "conflict"
        | "not_found"
        | "forbidden"
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

// Clear all cached versions for a source
export async function clearSourceVersions(
  sourceId: string,
): Promise<ClearVersionsResult> {
  const url = getApiUrl(`/sources/${sourceId}/versions`);

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
      const data = await response.json();
      return {
        success: true,
        message: data.message || "Version cache cleared",
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
        message: "Source not found",
      };
    }

    if (response.status === 409) {
      return {
        success: false,
        error: "conflict",
        message: "Cannot clear versions while sync is in progress",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to clear versions",
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

export type GetVersionTypesResult =
  | { success: true; types: string[] }
  | {
      success: false;
      error:
        | "not_found"
        | "forbidden"
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

// Get distinct version types for a source
export async function getSourceVersionTypes(
  sourceId: string,
): Promise<GetVersionTypesResult> {
  const url = getApiUrl(`/sources/${sourceId}/versions/types`);

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
        types: data.types || [],
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
        message: "Source not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to get version types",
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
