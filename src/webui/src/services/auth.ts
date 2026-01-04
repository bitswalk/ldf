// Auth service for LDF server communication

import {
  getAuthEndpointUrl,
  getAuthToken,
  getRefreshToken,
  setAuthToken,
  setRefreshToken,
  setTokenExpiresAt,
  clearAllAuth,
} from "./storage";

export interface AuthRequest {
  auth: {
    identity: {
      methods: string[];
      password: {
        user: {
          name: string;
          password: string;
          email?: string;
          role?: string;
        };
      };
    };
  };
}

export interface AuthErrorResponse {
  error: string;
  message: string;
}

export interface UserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
  created_at?: string;
}

export interface AuthSuccessResponse {
  user: UserInfo;
}

export type LoginResult =
  | {
      success: true;
      user: UserInfo;
      token: string;
      refreshToken: string;
      expiresAt: string;
    }
  | {
      success: false;
      error:
        | "user_not_found"
        | "invalid_credentials"
        | "internal_error"
        | "network_error"
        | "not_configured";
      message: string;
    };

export type CreateResult =
  | {
      success: true;
      user: UserInfo;
      token: string;
      refreshToken: string;
      expiresAt: string;
    }
  | {
      success: false;
      error:
        | "email_exists"
        | "user_exists"
        | "root_exists"
        | "invalid_request"
        | "internal_error"
        | "network_error"
        | "not_configured";
      message: string;
    };

export type LogoutResult =
  | { success: true }
  | {
      success: false;
      error:
        | "not_configured"
        | "not_authenticated"
        | "internal_error"
        | "network_error";
      message: string;
    };

export type ValidateResult =
  | { success: true; user: UserInfo }
  | {
      success: false;
      error:
        | "not_configured"
        | "not_authenticated"
        | "token_expired"
        | "token_invalid"
        | "internal_error"
        | "network_error";
      message: string;
    };

export type RefreshResult =
  | {
      success: true;
      user: UserInfo;
      token: string;
      refreshToken: string;
      expiresAt: string;
    }
  | {
      success: false;
      error:
        | "not_configured"
        | "not_authenticated"
        | "refresh_token_invalid"
        | "refresh_token_expired"
        | "internal_error"
        | "network_error";
      message: string;
    };

function buildAuthRequest(
  name: string,
  password: string,
  email?: string,
  role?: string,
): AuthRequest {
  return {
    auth: {
      identity: {
        methods: ["password"],
        password: {
          user: {
            name,
            password,
            ...(email && { email }),
            ...(role && { role }),
          },
        },
      },
    },
  };
}

export async function login(
  username: string,
  password: string,
): Promise<LoginResult> {
  const loginUrl = getAuthEndpointUrl("login");

  if (!loginUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(loginUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(buildAuthRequest(username, password)),
    });

    const token = response.headers.get("X-Subject-Token");

    if (response.ok && token) {
      const data = await response.json();
      return {
        success: true,
        user: data.user,
        token: data.access_token || token,
        refreshToken: data.refresh_token,
        expiresAt: data.expires_at,
      };
    }

    const errorData: AuthErrorResponse = await response.json();

    if (response.status === 401) {
      // Backend returns "unauthorized" for both user not found and invalid password
      // We treat this as "user_not_found" to trigger registration flow
      return {
        success: false,
        error: "user_not_found",
        message: errorData.message,
      };
    }

    if (response.status === 500) {
      return {
        success: false,
        error: "internal_error",
        message: errorData.message,
      };
    }

    return {
      success: false,
      error: "invalid_credentials",
      message: errorData.message,
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

export async function validateToken(): Promise<ValidateResult> {
  const validateUrl = getAuthEndpointUrl("validate");

  if (!validateUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  const token = getAuthToken();

  if (!token) {
    return {
      success: false,
      error: "not_authenticated",
      message: "No active session",
    };
  }

  try {
    const response = await fetch(validateUrl, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    if (response.ok) {
      const data = await response.json();
      return {
        success: true,
        user: data.user,
      };
    }

    const errorData: AuthErrorResponse = await response.json();

    if (response.status === 401) {
      // Check if it's an expired token or invalid token
      if (errorData.error?.includes("token_expired")) {
        return {
          success: false,
          error: "token_expired",
          message: errorData.message,
        };
      }
      return {
        success: false,
        error: "token_invalid",
        message: errorData.message,
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: errorData.message,
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

export async function refreshAccessToken(): Promise<RefreshResult> {
  const refreshUrl = getAuthEndpointUrl("refresh");

  if (!refreshUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  const refreshToken = getRefreshToken();

  if (!refreshToken) {
    return {
      success: false,
      error: "not_authenticated",
      message: "No refresh token available",
    };
  }

  try {
    const response = await fetch(refreshUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (response.ok) {
      const data = await response.json();

      // Update stored tokens
      setAuthToken(data.access_token);
      setRefreshToken(data.refresh_token);
      setTokenExpiresAt(data.expires_at);

      return {
        success: true,
        user: data.user,
        token: data.access_token,
        refreshToken: data.refresh_token,
        expiresAt: data.expires_at,
      };
    }

    const errorData: AuthErrorResponse = await response.json();

    if (response.status === 401) {
      // Check specific error types
      if (errorData.error?.includes("refresh_token_expired")) {
        // Clear all auth on refresh token expiry
        clearAllAuth();
        return {
          success: false,
          error: "refresh_token_expired",
          message: errorData.message,
        };
      }
      if (errorData.error?.includes("refresh_token_revoked")) {
        clearAllAuth();
        return {
          success: false,
          error: "refresh_token_invalid",
          message: errorData.message,
        };
      }
      clearAllAuth();
      return {
        success: false,
        error: "refresh_token_invalid",
        message: errorData.message,
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: errorData.message,
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

// Helper to try refreshing the token if the access token is expired
export async function ensureValidToken(): Promise<boolean> {
  // First try validating the current token
  const validateResult = await validateToken();
  if (validateResult.success) {
    return true;
  }

  // If token is expired, try to refresh
  if (
    validateResult.error === "token_expired" ||
    validateResult.error === "token_invalid"
  ) {
    const refreshResult = await refreshAccessToken();
    return refreshResult.success;
  }

  return false;
}

export async function createUser(
  username: string,
  password: string,
  email: string,
  role: string = "developer",
): Promise<CreateResult> {
  const createUrl = getAuthEndpointUrl("create");

  if (!createUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(createUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(buildAuthRequest(username, password, email, role)),
    });

    const token = response.headers.get("X-Subject-Token");

    if (response.ok && token) {
      const data = await response.json();
      return {
        success: true,
        user: data.user,
        token: data.access_token || token,
        refreshToken: data.refresh_token,
        expiresAt: data.expires_at,
      };
    }

    const errorData: AuthErrorResponse = await response.json();

    if (response.status === 401) {
      // Check specific error messages from backend
      if (errorData.message.includes("Email already exists")) {
        return {
          success: false,
          error: "email_exists",
          message: errorData.message,
        };
      }
      if (errorData.message.includes("Username already exists")) {
        return {
          success: false,
          error: "user_exists",
          message: errorData.message,
        };
      }
      if (errorData.message.includes("Root user already exists")) {
        return {
          success: false,
          error: "root_exists",
          message: errorData.message,
        };
      }
    }

    if (response.status === 400) {
      return {
        success: false,
        error: "invalid_request",
        message: errorData.message,
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: errorData.message,
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

export async function logout(): Promise<LogoutResult> {
  const logoutUrl = getAuthEndpointUrl("logout");

  if (!logoutUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  const token = getAuthToken();

  if (!token) {
    return {
      success: false,
      error: "not_authenticated",
      message: "No active session",
    };
  }

  try {
    const response = await fetch(logoutUrl, {
      method: "POST",
      headers: {
        "X-Subject-Token": token,
      },
    });

    // Backend returns 498 on successful token revocation
    if (response.ok || response.status === 498 || response.status === 204) {
      return { success: true };
    }

    const errorData: AuthErrorResponse = await response.json();

    return {
      success: false,
      error: "internal_error",
      message: errorData.message,
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
