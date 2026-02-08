// Toolchain profiles service for LDF server communication

import { authFetch, getApiUrl } from "./api";

export interface ToolchainConfig {
  cross_compile_prefix?: string;
  extra_env?: Record<string, string>;
  compiler_flags?: string;
}

export interface ToolchainProfile {
  id: string;
  name: string;
  display_name: string;
  description: string;
  type: "gcc" | "llvm";
  config: ToolchainConfig;
  is_system: boolean;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateToolchainProfileRequest {
  name: string;
  display_name: string;
  description?: string;
  type: "gcc" | "llvm";
  config: ToolchainConfig;
}

export interface UpdateToolchainProfileRequest {
  name?: string;
  display_name?: string;
  description?: string;
  config?: ToolchainConfig;
}

interface ToolchainProfileListResponse {
  count: number;
  toolchain_profiles: ToolchainProfile[];
}

export type ListResult =
  | { success: true; profiles: ToolchainProfile[] }
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
  | { success: true; profile: ToolchainProfile }
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

export type CreateResult =
  | { success: true; profile: ToolchainProfile }
  | {
      success: false;
      error:
        | "unauthorized"
        | "forbidden"
        | "invalid_request"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type UpdateResult =
  | { success: true; profile: ToolchainProfile }
  | {
      success: false;
      error:
        | "not_found"
        | "unauthorized"
        | "forbidden"
        | "invalid_request"
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
        | "forbidden"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

/**
 * List all toolchain profiles, optionally filtered by type
 */
export async function listToolchainProfiles(
  type?: string,
): Promise<ListResult> {
  let path = "/toolchains";
  if (type) {
    path += `?type=${encodeURIComponent(type)}`;
  }

  const url = getApiUrl(path);
  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  const result = await authFetch<ToolchainProfileListResponse>(url);
  if (result.ok) {
    return { success: true, profiles: result.data!.toolchain_profiles };
  }

  if (result.status === 401) {
    return {
      success: false,
      error: "unauthorized",
      message: "Authentication required",
    };
  }

  return {
    success: false,
    error: "internal_error",
    message: result.error || "Failed to fetch toolchain profiles",
  };
}

/**
 * Get a single toolchain profile by ID
 */
export async function getToolchainProfile(id: string): Promise<GetResult> {
  const url = getApiUrl(`/toolchains/${id}`);
  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  const result = await authFetch<ToolchainProfile>(url);
  if (result.ok) {
    return { success: true, profile: result.data! };
  }

  if (result.status === 404) {
    return {
      success: false,
      error: "not_found",
      message: "Toolchain profile not found",
    };
  }
  if (result.status === 401) {
    return {
      success: false,
      error: "unauthorized",
      message: "Authentication required",
    };
  }

  return {
    success: false,
    error: "internal_error",
    message: result.error || "Failed to fetch toolchain profile",
  };
}

/**
 * Create a new toolchain profile
 */
export async function createToolchainProfile(
  data: CreateToolchainProfileRequest,
): Promise<CreateResult> {
  const url = getApiUrl("/toolchains");
  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  const result = await authFetch<ToolchainProfile>(url, {
    method: "POST",
    body: JSON.stringify(data),
  });

  if (result.ok) {
    return { success: true, profile: result.data! };
  }

  if (result.status === 400) {
    return {
      success: false,
      error: "invalid_request",
      message: result.error || "Invalid toolchain profile data",
    };
  }
  if (result.status === 401) {
    return {
      success: false,
      error: "unauthorized",
      message: "Authentication required",
    };
  }
  if (result.status === 403) {
    return {
      success: false,
      error: "forbidden",
      message: "Insufficient permissions",
    };
  }

  return {
    success: false,
    error: "internal_error",
    message: result.error || "Failed to create toolchain profile",
  };
}

/**
 * Update an existing toolchain profile
 */
export async function updateToolchainProfile(
  id: string,
  data: UpdateToolchainProfileRequest,
): Promise<UpdateResult> {
  const url = getApiUrl(`/toolchains/${id}`);
  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  const result = await authFetch<ToolchainProfile>(url, {
    method: "PUT",
    body: JSON.stringify(data),
  });

  if (result.ok) {
    return { success: true, profile: result.data! };
  }

  if (result.status === 400) {
    return {
      success: false,
      error: "invalid_request",
      message: result.error || "Invalid toolchain profile data",
    };
  }
  if (result.status === 401) {
    return {
      success: false,
      error: "unauthorized",
      message: "Authentication required",
    };
  }
  if (result.status === 403) {
    return {
      success: false,
      error: "forbidden",
      message: "Insufficient permissions",
    };
  }
  if (result.status === 404) {
    return {
      success: false,
      error: "not_found",
      message: "Toolchain profile not found",
    };
  }

  return {
    success: false,
    error: "internal_error",
    message: result.error || "Failed to update toolchain profile",
  };
}

/**
 * Delete a toolchain profile
 */
export async function deleteToolchainProfile(
  id: string,
): Promise<DeleteResult> {
  const url = getApiUrl(`/toolchains/${id}`);
  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  const result = await authFetch(url, { method: "DELETE" });

  if (result.ok || result.status === 204) {
    return { success: true };
  }

  if (result.status === 401) {
    return {
      success: false,
      error: "unauthorized",
      message: "Authentication required",
    };
  }
  if (result.status === 403) {
    return {
      success: false,
      error: "forbidden",
      message: result.error || "Insufficient permissions",
    };
  }
  if (result.status === 404) {
    return {
      success: false,
      error: "not_found",
      message: "Toolchain profile not found",
    };
  }

  return {
    success: false,
    error: "internal_error",
    message: result.error || "Failed to delete toolchain profile",
  };
}
