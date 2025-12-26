// Components service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";

export interface Component {
  id: string;
  name: string;
  category: string;
  display_name: string;
  description?: string;
  artifact_pattern?: string;
  default_url_template?: string;
  github_normalized_template?: string;
  is_optional: boolean;
  created_at: string;
  updated_at: string;
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
      error: "not_found" | "network_error" | "not_configured" | "internal_error";
      message: string;
    };

export type CategoriesResult =
  | { success: true; categories: string[] }
  | {
      success: false;
      error: "network_error" | "not_configured" | "internal_error";
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
  category: string
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
export function groupByCategory(
  components: Component[]
): Record<string, Component[]> {
  return components.reduce(
    (acc, component) => {
      const category = component.category;
      if (!acc[category]) {
        acc[category] = [];
      }
      acc[category].push(component);
      return acc;
    },
    {} as Record<string, Component[]>
  );
}

// Get category display name
export function getCategoryDisplayName(category: string): string {
  const names: Record<string, string> = {
    core: "Core",
    bootloader: "Bootloader",
    init: "Init System",
    runtime: "Runtime",
    security: "Security",
    desktop: "Desktop",
  };
  return names[category] || category.charAt(0).toUpperCase() + category.slice(1);
}
