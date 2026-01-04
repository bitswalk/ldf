// Centralized API client with automatic token refresh handling

import {
  getAuthToken,
  getRefreshToken,
  setAuthToken,
  setRefreshToken,
  setTokenExpiresAt,
  clearAllAuth,
  isTokenExpiringSoon,
} from "./storage";
import { refreshAccessToken } from "./auth";

// Flag to prevent multiple simultaneous refresh attempts
let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

// Callbacks to notify when auth state changes
type AuthChangeCallback = (isAuthenticated: boolean) => void;
const authChangeCallbacks: AuthChangeCallback[] = [];

export function onAuthChange(callback: AuthChangeCallback): () => void {
  authChangeCallbacks.push(callback);
  return () => {
    const index = authChangeCallbacks.indexOf(callback);
    if (index > -1) {
      authChangeCallbacks.splice(index, 1);
    }
  };
}

function notifyAuthChange(isAuthenticated: boolean): void {
  authChangeCallbacks.forEach((cb) => cb(isAuthenticated));
}

// Try to refresh the token if needed
async function tryRefreshToken(): Promise<boolean> {
  // If already refreshing, wait for the existing refresh to complete
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }

  const refreshToken = getRefreshToken();
  if (!refreshToken) {
    return false;
  }

  isRefreshing = true;
  refreshPromise = (async () => {
    try {
      const result = await refreshAccessToken();
      if (result.success) {
        return true;
      }
      // Refresh failed - clear auth and notify
      clearAllAuth();
      notifyAuthChange(false);
      return false;
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

// Proactively refresh token if expiring soon
async function ensureFreshToken(): Promise<void> {
  if (isTokenExpiringSoon()) {
    await tryRefreshToken();
  }
}

export interface FetchOptions extends RequestInit {
  skipAuth?: boolean;
  skipRetry?: boolean;
}

export interface ApiResponse<T> {
  ok: boolean;
  status: number;
  data?: T;
  error?: string;
  message?: string;
}

/**
 * Authenticated fetch wrapper that handles token refresh automatically.
 *
 * - Adds Authorization header with Bearer token
 * - Proactively refreshes token if expiring soon
 * - On 401, attempts to refresh token and retry the request
 * - Notifies listeners when auth fails completely
 */
export async function authFetch<T = unknown>(
  url: string,
  options: FetchOptions = {}
): Promise<ApiResponse<T>> {
  const { skipAuth = false, skipRetry = false, ...fetchOptions } = options;

  // Proactively refresh if token is expiring soon
  if (!skipAuth) {
    await ensureFreshToken();
  }

  // Build headers with auth token
  const headers = new Headers(fetchOptions.headers);

  if (!skipAuth) {
    const token = getAuthToken();
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }
  }

  if (!headers.has("Content-Type") && fetchOptions.body) {
    headers.set("Content-Type", "application/json");
  }

  try {
    const response = await fetch(url, {
      ...fetchOptions,
      headers,
    });

    // Handle 401 Unauthorized
    if (response.status === 401 && !skipAuth && !skipRetry) {
      // Try to refresh the token
      const refreshed = await tryRefreshToken();

      if (refreshed) {
        // Retry the request with new token
        const newToken = getAuthToken();
        if (newToken) {
          headers.set("Authorization", `Bearer ${newToken}`);
        }

        const retryResponse = await fetch(url, {
          ...fetchOptions,
          headers,
        });

        if (retryResponse.ok) {
          const data = await retryResponse.json().catch(() => undefined);
          return { ok: true, status: retryResponse.status, data };
        }

        // Still failed after refresh
        if (retryResponse.status === 401) {
          clearAllAuth();
          notifyAuthChange(false);
        }

        const errorData = await retryResponse.json().catch(() => ({}));
        return {
          ok: false,
          status: retryResponse.status,
          error: errorData.error,
          message: errorData.message,
        };
      }

      // Refresh failed - auth is invalid
      const errorData = await response.json().catch(() => ({}));
      return {
        ok: false,
        status: 401,
        error: errorData.error || "unauthorized",
        message: errorData.message || "Authentication required",
      };
    }

    // Handle other responses
    if (response.ok) {
      const data = await response.json().catch(() => undefined);
      return { ok: true, status: response.status, data };
    }

    const errorData = await response.json().catch(() => ({}));
    return {
      ok: false,
      status: response.status,
      error: errorData.error,
      message: errorData.message,
    };
  } catch (err) {
    return {
      ok: false,
      status: 0,
      error: "network_error",
      message: err instanceof Error ? err.message : "Network error",
    };
  }
}

/**
 * Helper to build authenticated headers for services that need direct fetch access
 */
export function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = getAuthToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  return headers;
}
