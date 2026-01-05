// Components service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";

export type VersionRule = "pinned" | "latest-stable" | "latest-lts";

export interface Component {
  id: string;
  name: string;
  category: string; // Primary category (first in categories list)
  categories?: string[]; // All categories the component belongs to
  display_name: string;
  description?: string;
  artifact_pattern?: string;
  default_url_template?: string;
  github_normalized_template?: string;
  is_optional: boolean;
  is_system: boolean;
  owner_id?: string;
  default_version?: string;
  default_version_rule?: VersionRule;
  created_at: string;
  updated_at: string;
}

export interface SourceVersion {
  id: string;
  source_id: string;
  source_type: string;
  version: string;
  version_type: "mainline" | "stable" | "longterm" | "linux-next";
  release_date?: string;
  download_url?: string;
  checksum?: string;
  checksum_type?: string;
  file_size?: number;
  is_stable: boolean;
  discovered_at: string;
}

export interface ComponentVersionsResponse {
  versions: SourceVersion[];
  total: number;
  limit: number;
  offset: number;
}

export interface ResolvedVersionResponse {
  rule: string;
  resolved_version: string;
  version?: SourceVersion;
}

export type ListResult =
  | { success: true; components: Component[] }
  | {
      success: false;
      error: "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export type GetResult =
  | { success: true; component: Component }
  | {
      success: false;
      error:
        | "not_found"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type CategoriesResult =
  | { success: true; categories: string[] }
  | {
      success: false;
      error: "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export interface CreateComponentRequest {
  name: string;
  category: string;
  display_name: string;
  description?: string;
  artifact_pattern?: string;
  default_url_template?: string;
  github_normalized_template?: string;
  is_optional?: boolean;
  default_version?: string;
  default_version_rule?: VersionRule;
}

export interface UpdateComponentRequest {
  name?: string;
  category?: string;
  display_name?: string;
  description?: string;
  artifact_pattern?: string;
  default_url_template?: string;
  github_normalized_template?: string;
  is_optional?: boolean;
  default_version?: string;
  default_version_rule?: VersionRule;
}

export interface VersionQueryParams {
  limit?: number;
  offset?: number;
  version_type?: "all" | "stable" | "longterm" | "mainline";
}

export type VersionsResult =
  | { success: true; data: ComponentVersionsResponse }
  | {
      success: false;
      error:
        | "not_found"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type ResolveVersionResult =
  | { success: true; data: ResolvedVersionResponse }
  | {
      success: false;
      error:
        | "not_found"
        | "invalid_rule"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type CreateResult =
  | { success: true; component: Component }
  | {
      success: false;
      error:
        | "conflict"
        | "unauthorized"
        | "forbidden"
        | "invalid_request"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type UpdateResult =
  | { success: true; component: Component }
  | {
      success: false;
      error:
        | "not_found"
        | "conflict"
        | "forbidden"
        | "unauthorized"
        | "invalid_request"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type DeleteResult =
  | { success: true }
  | {
      success: false;
      error:
        | "not_found"
        | "forbidden"
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

// List all components
export async function listComponents(): Promise<ListResult> {
  const url = getApiUrl("/components");

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
      const data = await response.json();
      const components: Component[] = data.components || [];
      return { success: true, components };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch components",
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

// Get a component by ID
export async function getComponent(id: string): Promise<GetResult> {
  const url = getApiUrl(`/components/${id}`);

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
      const component = await response.json();
      return { success: true, component };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Component not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch component",
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

// List components by category
export async function listComponentsByCategory(
  category: string,
): Promise<ListResult> {
  const url = getApiUrl(`/components/category/${encodeURIComponent(category)}`);

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
      const data = await response.json();
      const components: Component[] = data.components || [];
      return { success: true, components };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch components",
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

// Get all component categories
export async function getCategories(): Promise<CategoriesResult> {
  const url = getApiUrl("/components/categories");

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
      const data = await response.json();
      const categories: string[] = data.categories || [];
      return { success: true, categories };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch categories",
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

// Group components by category
// Components with multiple categories will appear in each category they belong to
export function groupByCategory(
  components: Component[],
): Record<string, Component[]> {
  return components.reduce(
    (acc, component) => {
      // Use categories array if available, otherwise fall back to single category
      const cats = component.categories?.length
        ? component.categories
        : [component.category];
      for (const category of cats) {
        if (!acc[category]) {
          acc[category] = [];
        }
        acc[category].push(component);
      }
      return acc;
    },
    {} as Record<string, Component[]>,
  );
}

// Get category display name
export function getCategoryDisplayName(category: string): string {
  const names: Record<string, string> = {
    core: "Core",
    bootloader: "Bootloader",
    init: "Init System",
    systemd: "systemd",
    network: "Network",
    dns: "DNS",
    storage: "Storage",
    device: "Device Management",
    user: "User Management",
    extensions: "Extensions",
    tools: "Tools",
    runtime: "Runtime",
    security: "Security",
    desktop: "Desktop",
    container: "Container",
    virtualization: "Virtualization",
  };
  return (
    names[category] || category.charAt(0).toUpperCase() + category.slice(1)
  );
}

// Create a new component (root only)
export async function createComponent(
  request: CreateComponentRequest,
): Promise<CreateResult> {
  const url = getApiUrl("/components");

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
      body: JSON.stringify(request),
    });

    if (response.ok) {
      const component = await response.json();
      return { success: true, component };
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
        message: "Admin access required",
      };
    }

    if (response.status === 400) {
      const data = await response.json().catch(() => ({}));
      return {
        success: false,
        error: "invalid_request",
        message: data.message || "Invalid component data",
      };
    }

    if (response.status === 409) {
      return {
        success: false,
        error: "conflict",
        message: "A component with this name already exists",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to create component",
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

// Update a component (root only)
export async function updateComponent(
  id: string,
  request: UpdateComponentRequest,
): Promise<UpdateResult> {
  const url = getApiUrl(`/components/${id}`);

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
      body: JSON.stringify(request),
    });

    if (response.ok) {
      const component = await response.json();
      return { success: true, component };
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
        message: "Admin access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Component not found",
      };
    }

    if (response.status === 400) {
      const data = await response.json().catch(() => ({}));
      return {
        success: false,
        error: "invalid_request",
        message: data.message || "Invalid component data",
      };
    }

    if (response.status === 409) {
      return {
        success: false,
        error: "conflict",
        message: "A component with this name already exists",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to update component",
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

// Delete a component (root only)
export async function deleteComponent(id: string): Promise<DeleteResult> {
  const url = getApiUrl(`/components/${id}`);

  if (!url) {
    return {
      success: false,
      error: "not_configured",
      message: "Server connection not configured",
    };
  }

  try {
    const response = await fetch(url, {
      method: "DELETE",
      headers: getAuthHeaders(),
    });

    if (response.ok || response.status === 204) {
      return { success: true };
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
        message: "Admin access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Component not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to delete component",
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

// Available component categories
export const COMPONENT_CATEGORIES = [
  "core",
  "bootloader",
  "init",
  "systemd",
  "container",
  "virtualization",
  "runtime",
  "security",
  "desktop",
] as const;

export type ComponentCategory = (typeof COMPONENT_CATEGORIES)[number];

// Version rules
export const VERSION_RULES: { value: VersionRule; label: string }[] = [
  { value: "latest-stable", label: "Latest Stable" },
  { value: "latest-lts", label: "Latest LTS" },
  { value: "pinned", label: "Pinned Version" },
];

// Get versions for a component
export async function getComponentVersions(
  componentId: string,
  params?: VersionQueryParams,
): Promise<VersionsResult> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.set("limit", params.limit.toString());
  if (params?.offset) queryParams.set("offset", params.offset.toString());
  if (params?.version_type)
    queryParams.set("version_type", params.version_type);

  const queryString = queryParams.toString();
  const url = getApiUrl(
    `/components/${componentId}/versions${queryString ? `?${queryString}` : ""}`,
  );

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
      const data: ComponentVersionsResponse = await response.json();
      return { success: true, data };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Component not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch component versions",
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

// Resolve a version rule to an actual version
export async function resolveVersionRule(
  componentId: string,
  rule: VersionRule,
): Promise<ResolveVersionResult> {
  const url = getApiUrl(
    `/components/${componentId}/resolve-version?rule=${encodeURIComponent(rule)}`,
  );

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
      const data: ResolvedVersionResponse = await response.json();
      return { success: true, data };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Component or version not found",
      };
    }

    if (response.status === 400) {
      return {
        success: false,
        error: "invalid_rule",
        message: "Invalid version rule",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to resolve version",
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

// Get version rule display label
export function getVersionRuleLabel(rule: VersionRule | undefined): string {
  if (!rule) return "Latest Stable";
  const found = VERSION_RULES.find((r) => r.value === rule);
  return found?.label || rule;
}

// Get version type display label
export function getVersionTypeLabel(
  versionType: SourceVersion["version_type"],
): string {
  const labels: Record<SourceVersion["version_type"], string> = {
    mainline: "Mainline",
    stable: "Stable",
    longterm: "LTS",
    "linux-next": "Linux Next",
  };
  return labels[versionType] || versionType;
}
