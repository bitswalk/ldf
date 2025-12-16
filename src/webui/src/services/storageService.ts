// Storage service for managing persistent local data

const STORAGE_KEYS = {
  SERVER_URL: "ldf_server_url",
  AUTH_TOKEN: "ldf_auth_token",
  USER_INFO: "ldf_user_info",
  API_ENDPOINTS: "ldf_api_endpoints",
} as const;

export interface StoredUserInfo {
  id: string;
  name: string;
  email: string;
  role: string;
}

export interface AuthEndpoints {
  create: string;
  login: string;
  logout: string;
}

export interface APIEndpoints {
  health: string;
  version: string;
  api_v1: string;
  auth: AuthEndpoints;
}

export interface APIInfo {
  name: string;
  description: string;
  version: string;
  api_versions: string[];
  endpoints: APIEndpoints;
}

export function getServerUrl(): string | null {
  return localStorage.getItem(STORAGE_KEYS.SERVER_URL);
}

export function setServerUrl(url: string): void {
  localStorage.setItem(STORAGE_KEYS.SERVER_URL, url);
}

export function clearServerUrl(): void {
  localStorage.removeItem(STORAGE_KEYS.SERVER_URL);
}

export function getAPIEndpoints(): APIEndpoints | null {
  const stored = localStorage.getItem(STORAGE_KEYS.API_ENDPOINTS);
  if (!stored) return null;

  try {
    return JSON.parse(stored) as APIEndpoints;
  } catch {
    return null;
  }
}

export function setAPIEndpoints(endpoints: APIEndpoints): void {
  localStorage.setItem(STORAGE_KEYS.API_ENDPOINTS, JSON.stringify(endpoints));
}

export function clearAPIEndpoints(): void {
  localStorage.removeItem(STORAGE_KEYS.API_ENDPOINTS);
}

export function getAuthToken(): string | null {
  return localStorage.getItem(STORAGE_KEYS.AUTH_TOKEN);
}

export function setAuthToken(token: string): void {
  localStorage.setItem(STORAGE_KEYS.AUTH_TOKEN, token);
}

export function clearAuthToken(): void {
  localStorage.removeItem(STORAGE_KEYS.AUTH_TOKEN);
}

export function getUserInfo(): StoredUserInfo | null {
  const stored = localStorage.getItem(STORAGE_KEYS.USER_INFO);
  if (!stored) return null;

  try {
    return JSON.parse(stored) as StoredUserInfo;
  } catch {
    return null;
  }
}

export function setUserInfo(user: StoredUserInfo): void {
  localStorage.setItem(STORAGE_KEYS.USER_INFO, JSON.stringify(user));
}

export function clearUserInfo(): void {
  localStorage.removeItem(STORAGE_KEYS.USER_INFO);
}

export function clearAllAuth(): void {
  clearAuthToken();
  clearUserInfo();
}

export function clearAll(): void {
  clearServerUrl();
  clearAPIEndpoints();
  clearAllAuth();
}

export function hasServerConnection(): boolean {
  const url = getServerUrl();
  return url !== null && url.length > 0;
}

export function hasCompleteServerConnection(): boolean {
  const url = getServerUrl();
  const endpoints = getAPIEndpoints();
  return url !== null && url.length > 0 && endpoints !== null;
}

export async function discoverAPIEndpoints(): Promise<{
  success: boolean;
  error?: string;
}> {
  const serverUrl = getServerUrl();
  if (!serverUrl) {
    return { success: false, error: "No server URL configured" };
  }

  try {
    const baseUrl = serverUrl.replace(/\/$/, "");
    const response = await fetch(`${baseUrl}/`, {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
    });

    if (response.ok) {
      const apiInfo: APIInfo = await response.json();

      if (apiInfo.name === "ldfd" && apiInfo.endpoints?.auth) {
        setAPIEndpoints(apiInfo.endpoints);
        return { success: true };
      } else {
        return {
          success: false,
          error: "Server does not appear to be an LDF server",
        };
      }
    } else {
      return {
        success: false,
        error: `Server responded with status ${response.status}`,
      };
    }
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Failed to connect to server",
    };
  }
}

export function hasAuthSession(): boolean {
  return getAuthToken() !== null && getUserInfo() !== null;
}

export function getAuthEndpointUrl(
  endpoint: keyof AuthEndpoints,
): string | null {
  const serverUrl = getServerUrl();
  const endpoints = getAPIEndpoints();

  if (!serverUrl || !endpoints) return null;

  const baseUrl = serverUrl.replace(/\/$/, "");
  return `${baseUrl}${endpoints.auth[endpoint]}`;
}
