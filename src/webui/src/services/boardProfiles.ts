// Board profiles service for LDF server communication

import { authFetch } from "./api";
import { getServerUrl } from "./storage";

export interface BoardConfig {
  device_trees?: DeviceTreeSpec[];
  kernel_overlay?: Record<string, string>;
  kernel_defconfig?: string;
  boot_params?: BoardBootParams;
  firmware?: BoardFirmware[];
  kernel_cmdline?: string;
}

export interface DeviceTreeSpec {
  source: string;
  overlays?: string[];
}

export interface BoardBootParams {
  bootloader_override?: string;
  uboot_board?: string;
  extra_files?: Record<string, string>;
  config_txt?: string;
}

export interface BoardFirmware {
  name: string;
  component_id?: string;
  path?: string;
  description?: string;
}

export interface BoardProfile {
  id: string;
  name: string;
  display_name: string;
  description: string;
  arch: "x86_64" | "aarch64";
  config: BoardConfig;
  is_system: boolean;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateBoardProfileRequest {
  name: string;
  display_name: string;
  description?: string;
  arch: "x86_64" | "aarch64";
  config: BoardConfig;
}

export interface UpdateBoardProfileRequest {
  name?: string;
  display_name?: string;
  description?: string;
  config?: BoardConfig;
}

interface BoardProfileListResponse {
  count: number;
  profiles: BoardProfile[];
}

export type ListResult =
  | { success: true; profiles: BoardProfile[] }
  | {
      success: false;
      error: "unauthorized" | "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export type GetResult =
  | { success: true; profile: BoardProfile }
  | {
      success: false;
      error: "not_found" | "unauthorized" | "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export type CreateResult =
  | { success: true; profile: BoardProfile }
  | {
      success: false;
      error: "unauthorized" | "forbidden" | "invalid_request" | "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export type UpdateResult =
  | { success: true; profile: BoardProfile }
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

function getApiUrl(path: string): string | null {
  const serverUrl = getServerUrl();
  if (!serverUrl) return null;
  return `${serverUrl}/v1${path}`;
}

/**
 * List all board profiles, optionally filtered by architecture
 */
export async function listBoardProfiles(arch?: string): Promise<ListResult> {
  let path = "/board/profiles";
  if (arch) {
    path += `?arch=${encodeURIComponent(arch)}`;
  }

  const url = getApiUrl(path);
  if (!url) {
    return { success: false, error: "not_configured", message: "Server connection not configured" };
  }

  const result = await authFetch<BoardProfileListResponse>(url);
  if (result.ok) {
    return { success: true, profiles: result.data!.profiles };
  }

  if (result.status === 401) {
    return { success: false, error: "unauthorized", message: "Authentication required" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to fetch board profiles" };
}

/**
 * Get a single board profile by ID
 */
export async function getBoardProfile(id: string): Promise<GetResult> {
  const url = getApiUrl(`/board/profiles/${id}`);
  if (!url) {
    return { success: false, error: "not_configured", message: "Server connection not configured" };
  }

  const result = await authFetch<BoardProfile>(url);
  if (result.ok) {
    return { success: true, profile: result.data! };
  }

  if (result.status === 404) {
    return { success: false, error: "not_found", message: "Board profile not found" };
  }
  if (result.status === 401) {
    return { success: false, error: "unauthorized", message: "Authentication required" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to fetch board profile" };
}

/**
 * Create a new board profile
 */
export async function createBoardProfile(data: CreateBoardProfileRequest): Promise<CreateResult> {
  const url = getApiUrl("/board/profiles");
  if (!url) {
    return { success: false, error: "not_configured", message: "Server connection not configured" };
  }

  const result = await authFetch<BoardProfile>(url, {
    method: "POST",
    body: JSON.stringify(data),
  });

  if (result.ok) {
    return { success: true, profile: result.data! };
  }

  if (result.status === 400) {
    return { success: false, error: "invalid_request", message: result.error || "Invalid board profile data" };
  }
  if (result.status === 401) {
    return { success: false, error: "unauthorized", message: "Authentication required" };
  }
  if (result.status === 403) {
    return { success: false, error: "forbidden", message: "Insufficient permissions" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to create board profile" };
}

/**
 * Update an existing board profile
 */
export async function updateBoardProfile(id: string, data: UpdateBoardProfileRequest): Promise<UpdateResult> {
  const url = getApiUrl(`/board/profiles/${id}`);
  if (!url) {
    return { success: false, error: "not_configured", message: "Server connection not configured" };
  }

  const result = await authFetch<BoardProfile>(url, {
    method: "PUT",
    body: JSON.stringify(data),
  });

  if (result.ok) {
    return { success: true, profile: result.data! };
  }

  if (result.status === 400) {
    return { success: false, error: "invalid_request", message: result.error || "Invalid board profile data" };
  }
  if (result.status === 401) {
    return { success: false, error: "unauthorized", message: "Authentication required" };
  }
  if (result.status === 403) {
    return { success: false, error: "forbidden", message: "Insufficient permissions" };
  }
  if (result.status === 404) {
    return { success: false, error: "not_found", message: "Board profile not found" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to update board profile" };
}

/**
 * Delete a board profile
 */
export async function deleteBoardProfile(id: string): Promise<DeleteResult> {
  const url = getApiUrl(`/board/profiles/${id}`);
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
    return { success: false, error: "forbidden", message: result.error || "Insufficient permissions" };
  }
  if (result.status === 404) {
    return { success: false, error: "not_found", message: "Board profile not found" };
  }

  return { success: false, error: "internal_error", message: result.error || "Failed to delete board profile" };
}
