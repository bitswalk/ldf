// Sources service for LDF server communication

import { getServerUrl, getAuthToken } from "./storage";

export interface Source {
  id: string;
  name: string;
  url: string;
  component_id?: string;
  retrieval_method: string;
  url_template?: string;
  priority: number;
  enabled: boolean;
  is_system: boolean;
  owner_id?: string;
  created_at: string;
  updated_at: string;
}

export interface SourceDefault {
  id: string;
  name: string;
  url: string;
  component_id?: string;
  retrieval_method: string;
  url_template?: string;
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserSource {
  id: string;
  owner_id: string;
  name: string;
  url: string;
  component_id?: string;
  retrieval_method: string;
  url_template?: string;
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateSourceRequest {
  name: string;
  url: string;
  component_id?: string;
  retrieval_method?: string;
  url_template?: string;
  priority?: number;
  enabled?: boolean;
}

export interface UpdateSourceRequest {
  name?: string;
  url?: string;
  component_id?: string;
  retrieval_method?: string;
  url_template?: string;
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

export type ListDefaultsResult =
  | { success: true; sources: SourceDefault[] }
  | {
      success: false;
      error:
        | "unauthorized"
        | "forbidden"
        | "network_error"
        | "not_configured"
        | "internal_error";
      message: string;
    };

export type CreateResult =
  | { success: true; source: SourceDefault | UserSource }
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
  | { success: true; source: SourceDefault | UserSource }
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

// List merged sources (defaults + user sources) for the current user
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

// List default sources (admin only)
export async function listDefaultSources(): Promise<ListDefaultsResult> {
  const url = getApiUrl("/sources/defaults");

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
      const sources: SourceDefault[] = data.sources || [];
      return { success: true, sources };
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

    return {
      success: false,
      error: "internal_error",
      message: "Failed to fetch default sources",
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

// Create a user source
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

// Create a default source (admin only)
export async function createDefaultSource(
  request: CreateSourceRequest,
): Promise<CreateResult> {
  const url = getApiUrl("/sources/defaults");

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
        message: "Admin access required",
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
        message: "A default source with this URL already exists",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to create default source",
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

// Update a user source
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
        message: "You can only update your own sources",
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

// Update a default source (admin only)
export async function updateDefaultSource(
  id: string,
  request: UpdateSourceRequest,
): Promise<UpdateResult> {
  const url = getApiUrl(`/sources/defaults/${id}`);

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
        message: "Admin access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Default source not found",
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
      message: "Failed to update default source",
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

// Delete a user source
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
        message: "You can only delete your own sources",
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

// Delete a default source (admin only)
export async function deleteDefaultSource(id: string): Promise<DeleteResult> {
  const url = getApiUrl(`/sources/defaults/${id}`);

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
        message: "Admin access required",
      };
    }

    if (response.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Default source not found",
      };
    }

    return {
      success: false,
      error: "internal_error",
      message: "Failed to delete default source",
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
