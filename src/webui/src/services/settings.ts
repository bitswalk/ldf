// Settings service for LDF server settings management

import { setDevModeLocal } from "../lib/utils";
import { getServerUrl, getAuthToken, getUserInfo } from "./storage";

export interface ServerSetting {
  key: string;
  value: string | number | boolean;
  type: "string" | "int" | "bool";
  description: string;
  rebootRequired: boolean;
  category: "server" | "log" | "database" | "storage" | "webui";
}

export interface ServerSettingsResponse {
  settings: ServerSetting[];
}

export type GetSettingsResult =
  | { success: true; settings: ServerSetting[] }
  | {
      success: false;
      error:
        | "unauthorized"
        | "forbidden"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type GetSettingResult =
  | { success: true; setting: ServerSetting }
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

export type UpdateSettingResult =
  | {
      success: true;
      key: string;
      value: string | number | boolean;
      rebootRequired: boolean;
      message: string;
    }
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

/**
 * Check if the current user has root access
 */
export function isRootUser(): boolean {
  const userInfo = getUserInfo();
  return userInfo?.role === "root";
}

/**
 * Fetch all server settings (requires root access)
 */
export async function getServerSettings(): Promise<GetSettingsResult> {
  const url = getApiUrl("/settings");

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
      const data: ServerSettingsResponse = await response.json();
      return { success: true, settings: data.settings };
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
        message: "Root access required",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch settings",
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

/**
 * Fetch a single server setting by key (requires root access)
 */
export async function getServerSetting(key: string): Promise<GetSettingResult> {
  const url = getApiUrl(`/settings/${key}`);

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
      const setting: ServerSetting = await response.json();
      return { success: true, setting };
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
        message: "Root access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: `Setting '${key}' not found`,
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch setting",
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

/**
 * Update a server setting (requires root access)
 */
export async function updateServerSetting(
  key: string,
  value: string | number | boolean,
): Promise<UpdateSettingResult> {
  const url = getApiUrl(`/settings/${key}`);

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
      body: JSON.stringify({ value }),
    });

    if (response.ok) {
      const data = await response.json();
      return {
        success: true,
        key: data.key,
        value: data.value,
        rebootRequired: data.rebootRequired,
        message: data.message,
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
        message: "Root access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: `Setting '${key}' not found`,
      };
    }

    if (response.status === 400) {
      const errorData = await response.json().catch(() => ({}));
      return {
        success: false,
        error: "invalid_request",
        message: errorData.message || "Invalid setting value",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to update setting",
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

/**
 * Group settings by category
 */
export function groupSettingsByCategory(
  settings: ServerSetting[],
): Record<string, ServerSetting[]> {
  return settings.reduce(
    (acc, setting) => {
      const category = setting.category;
      if (!acc[category]) {
        acc[category] = [];
      }
      acc[category].push(setting);
      return acc;
    },
    {} as Record<string, ServerSetting[]>,
  );
}

/**
 * Sync devmode setting from server to localStorage
 * This is called on app initialization for root users
 */
export async function syncDevModeFromServer(): Promise<void> {
  if (!isRootUser()) {
    // Non-root users should have devmode disabled
    setDevModeLocal(false);
    return;
  }

  const result = await getServerSetting("webui.devmode");
  if (result.success) {
    const enabled = result.setting.value === true;
    setDevModeLocal(enabled);
  }
}

/**
 * Update devmode setting on server and sync to localStorage
 */
export async function setDevMode(
  enabled: boolean,
): Promise<UpdateSettingResult> {
  const result = await updateServerSetting("webui.devmode", enabled);
  if (result.success) {
    setDevModeLocal(enabled);
  }
  return result;
}

export type ResetDatabaseResult =
  | { success: true; message: string }
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

/**
 * Reset the database to its default state (requires root access)
 * This is a destructive operation that deletes all user data
 */
export async function resetDatabase(
  confirmation: string,
): Promise<ResetDatabaseResult> {
  const url = getApiUrl("/settings/database/reset");

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
      body: JSON.stringify({ confirmation }),
    });

    if (response.ok) {
      const data = await response.json();
      return {
        success: true,
        message: data.message,
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
        message: "Root access required",
      };
    }

    if (response.status === 400) {
      const errorData = await response.json().catch(() => ({}));
      return {
        success: false,
        error: "invalid_request",
        message: errorData.message || "Invalid confirmation",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to reset database",
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
