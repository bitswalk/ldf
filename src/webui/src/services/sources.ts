import { authFetch, getApiUrl } from "./api";
// Sources service for LDF server communication


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
    const result = await authFetch(url);

    if (result.ok) {
      const data = result.data as any;
      const sources: Source[] = data.sources || [];
      return { success: true, sources };
    }

    if (result.status === 401) {
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
    const result = await authFetch(url);

    if (result.ok) {
      const source = result.data as any;
      return { success: true, source };
    }

    if (result.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (result.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "You do not have access to this source",
      };
    }

    if (result.status === 404) {
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
    const result = await authFetch(url, {
      method: "POST",
      body: JSON.stringify(request),
    });

    if (result.ok) {
      const source = result.data as any;
      return { success: true, source };
    }

    if (result.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (result.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "Admin access required for system sources",
      };
    }

    if (result.status === 400) {
      return {
        success: false,
        error: "invalid_request",
        message: "Invalid source data",
      };
    }

    if (result.status === 409) {
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
    const result = await authFetch(url, {
      method: "PUT",
      body: JSON.stringify(request),
    });

    if (result.ok) {
      const source = result.data as any;
      return { success: true, source };
    }

    if (result.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (result.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "You do not have permission to update this source",
      };
    }

    if (result.status === 404) {
      return {
        success: false,
        error: "not_found",
        message: "Source not found",
      };
    }

    if (result.status === 400) {
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
    const result = await authFetch(url, { method: "DELETE" });

    if (result.ok) {
      return { success: true };
    }

    if (result.status === 401) {
      return {
        success: false,
        error: "unauthorized",
        message: "Authentication required",
      };
    }

    if (result.status === 403) {
      return {
        success: false,
        error: "forbidden",
        message: "You do not have permission to delete this source",
      };
    }

    if (result.status === 404) {
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
