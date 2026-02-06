---
name: add-webui-view
description: Scaffold a new WebUI view for the SolidJS frontend with component file, service layer, i18n translations, and App.tsx wiring. Use when adding a new page to the web interface.
argument-hint: "[view-name] [description]"

---

# Add WebUI View

Scaffold a complete SolidJS view following the project's established patterns.

## Arguments

- `$ARGUMENTS[0]` -- View name in PascalCase (e.g., `BoardProfiles`, `BuildHistory`)
- `$ARGUMENTS[1]` -- Short description (e.g., "Board profile management")

## Steps

### 1. Create the view directory

Create `src/webui/src/views/$0/`:

**`$0.tsx`** -- Main view component:

```tsx
import type { Component } from "solid-js";
import { createSignal, onMount, For, Show } from "solid-js";
import { useI18n } from "../../services/i18n";
// Import service functions as needed

interface ${0}Props {
  isLoggedIn: boolean;
  user: { role: string; username: string } | null;
  // Add navigation callbacks as needed
}

export const $0: Component<${0}Props> = (props) => {
  const { t } = useI18n();
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  onMount(async () => {
    await fetchData();
  });

  const fetchData = async () => {
    setIsLoading(true);
    setError(null);
    try {
      // Call service functions
    } catch (err) {
      setError(String(err));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <section class="h-full w-full relative p-6">
      <header class="mb-6">
        <h1 class="text-2xl font-bold">{t("$0_lower.title")}</h1>
        <p class="text-sm text-neutral-400">{t("$0_lower.subtitle")}</p>
      </header>

      <Show when={error()}>
        <aside class="bg-red-900/20 border border-red-800 rounded-lg p-4 mb-4">
          <p class="text-red-400">{error()}</p>
        </aside>
      </Show>

      <Show when={!isLoading()} fallback={<p>{t("common.loading")}</p>}>
        {/* View content using semantic HTML5 tags only */}
        <article>
          {/* Content here */}
        </article>
      </Show>
    </section>
  );
};
```

**`index.tsx`** -- Barrel export:

```tsx
export { $0 } from "./$0";
```

### 2. Create the service file

Create `src/webui/src/services/$0_lower.ts`:

```typescript
import { getApiUrl, getAuthHeaders } from "./api";

export interface $0Item {
  id: string;
  name: string;
  // Add fields matching the API response
  created_at: string;
  updated_at: string;
}

export type ListResult =
  | { success: true; items: $0Item[] }
  | { success: false; error: "network_error" | "not_configured" | "internal_error"; message: string };

export async function list$0Items(): Promise<ListResult> {
  const url = getApiUrl("/$0_lower");
  if (!url) {
    return { success: false, error: "not_configured", message: "Server not configured" };
  }

  try {
    const response = await fetch(url, {
      method: "GET",
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      return { success: false, error: "internal_error", message: error.message || "Request failed" };
    }

    const data = await response.json();
    return { success: true, items: data.items || [] };
  } catch (err) {
    return { success: false, error: "network_error", message: String(err) };
  }
}
```

### 3. Add i18n translations

Create translation files in all 3 locales. At minimum:

**`src/webui/src/locales/en/$0_lower.json`**:
```json
{
  "title": "$1",
  "subtitle": "Description of this view",
  "empty": {
    "title": "No items found",
    "description": "No items have been created yet."
  },
  "table": {
    "columns": {
      "name": "Name",
      "created": "Created"
    }
  }
}
```

Create equivalent files for `fr/` and `de/` locales with translated values.

### 4. Register translations in i18n service

Edit `src/webui/src/services/i18n.ts`:

- Add the dynamic import in the `loadLocale` function alongside the existing ones
- Add the namespace key to the merged dictionary

### 5. Wire into App.tsx

Edit `src/webui/src/App.tsx`:

1. Import the view: `import { $0 } from "./views/$0";`
2. Add to `ViewType` union: `| "$0_lower"`
3. Add navigation state and handlers
4. Add `<Match>` block inside the `<Switch>`:
```tsx
<Match when={currentView() === "$0_lower"}>
  <$0 isLoggedIn={isLoggedIn()} user={authState().user} />
</Match>
```
5. Add menu item if applicable in `menuItems()`

### 6. Verify

Run: `cd src/webui && /home/flint/.bun/bin/bun run build` to confirm compilation.

## Conventions to follow

- **NEVER** use `<div>` or `<span>` -- only semantic HTML5 tags (`<section>`, `<article>`, `<header>`, `<nav>`, `<aside>`, `<figure>`, `<>...</>`)
- TailwindCSS 4.x for all styling
- Phosphor Icons for iconography
- i18n keys follow pattern: `namespace.section.key` (e.g., `boards.table.columns.name`)
- Service functions return discriminated unions: `{ success: true; data } | { success: false; error; message }`
- Auth headers via `getAuthHeaders()`, URL construction via `getApiUrl(path)`
- Admin-only features gated by `props.user?.role === "root"`
