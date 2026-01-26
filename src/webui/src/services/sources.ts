// Sources service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";

export interface Source {
  id: string;
  name: string;
  url: string;
  component_ids: string[];
  retrieval_method: string;
  url_template?: string;
  forge_type: string;
  version_filter?: string;
  default_version?: string;
  priority: number;
  enabled: boolean;
  is_system: boolean;
  owner_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSourceRequest {
  name: string;
  url: string;
  component_ids?: string[];
  retrieval_method?: string;
  url_template?: string;
  forge_type?: string;
  version_filter?: string;
  default_version?: string;
  priority?: number;
  enabled?: boolean;
  is_system?: boolean;
}

export interface UpdateSourceRequest {
  name?: string;
  url?: string;
  component_ids?: string[];
  retrieval_method?: string;
  url_template?: string;
  forge_type?: string;
  version_filter?: string;
  default_version?: string;
  priority?: number;
  enabled?: boolean;
}

export type ListResult =
  | { success: true; sources: Source[] }
  | {
      success: false;
      error:
        | "unauthorized"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type GetResult =
  | { success: true; source: Source }
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

export type CreateResult =
  | { success: true; source: Source }
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
  | { success: true; source: Source }
  | {
      success: false;
      error:
        | "not_found"
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

// List all sources (system + user sources) for the current user
export async function listSources(): Promise<ListResult> {
  const url = getApiUrl("/sources");

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
      const sources: Source[] = data.sources || [];
      return { success: true, sources };
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
      message: "Failed to fetch sources",
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

// Get a source by ID
export async function getSourceById(id: string): Promise<GetResult> {
  const url = getApiUrl(`/sources/${id}`);

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
      const source: Source = await response.json();
      return { success: true, source };
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
        message: "You do not have access to this source",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Source not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch source",
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

// Create a source (user source, or system source if is_system=true and user is admin)
export async function createSource(
  request: CreateSourceRequest,
): Promise<CreateResult> {
  const url = getApiUrl("/sources");

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
      const source = await response.json();
      return { success: true, source };
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
        message: "Admin access required for system sources",
      };
    }

    if (response.status === 400) {
      return {
        success: false,
        error: "invalid_request",
        message: "Invalid source data",
      };
    }

    if (response.status === 409) {
      return {
        success: false,
        error: "conflict",
        message: "A source with this URL already exists",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to create source",
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

// Update a source by ID
export async function updateSource(
  id: string,
  request: UpdateSourceRequest,
): Promise<UpdateResult> {
  const url = getApiUrl(`/sources/${id}`);

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
      const source = await response.json();
      return { success: true, source };
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
        message: "You do not have permission to update this source",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Source not found",
      };
    }

    if (response.status === 400) {
      return {
        success: false,
        error: "invalid_request",
        message: "Invalid source data",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to update source",
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

// Delete a source by ID
export async function deleteSource(id: string): Promise<DeleteResult> {
  const url = getApiUrl(`/sources/${id}`);

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

    if (response.ok) {
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
        message: "You do not have permission to delete this source",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Source not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to delete source",
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
