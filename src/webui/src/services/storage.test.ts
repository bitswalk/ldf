import { describe, it, expect, beforeEach } from "vitest";
import {
  getServerUrl,
  setServerUrl,
  clearServerUrl,
  getAPIEndpoints,
  setAPIEndpoints,
  clearAPIEndpoints,
  getAuthToken,
  setAuthToken,
  clearAuthToken,
  getRefreshToken,
  setRefreshToken,
  clearRefreshToken,
  getTokenExpiresAt,
  setTokenExpiresAt,
  clearTokenExpiresAt,
  isTokenExpired,
  isTokenExpiringSoon,
  getUserInfo,
  setUserInfo,
  clearUserInfo,
  clearAllAuth,
  clearAll,
  hasServerConnection,
  hasCompleteServerConnection,
  hasAuthSession,
  getAuthEndpointUrl,
} from "./storage";
import type { APIEndpoints, StoredUserInfo } from "./storage";

const mockEndpoints: APIEndpoints = {
  health: "/health",
  version: "/version",
  api_v1: "/v1",
  auth: {
    create: "/v1/auth/create",
    login: "/v1/auth/login",
    logout: "/v1/auth/logout",
    refresh: "/v1/auth/refresh",
    validate: "/v1/auth/validate",
  },
};

const mockUser: StoredUserInfo = {
  id: "user-1",
  name: "testuser",
  email: "test@example.com",
  role: "developer",
};

beforeEach(() => {
  localStorage.clear();
});

describe("Server URL", () => {
  it("returns null when no URL is set", () => {
    expect(getServerUrl()).toBeNull();
  });

  it("stores and retrieves the server URL", () => {
    setServerUrl("https://example.com");
    expect(getServerUrl()).toBe("https://example.com");
  });

  it("clears the server URL", () => {
    setServerUrl("https://example.com");
    clearServerUrl();
    expect(getServerUrl()).toBeNull();
  });
});

describe("API Endpoints", () => {
  it("returns null when no endpoints are set", () => {
    expect(getAPIEndpoints()).toBeNull();
  });

  it("stores and retrieves API endpoints", () => {
    setAPIEndpoints(mockEndpoints);
    expect(getAPIEndpoints()).toEqual(mockEndpoints);
  });

  it("returns null for invalid JSON", () => {
    localStorage.setItem("ldf_api_endpoints", "invalid-json");
    expect(getAPIEndpoints()).toBeNull();
  });

  it("clears endpoints", () => {
    setAPIEndpoints(mockEndpoints);
    clearAPIEndpoints();
    expect(getAPIEndpoints()).toBeNull();
  });
});

describe("Auth Token", () => {
  it("returns null when no token is set", () => {
    expect(getAuthToken()).toBeNull();
  });

  it("stores and retrieves the auth token", () => {
    setAuthToken("test-token-123");
    expect(getAuthToken()).toBe("test-token-123");
  });

  it("clears the auth token", () => {
    setAuthToken("test-token-123");
    clearAuthToken();
    expect(getAuthToken()).toBeNull();
  });
});

describe("Refresh Token", () => {
  it("returns null when no token is set", () => {
    expect(getRefreshToken()).toBeNull();
  });

  it("stores and retrieves the refresh token", () => {
    setRefreshToken("refresh-token-123");
    expect(getRefreshToken()).toBe("refresh-token-123");
  });

  it("clears the refresh token", () => {
    setRefreshToken("refresh-token-123");
    clearRefreshToken();
    expect(getRefreshToken()).toBeNull();
  });
});

describe("Token Expiry", () => {
  it("returns null when no expiry is set", () => {
    expect(getTokenExpiresAt()).toBeNull();
  });

  it("stores and retrieves expiry as Date object", () => {
    const date = new Date("2026-01-01T00:00:00Z");
    setTokenExpiresAt(date);
    expect(getTokenExpiresAt()).toEqual(date);
  });

  it("stores and retrieves expiry as string", () => {
    setTokenExpiresAt("2026-01-01T00:00:00Z");
    expect(getTokenExpiresAt()).toEqual(new Date("2026-01-01T00:00:00Z"));
  });

  it("returns null for invalid date", () => {
    localStorage.setItem("ldf_token_expires_at", "not-a-date");
    expect(getTokenExpiresAt()).toBeNull();
  });

  it("clears token expiry", () => {
    setTokenExpiresAt(new Date());
    clearTokenExpiresAt();
    expect(getTokenExpiresAt()).toBeNull();
  });
});

describe("isTokenExpired", () => {
  it("returns true when no expiry is set", () => {
    expect(isTokenExpired()).toBe(true);
  });

  it("returns true when token has expired", () => {
    const pastDate = new Date(Date.now() - 60_000);
    setTokenExpiresAt(pastDate);
    expect(isTokenExpired()).toBe(true);
  });

  it("returns false when token is still valid", () => {
    const futureDate = new Date(Date.now() + 3_600_000);
    setTokenExpiresAt(futureDate);
    expect(isTokenExpired()).toBe(false);
  });

  it("returns true when within 30-second buffer", () => {
    const nearDate = new Date(Date.now() + 20_000);
    setTokenExpiresAt(nearDate);
    expect(isTokenExpired()).toBe(true);
  });
});

describe("isTokenExpiringSoon", () => {
  it("returns true when no expiry is set", () => {
    expect(isTokenExpiringSoon()).toBe(true);
  });

  it("returns true when within 2-minute window", () => {
    const nearDate = new Date(Date.now() + 60_000);
    setTokenExpiresAt(nearDate);
    expect(isTokenExpiringSoon()).toBe(true);
  });

  it("returns false when well beyond 2-minute window", () => {
    const futureDate = new Date(Date.now() + 3_600_000);
    setTokenExpiresAt(futureDate);
    expect(isTokenExpiringSoon()).toBe(false);
  });
});

describe("User Info", () => {
  it("returns null when no user info is set", () => {
    expect(getUserInfo()).toBeNull();
  });

  it("stores and retrieves user info", () => {
    setUserInfo(mockUser);
    expect(getUserInfo()).toEqual(mockUser);
  });

  it("returns null for invalid JSON", () => {
    localStorage.setItem("ldf_user_info", "invalid");
    expect(getUserInfo()).toBeNull();
  });

  it("clears user info", () => {
    setUserInfo(mockUser);
    clearUserInfo();
    expect(getUserInfo()).toBeNull();
  });
});

describe("clearAllAuth", () => {
  it("clears token, refresh token, expiry, and user info", () => {
    setAuthToken("token");
    setRefreshToken("refresh");
    setTokenExpiresAt(new Date());
    setUserInfo(mockUser);

    clearAllAuth();

    expect(getAuthToken()).toBeNull();
    expect(getRefreshToken()).toBeNull();
    expect(getTokenExpiresAt()).toBeNull();
    expect(getUserInfo()).toBeNull();
  });

  it("does not clear server URL or endpoints", () => {
    setServerUrl("https://example.com");
    setAuthToken("token");
    clearAllAuth();
    expect(getServerUrl()).toBe("https://example.com");
  });
});

describe("clearAll", () => {
  it("clears everything including server URL and endpoints", () => {
    setServerUrl("https://example.com");
    setAPIEndpoints(mockEndpoints);
    setAuthToken("token");
    setRefreshToken("refresh");

    clearAll();

    expect(getServerUrl()).toBeNull();
    expect(getAPIEndpoints()).toBeNull();
    expect(getAuthToken()).toBeNull();
    expect(getRefreshToken()).toBeNull();
  });
});

describe("hasServerConnection", () => {
  it("returns false when no URL is set", () => {
    expect(hasServerConnection()).toBe(false);
  });

  it("returns true when URL is set", () => {
    setServerUrl("https://example.com");
    expect(hasServerConnection()).toBe(true);
  });
});

describe("hasCompleteServerConnection", () => {
  it("returns false when neither URL nor endpoints are set", () => {
    expect(hasCompleteServerConnection()).toBe(false);
  });

  it("returns false when only URL is set", () => {
    setServerUrl("https://example.com");
    expect(hasCompleteServerConnection()).toBe(false);
  });

  it("returns true when both URL and endpoints are set", () => {
    setServerUrl("https://example.com");
    setAPIEndpoints(mockEndpoints);
    expect(hasCompleteServerConnection()).toBe(true);
  });
});

describe("hasAuthSession", () => {
  it("returns false with no auth data", () => {
    expect(hasAuthSession()).toBe(false);
  });

  it("returns false with only token", () => {
    setAuthToken("token");
    expect(hasAuthSession()).toBe(false);
  });

  it("returns false with only user info", () => {
    setUserInfo(mockUser);
    expect(hasAuthSession()).toBe(false);
  });

  it("returns true with both token and user info", () => {
    setAuthToken("token");
    setUserInfo(mockUser);
    expect(hasAuthSession()).toBe(true);
  });
});

describe("getAuthEndpointUrl", () => {
  it("returns null when server URL is not set", () => {
    expect(getAuthEndpointUrl("login")).toBeNull();
  });

  it("returns null when endpoints are not set", () => {
    setServerUrl("https://example.com");
    expect(getAuthEndpointUrl("login")).toBeNull();
  });

  it("constructs correct URL for each auth endpoint", () => {
    setServerUrl("https://example.com");
    setAPIEndpoints(mockEndpoints);
    expect(getAuthEndpointUrl("login")).toBe(
      "https://example.com/v1/auth/login",
    );
    expect(getAuthEndpointUrl("create")).toBe(
      "https://example.com/v1/auth/create",
    );
    expect(getAuthEndpointUrl("refresh")).toBe(
      "https://example.com/v1/auth/refresh",
    );
  });

  it("strips trailing slash from server URL", () => {
    setServerUrl("https://example.com/");
    setAPIEndpoints(mockEndpoints);
    expect(getAuthEndpointUrl("login")).toBe(
      "https://example.com/v1/auth/login",
    );
  });
});
