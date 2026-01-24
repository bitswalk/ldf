// Forge service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";

export type ForgeType =
  | "github"
  | "gitlab"
  | "gitea"
  | "codeberg"
  | "forgejo"
  | "generic";

export interface ForgeTypeInfo {
  type: ForgeType;
  display_name: string;
  description: string;
}

export interface RepoInfo {
  owner: string;
  repo: string;
  base_url: string;
  api_base_url: string;
}

export interface ForgeDefaults {
  url_template: string;
  version_filter: string;
  filter_source: "upstream" | "default";
}

export interface DetectResponse {
  forge_type: ForgeType;
  repo_info?: RepoInfo;
  defaults?: ForgeDefaults;
  forge_types: ForgeTypeInfo[];
}

export interface VersionPreview {
  version: string;
  included: boolean;
  reason?: string;
  is_prerelease: boolean;
}

export interface PreviewFilterResponse {
  total_versions: number;
  included_versions: number;
  excluded_versions: number;
  versions: VersionPreview[];
  applied_filter: string;
  filter_source: "custom" | "upstream" | "default";
}

export interface CommonFiltersResponse {
  filters: Record<string, string>;
}

export type DetectResult =
  | { success: true; data: DetectResponse }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type PreviewFilterResult =
  | { success: true; data: PreviewFilterResponse }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "upstream_error"
        | "internal_error";
      message: string;
    };

export type ListForgeTypesResult =
  | { success: true; forge_types: ForgeTypeInfo[] }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type CommonFiltersResult =
  | { success: true; filters: Record<string, string> }
  | {
      success: false;
      error:
        | "unauthorized"
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

// Detect forge type and get defaults for a URL
export async function detectForge(url: string): Promise<DetectResult> {
  const apiUrl = getApiUrl("/forge/detect");

  if (!apiUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(apiUrl, {
      method: "POST",
      headers: getAuthHeaders(),
      body: JSON.stringify({ url }),
    });

    if (response.ok) {
      const data: DetectResponse = await response.json();
      return { success: true, data };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to detect forge type",
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

// Preview version filter results with actual upstream versions
export async function previewFilter(
  url: string,
  forgeType?: string,
  versionFilter?: string,
): Promise<PreviewFilterResult> {
  const apiUrl = getApiUrl("/forge/preview-filter");

  if (!apiUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(apiUrl, {
      method: "POST",
      headers: getAuthHeaders(),
      body: JSON.stringify({
        url,
        forge_type: forgeType,
        version_filter: versionFilter,
      }),
    });

    if (response.ok) {
      const data: PreviewFilterResponse = await response.json();
      return { success: true, data };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (response.status === 502) {
      return {
        success: false,
        error: "upstream_error",
        message: "Failed to fetch versions from upstream repository",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to preview filter",
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

// List all available forge types
export async function listForgeTypes(): Promise<ListForgeTypesResult> {
  const apiUrl = getApiUrl("/forge/types");

  if (!apiUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(apiUrl, {
      method: "GET",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      const data = await response.json();
      return { success: true, forge_types: data.forge_types };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to list forge types",
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

// Get common filter presets
export async function getCommonFilters(): Promise<CommonFiltersResult> {
  const apiUrl = getApiUrl("/forge/common-filters");

  if (!apiUrl) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(apiUrl, {
      method: "GET",
      headers: getAuthHeaders(),
    });

    if (response.ok) {
      const data: CommonFiltersResponse = await response.json();
      return { success: true, filters: data.filters };
    }

    if (response.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to get common filters",
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

// Get display name for a forge type
export function getForgeTypeDisplayName(forgeType: ForgeType): string {
  const displayNames: Record<ForgeType, string> = {
    github: "GitHub",
    gitlab: "GitLab",
    gitea: "Gitea",
    codeberg: "Codeberg",
    forgejo: "Forgejo",
    generic: "Generic",
  };
  return displayNames[forgeType] || forgeType;
}

// All available forge types for local use
export const FORGE_TYPES: ForgeTypeInfo[] = [
  { type: "github", display_name: "GitHub", description: "GitHub.com repositories" },
  { type: "gitlab", display_name: "GitLab", description: "GitLab.com or self-hosted GitLab instances" },
  { type: "gitea", display_name: "Gitea", description: "Gitea self-hosted instances" },
  { type: "codeberg", display_name: "Codeberg", description: "Codeberg.org repositories" },
  { type: "forgejo", display_name: "Forgejo", description: "Forgejo self-hosted instances" },
  { type: "generic", display_name: "Generic", description: "Generic Git/HTTP sources (kernel.org style)" },
];
