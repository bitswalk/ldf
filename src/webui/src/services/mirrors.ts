// Mirror management service for LDF server communication

import { authFetch, getApiUrl } from "./api";

export interface Mirror {
  id: string;
  name: string;
  url_prefix: string;
  mirror_url: string;
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateMirrorRequest {
  name: string;
  url_prefix: string;
  mirror_url: string;
  priority?: number;
  enabled?: boolean;
}

export interface UpdateMirrorRequest {
  name?: string;
  url_prefix?: string;
  mirror_url?: string;
  priority?: number;
  enabled?: boolean;
}

interface MirrorListResponse {
  count: number;
  mirrors: Mirror[];
}

export type ListResult =
  | { success: true; mirrors: Mirror[] }
  | {
      success: false;
      error: "unauthorized" | "forbidden" | "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export type CreateResult =
  | { success: true; mirror: Mirror }
  | {
      success: false;
      error: "unauthorized" | "forbidden" | "invalid_request" | "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export type UpdateResult =
  | { success: true; mirror: Mirror }
  | {
      success: false;
      error: "not_found" | "unauthorized" | "forbidden" | "invalid_request" | "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export type DeleteResult =
  | { success: true }
  | {
      success: false;
      error: "not_found" | "unauthorized" | "forbidden" | "network_error" | "not_configured" | "internal_error";
      message: string;
    };

/**
 * List all configured mirrors
 */
export async function listMirrors(): Promise<ListResult> {
  const url = getApiUrl("/mirrors");
  if (!url) {
    return { success: false, error: "not_configured", message: "Server connection not configured" };
  }

  const result = await authFetch<MirrorListResponse>(url);
  if (result.ok) {
    return { success: true, mirrors: result.data!.mirrors };
  }

  if (result.status === 401) {
    return { success: false, error: "unauthorized", message: "Authentication required" };
  }
  if (result.status === 403) {
    return { success: false, error: "forbidden", message: "Admin access required" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to fetch mirrors" };
}

/**
 * Create a new mirror configuration
 */
export async function createMirror(data: CreateMirrorRequest): Promise<CreateResult> {
  const url = getApiUrl("/mirrors");
  if (!url) {
    return { success: false, error: "not_configured", message: "Server connection not configured" };
  }

  const result = await authFetch<Mirror>(url, {
    method: "POST",
    body: JSON.stringify(data),
  });

  if (result.ok) {
    return { success: true, mirror: result.data! };
  }

  if (result.status === 400) {
    return { success: false, error: "invalid_request", message: result.error || "Invalid mirror data" };
  }
  if (result.status === 401) {
    return { success: false, error: "unauthorized", message: "Authentication required" };
  }
  if (result.status === 403) {
    return { success: false, error: "forbidden", message: "Admin access required" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to create mirror" };
}

/**
 * Update an existing mirror configuration
 */
export async function updateMirror(id: string, data: UpdateMirrorRequest): Promise<UpdateResult> {
  const url = getApiUrl(`/mirrors/${id}`);
  if (!url) {
    return { success: false, error: "not_configured", message: "Server connection not configured" };
  }

  const result = await authFetch<Mirror>(url, {
    method: "PUT",
    body: JSON.stringify(data),
  });

  if (result.ok) {
    return { success: true, mirror: result.data! };
  }

  if (result.status === 400) {
    return { success: false, error: "invalid_request", message: result.error || "Invalid mirror data" };
  }
  if (result.status === 401) {
    return { success: false, error: "unauthorized", message: "Authentication required" };
  }
  if (result.status === 403) {
    return { success: false, error: "forbidden", message: "Admin access required" };
  }
  if (result.status === 404) {
    return { success: false, error: "not_found", message: "Mirror not found" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to update mirror" };
}

/**
 * Delete a mirror configuration
 */
export async function deleteMirror(id: string): Promise<DeleteResult> {
  const url = getApiUrl(`/mirrors/${id}`);
  if (!url) {
    return { success: false, error: "not_configured", message: "Server connection not configured" };
  }

  const result = await authFetch(url, { method: "DELETE" });

  if (result.ok || result.status === 204) {
    return { success: true };
  }

  if (result.status === 401) {
    return { success: false, error: "unauthorized", message: "Authentication required" };
  }
  if (result.status === 403) {
    return { success: false, error: "forbidden", message: "Admin access required" };
  }
  if (result.status === 404) {
    return { success: false, error: "not_found", message: "Mirror not found" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to delete mirror" };
}
