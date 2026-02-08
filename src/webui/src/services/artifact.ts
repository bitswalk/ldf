import { authFetch, getApiUrl } from "./api";
// Artifact service for LDF server communication

import { debugError } from "../lib/utils";
import { getAuthToken } from "./storage";

export interface Artifact {
  key: string;
  full_key: string;
  size: number;
  content_type?: string;
  etag?: string;
  last_modified: string;
  distribution_id: string;
  distribution_name: string;
  owner_id?: string;
}

export interface ArtifactListResponse {
  count: number;
  artifacts: Artifact[];
}

export type ListResult =
  | { success: true; artifacts: Artifact[] }
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

export interface ArtifactURLResponse {
  url: string;
  web_url?: string;
  expires_at: string;
}

export type GetURLResult =
  | { success: true; url: string; webUrl?: string; expiresAt: string }
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

export async function listArtifacts(): Promise<ListResult> {
  const url = getApiUrl("/artifacts");

  if (!url) {
    const error = "Server connection not configured";
    debugError("[ArtifactService] listArtifacts:", error);
    return {
      success: false,
      error: "not_configured",
      message: error,
    };
  }

  try {
    const result = await authFetch(url);

    if (result.ok) {
      const data = result.data as any;
      return { success: true, artifacts: data.artifacts || [] };
    }

    // Try to get error details from response
    let errorMessage = "";
    try {
      const errorData = result.data as any;
      errorMessage = errorData.message || errorData.error || "";
    } catch {
      // Response wasn't JSON
    }

    if (result.status === 401) {
      const msg = errorMessage || "Authentication required";
      debugError("[ArtifactService] listArtifacts: 401 -", msg);
      return {
        success: false,
        error: "unauthorized",
        message: msg,
      };
    }

    if (result.status === 503) {
      const msg = errorMessage || "Storage service not configured";
      debugError("[ArtifactService] listArtifacts: 503 -", msg);
      return {
        success: false,
        error: "service_unavailable",
        message: msg,
      };
    }

    if (result.status === 500) {
      const msg = errorMessage || "Failed to reach storage backend";
      debugError("[ArtifactService] listArtifacts: 500 -", msg);
      return {
        success: false,
        error: "internal_error",
        message: msg,
      };
    }

    const msg = errorMessage || `Server error (${result.status})`;
    debugError("[ArtifactService] listArtifacts:", result.status, "-", msg);
    return {
      success: false,
      error: "internal_error",
      message: msg,
    };
  } catch (err) {
    const msg =
      err instanceof Error ? err.message : "Failed to connect to server";
    debugError("[ArtifactService] listArtifacts: Network error -", msg);
    return {
      success: false,
      error: "network_error",
      message: msg,
    };
  }
}

export async function deleteArtifact(
  distributionId: string,
  artifactPath: string,
): Promise<DeleteResult> {
  const url = getApiUrl(
    `/distributions/${distributionId}/artifacts/${artifactPath}`,
  );

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const result = await authFetch(url, { method: "DELETE" });

    if (response.ok || result.status === 204) {
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
      return {
        success: false,
        error: "not_found",
        message: "Artifact not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to delete artifact",
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

export async function getArtifactURL(
  distributionId: string,
  artifactPath: string,
  expiry?: number,
): Promise<GetURLResult> {
  let url = getApiUrl(
    `/distributions/${distributionId}/artifacts-url/${artifactPath}`,
  );

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  if (expiry) {
    url += `?expiry=${expiry}`;
  }

  try {
    const result = await authFetch(url);

    if (result.ok) {
      const data = result.data as any;
      return {
        success: true,
        url: data.url,
        webUrl: data.web_url,
        expiresAt: data.expires_at,
      };
    }

    if (result.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (result.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Artifact not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to get artifact URL",
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

export interface UploadArtifactResponse {
  key: string;
  size: number;
  message: string;
}

export type UploadResult =
  | { success: true; key: string; size: number }
  | {
      success: false;
      error:
        | "unauthorized"
        | "not_found"
        | "network_error"
        | "not_configured"
        | "service_unavailable"
        | "internal_error";
      message: string;
    };

export async function uploadArtifact(
  distributionId: string,
  file: File,
  path?: string,
  onProgress?: (progress: number) => void,
): Promise<UploadResult> {
  const url = getApiUrl(`/distributions/${distributionId}/artifacts`);

  if (!url) {
    const error = "Server connection not configured";
    debugError("[ArtifactService] uploadArtifact:", error);
    return {
      success: false,
      error: "not_configured",
      message: error,
    };
  }

  // Helper to parse error from XHR response
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
    if (path) {
      formData.append("path", path);
    }

    // Build headers without Content-Type (browser sets it with boundary for FormData)
    const headers: Record<string, string> = {};
    const token = getAuthToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

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
          if (xhr.status === 201 || xhr.status === 200) {
            const data: UploadArtifactResponse = JSON.parse(xhr.responseText);
            resolve({ success: true, key: data.key, size: data.size });
          } else if (xhr.status === 401) {
            const msg = parseXHRError(xhr) || "Authentication required";
            debugError("[ArtifactService] uploadArtifact: 401 -", msg);
            resolve({
              success: false,
              error: "unauthorized",
              message: msg,
            });
          } else if (xhr.status === 404) {
            const msg = parseXHRError(xhr) || "Distribution not found";
            debugError("[ArtifactService] uploadArtifact: 404 -", msg);
            resolve({
              success: false,
              error: "not_found",
              message: msg,
            });
          } else if (xhr.status === 503) {
            const msg = parseXHRError(xhr) || "Storage service not configured";
            debugError("[ArtifactService] uploadArtifact: 503 -", msg);
            resolve({
              success: false,
              error: "service_unavailable",
              message: msg,
            });
          } else if (xhr.status === 500) {
            const msg = parseXHRError(xhr) || "Failed to reach storage backend";
            debugError("[ArtifactService] uploadArtifact: 500 -", msg);
            resolve({
              success: false,
              error: "internal_error",
              message: msg,
            });
          } else {
            const msg = parseXHRError(xhr) || `Server error (${xhr.status})`;
            debugError(
              "[ArtifactService] uploadArtifact:",
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
          debugError("[ArtifactService] uploadArtifact: Network error -", msg);
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
    const response = await fetch(url, {
      method: "POST",
      headers,
      body: formData,
    });

    if (response.ok) {
      const data = await response.json();
      return { success: true, key: data.key, size: data.size };
    }

    // Try to get error details from response
    let errorMessage = "";
    try {
      const errorData = await response.json();
      errorMessage = errorData.message || errorData.error || "";
    } catch {
      // Response wasn't JSON
    }

    if (response.status === 401) {
      const msg = errorMessage || "Authentication required";
      debugError("[ArtifactService] uploadArtifact: 401 -", msg);
      return {
        success: false,
        error: "unauthorized",
        message: msg,
      };
    }

    if (response.status === 404) {
      const msg = errorMessage || "Distribution not found";
      debugError("[ArtifactService] uploadArtifact: 404 -", msg);
      return {
        success: false,
        error: "not_found",
        message: msg,
      };
    }

    if (response.status === 503) {
      const msg = errorMessage || "Storage service not configured";
      debugError("[ArtifactService] uploadArtifact: 503 -", msg);
      return {
        success: false,
        error: "service_unavailable",
        message: msg,
      };
    }

    if (response.status === 500) {
      const msg = errorMessage || "Failed to reach storage backend";
      debugError("[ArtifactService] uploadArtifact: 500 -", msg);
      return {
        success: false,
        error: "internal_error",
        message: msg,
      };
    }

    const msg = errorMessage || `Server error (${response.status})`;
    debugError("[ArtifactService] uploadArtifact:", response.status, "-", msg);
    return {
      success: false,
      error: "internal_error",
      message: msg,
    };
  } catch (err) {
    const msg =
      err instanceof Error ? err.message : "Failed to connect to server";
    debugError("[ArtifactService] uploadArtifact: Network error -", msg);
    return {
      success: false,
      error: "network_error",
      message: msg,
    };
  }
}

// Helper function to format file size
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}
