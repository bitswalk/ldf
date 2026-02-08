// Branding service for LDF server communication

import { debugError } from "../lib/utils";
import { getAuthToken, getServerUrl } from "./storage";
import { authFetch, getApiUrl } from "./api";
import { getServerSetting, updateServerSetting } from "./settings";

export type BrandingAsset = "logo" | "favicon";

// Default app name fallback
export const DEFAULT_APP_NAME = "Linux Distribution Factory";
export const APP_NAME_MAX_LENGTH = 32;

export interface BrandingAssetInfo {
  asset: string;
  url: string;
  content_type: string;
  size: number;
  exists: boolean;
}

export type GetInfoResult =
  | { success: true; info: BrandingAssetInfo }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "service_unavailable"
        | "internal_error";
      message: string;
    };

export type UploadResult =
  | { success: true; url: string; contentType: string; size: number }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "service_unavailable"
        | "bad_request"
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

/**
 * Get the direct URL for a branding asset (for use in img src, etc.)
 */
export function getBrandingAssetURL(asset: BrandingAsset): string | null {
  const serverUrl = getServerUrl();
  if (!serverUrl) return null;
  return `${serverUrl}/v1/branding/${asset}`;
}

/**
 * Get metadata about a branding asset
 */
export async function getBrandingAssetInfo(
  asset: BrandingAsset,
): Promise<GetInfoResult> {
  const url = getApiUrl(`/branding/${asset}/info`);

  if (!url) {
    const error = "Server connection not configured";
    debugError("[BrandingService] getBrandingAssetInfo:", error);
    return {
      success: false,
      error: "not_configured",
      message: error,
    };
  }

  const result = await authFetch<BrandingAssetInfo>(url);

  if (result.ok) {
    return { success: true, info: result.data! };
  }

  const errorMessage = result.message || result.error || "";

  if (result.status === 401) {
    const msg = errorMessage || "Authentication required";
    debugError("[BrandingService] getBrandingAssetInfo: 401 -", msg);
    return { success: false, error: "unauthorized", message: msg };
  }

  if (result.status === 503) {
    const msg = errorMessage || "Storage service not configured";
    debugError("[BrandingService] getBrandingAssetInfo: 503 -", msg);
    return { success: false, error: "service_unavailable", message: msg };
  }

  const msg = errorMessage || `Server error (${result.status})`;
  debugError(
    "[BrandingService] getBrandingAssetInfo:",
    result.status,
    "-",
    msg,
  );
  return { success: false, error: "internal_error", message: msg };
}

/**
 * Upload a branding asset (logo or favicon)
 */
export async function uploadBrandingAsset(
  asset: BrandingAsset,
  file: File,
  onProgress?: (progress: number) => void,
): Promise<UploadResult> {
  const url = getApiUrl(`/branding/${asset}`);

  if (!url) {
    const error = "Server connection not configured";
    debugError("[BrandingService] uploadBrandingAsset:", error);
    return {
      success: false,
      error: "not_configured",
      message: error,
    };
  }

  const parseXHRError = (xhr: XMLHttpRequest): string => {
    try {
      const data = JSON.parse(xhr.responseText);
      return data.message || data.error || "";
    } catch {
      return "";
    }
  };

  try {
    const formData = new FormData();
    formData.append("file", file);

    const token = getAuthToken();

    // Use XMLHttpRequest for progress tracking if callback provided
    if (onProgress) {
      return new Promise((resolve) => {
        const xhr = new XMLHttpRequest();

        xhr.upload.addEventListener("progress", (event) => {
          if (event.lengthComputable) {
            const progress = Math.round((event.loaded / event.total) * 100);
            onProgress(progress);
          }
        });

        xhr.addEventListener("load", () => {
          if (xhr.status === 200 || xhr.status === 201) {
            const data = JSON.parse(xhr.responseText);
            resolve({
              success: true,
              url: data.url,
              contentType: data.content_type,
              size: data.size,
            });
          } else if (xhr.status === 400) {
            const msg = parseXHRError(xhr) || "Invalid file type";
            debugError("[BrandingService] uploadBrandingAsset: 400 -", msg);
            resolve({
              success: false,
              error: "bad_request",
              message: msg,
            });
          } else if (xhr.status === 401) {
            const msg = parseXHRError(xhr) || "Authentication required";
            debugError("[BrandingService] uploadBrandingAsset: 401 -", msg);
            resolve({
              success: false,
              error: "unauthorized",
              message: msg,
            });
          } else if (xhr.status === 503) {
            const msg = parseXHRError(xhr) || "Storage service not configured";
            debugError("[BrandingService] uploadBrandingAsset: 503 -", msg);
            resolve({
              success: false,
              error: "service_unavailable",
              message: msg,
            });
          } else {
            const msg = parseXHRError(xhr) || `Server error (${xhr.status})`;
            debugError(
              "[BrandingService] uploadBrandingAsset:",
              xhr.status,
              "-",
              msg,
            );
            resolve({
              success: false,
              error: "internal_error",
              message: msg,
            });
          }
        });

        xhr.addEventListener("error", () => {
          const msg = "Failed to connect to server";
          debugError(
            "[BrandingService] uploadBrandingAsset: Network error -",
            msg,
          );
          resolve({
            success: false,
            error: "network_error",
            message: msg,
          });
        });

        xhr.open("POST", url);
        if (token) {
          xhr.setRequestHeader("Authorization", `Bearer ${token}`);
        }
        xhr.send(formData);
      });
    }

    // Simple fetch without progress
    const headers: Record<string, string> = {};
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      method: "POST",
      headers,
      body: formData,
    });

    if (response.ok) {
      const data = await response.json();
      return {
        success: true,
        url: data.url,
        contentType: data.content_type,
        size: data.size,
      };
    }

    let errorMessage = "";
    try {
      const errorData = await response.json();
      errorMessage = errorData.message || errorData.error || "";
    } catch {
      // Response wasn't JSON
    }

    if (response.status === 400) {
      const msg = errorMessage || "Invalid file type";
      debugError("[BrandingService] uploadBrandingAsset: 400 -", msg);
      return {
        success: false,
        error: "bad_request",
        message: msg,
      };
    }

    if (response.status === 401) {
      const msg = errorMessage || "Authentication required";
      debugError("[BrandingService] uploadBrandingAsset: 401 -", msg);
      return {
        success: false,
        error: "unauthorized",
        message: msg,
      };
    }

    if (response.status === 503) {
      const msg = errorMessage || "Storage service not configured";
      debugError("[BrandingService] uploadBrandingAsset: 503 -", msg);
      return {
        success: false,
        error: "service_unavailable",
        message: msg,
      };
    }

    const msg = errorMessage || `Server error (${response.status})`;
    debugError(
      "[BrandingService] uploadBrandingAsset:",
      response.status,
      "-",
      msg,
    );
    return {
      success: false,
      error: "internal_error",
      message: msg,
    };
  } catch (err) {
    const msg =
      err instanceof Error ? err.message : "Failed to connect to server";
    debugError("[BrandingService] uploadBrandingAsset: Network error -", msg);
    return {
      success: false,
      error: "network_error",
      message: msg,
    };
  }
}

/**
 * Initialize the favicon from the server if one has been uploaded.
 * Call this on app startup.
 */
export async function initializeFavicon(): Promise<void> {
  const result = await getBrandingAssetInfo("favicon");
  if (result.success && result.info.exists) {
    const url = getBrandingAssetURL("favicon");
    if (url) {
      updateFavicon(url);
    }
  }
}

/**
 * Update the favicon in the document head
 */
export function updateFavicon(url: string): void {
  const faviconLink = document.getElementById(
    "app-favicon",
  ) as HTMLLinkElement | null;
  if (faviconLink) {
    faviconLink.href = url + "?t=" + Date.now();
  } else {
    // Create the link element if it doesn't exist
    const link = document.createElement("link");
    link.id = "app-favicon";
    link.rel = "icon";
    link.href = url + "?t=" + Date.now();
    document.head.appendChild(link);
  }
}

/**
 * Delete a branding asset
 */
export async function deleteBrandingAsset(
  asset: BrandingAsset,
): Promise<DeleteResult> {
  const url = getApiUrl(`/branding/${asset}`);

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

  if (result.status === 404) {
    return { success: false, error: "not_found", message: "Asset not found" };
  }

  return {
    success: false,
    error: "internal_error",
    message: "Failed to delete asset",
  };
}

export type GetAppNameResult =
  | { success: true; appName: string }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type SetAppNameResult =
  | { success: true; appName: string }
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
 * Get the custom application name from server settings
 * Returns the default app name if not configured or on error
 */
export async function getAppName(): Promise<GetAppNameResult> {
  const result = await getServerSetting("webui.app_name");

  if (result.success) {
    const appName = result.setting.value as string;
    return {
      success: true,
      appName: appName && appName.trim() ? appName.trim() : DEFAULT_APP_NAME,
    };
  }

  // For 404 (not found) or forbidden errors, just return default
  if (result.error === "not_found" || result.error === "forbidden") {
    return { success: true, appName: DEFAULT_APP_NAME };
  }

  return {
    success: false,
    error:
      result.error === "unauthorized"
        ? "unauthorized"
        : result.error === "network_error"
          ? "network_error"
          : result.error === "not_configured"
            ? "not_configured"
            : "internal_error",
    message: result.message,
  };
}

/**
 * Set a custom application name (requires root access)
 * Pass empty string to reset to default
 */
export async function setAppName(name: string): Promise<SetAppNameResult> {
  const trimmedName = name.trim();

  if (trimmedName.length > APP_NAME_MAX_LENGTH) {
    return {
      success: false,
      error: "invalid_request",
      message: `Application name must be ${APP_NAME_MAX_LENGTH} characters or less`,
    };
  }

  const result = await updateServerSetting("webui.app_name", trimmedName);

  if (result.success) {
    return {
      success: true,
      appName: trimmedName || DEFAULT_APP_NAME,
    };
  }

  return {
    success: false,
    error:
      result.error === "unauthorized"
        ? "unauthorized"
        : result.error === "forbidden"
          ? "forbidden"
          : result.error === "invalid_request"
            ? "invalid_request"
            : result.error === "network_error"
              ? "network_error"
              : result.error === "not_configured"
                ? "not_configured"
                : "internal_error",
    message: result.message,
  };
}

/**
 * Initialize the document title with the custom app name.
 * Call this on app startup.
 */
export async function initializeAppName(): Promise<string> {
  const result = await getAppName();
  const appName = result.success ? result.appName : DEFAULT_APP_NAME;
  document.title = appName;
  return appName;
}

/**
 * Update the document title with the given app name
 */
export function updateDocumentTitle(appName: string): void {
  document.title = appName || DEFAULT_APP_NAME;
}
