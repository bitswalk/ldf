// Distribution service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";

export type DistributionStatus =
  | "pending"
  | "downloading"
  | "validating"
  | "ready"
  | "failed"
  | "deleted";

export type DistributionVisibility = "public" | "private";

export interface Distribution {
  id: string;
  name: string;
  version: string;
  status: DistributionStatus;
  visibility: DistributionVisibility;
  config?: DistributionConfig;
  source_url?: string;
  checksum?: string;
  size_bytes: number;
  owner_id?: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
  error_message?: string;
}

export interface DistributionStats {
  [status: string]: number;
}

// Configuration types matching the backend
export interface DistributionConfig {
  core: {
    kernel: {
      version: string;
    };
    bootloader: string;
    partitioning: {
      type: string;
      mode: string;
    };
  };
  system: {
    init: string;
    filesystem: {
      type: string;
      hierarchy: string;
    };
    packageManager: string;
  };
  security: {
    system: string;
  };
  runtime: {
    container: string;
    virtualization: string;
  };
  target: {
    type: string;
    desktop?: {
      environment: string;
      displayServer: string;
    };
  };
}

export interface CreateDistributionRequest {
  name: string;
  version?: string;
  config?: DistributionConfig;
  source_url?: string;
  checksum?: string;
}

export type ListResult =
  | { success: true; distributions: Distribution[] }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type GetResult =
  | { success: true; distribution: Distribution }
  | {
      success: false;
      error:
        | "not_found"
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type DeleteResult =
  | { success: true }
  | {
      success: false;
      error:
        | "not_found"
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type StatsResult =
  | { success: true; stats: DistributionStats }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type CreateResult =
  | { success: true; distribution: Distribution }
  | {
      success: false;
      error:
        | "conflict"
        | "unauthorized"
        | "invalid_request"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export interface UpdateDistributionRequest {
  name?: string;
  version?: string;
  status?: DistributionStatus;
  visibility?: DistributionVisibility;
  source_url?: string;
  checksum?: string;
  size_bytes?: number;
  config?: DistributionConfig;
}

export type UpdateResult =
  | { success: true; distribution: Distribution }
  | {
      success: false;
      error:
        | "not_found"
        | "forbidden"
        | "unauthorized"
        | "invalid_request"
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

export async function listDistributions(): Promise<ListResult> {
  const url = getApiUrl("/distributions");

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
      // API returns { count, distributions } wrapper
      const distributions: Distribution[] = data.distributions || [];
      return { success: true, distributions };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch distributions",
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

export async function getDistribution(id: string): Promise<GetResult> {
  const url = getApiUrl(`/distributions/${id}`);

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
      const distribution: Distribution = await response.json();
      return { success: true, distribution };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
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
      message: "Failed to fetch distribution",
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

export async function deleteDistribution(id: string): Promise<DeleteResult> {
  const url = getApiUrl(`/distributions/${id}`);

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
      message: "Failed to delete distribution",
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

export async function getDistributionStats(): Promise<StatsResult> {
  const url = getApiUrl("/distributions/stats");

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
      const stats: DistributionStats = await response.json();
      return { success: true, stats };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch statistics",
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

export async function createDistribution(
  request: CreateDistributionRequest,
): Promise<CreateResult> {
  const url = getApiUrl("/distributions");

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
      body: JSON.stringify(request),
    });

    if (response.ok) {
      const distribution: Distribution = await response.json();
      return { success: true, distribution };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 400) {
      return {
        success: false,
        error: "invalid_request",
        message: "Invalid distribution data",
      };
    }

    if (response.status === 409) {
      return {
        success: false,
        error: "conflict",
        message: "A distribution with this name already exists",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to create distribution",
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

export async function updateDistribution(
  id: string,
  request: UpdateDistributionRequest,
): Promise<UpdateResult> {
  const url = getApiUrl(`/distributions/${id}`);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "PUT",
      headers: getAuthHeaders(),
      body: JSON.stringify(request),
    });

    if (response.ok) {
      const distribution: Distribution = await response.json();
      return { success: true, distribution };
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
        message: "You don't have permission to update this distribution",
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
      return {
        success: false,
        error: "invalid_request",
        message: "Invalid distribution data",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to update distribution",
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
