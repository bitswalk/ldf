import type { Component } from "solid-js";
import { createSignal, Show } from "solid-js";
import { Spinner } from "../Spinner";
import type { CreateSourceRequest, Source } from "../../services/sources";

interface SourceFormProps {
  onSubmit: (data: CreateSourceRequest) => void;
  onCancel: () => void;
  initialData?: Source;
  isSubmitting?: boolean;
}

export const SourceForm: Component<SourceFormProps> = (props) => {
  const [name, setName] = createSignal(props.initialData?.name || "");
  const [url, setUrl] = createSignal(props.initialData?.url || "");
  const [priority, setPriority] = createSignal(props.initialData?.priority ?? 0);
  const [enabled, setEnabled] = createSignal(props.initialData?.enabled ?? true);
  const [errors, setErrors] = createSignal<{ name?: string; url?: string }>({});

  const validateForm = (): boolean => {
    const newErrors: { name?: string; url?: string } = {};

    if (!name().trim()) {
      newErrors.name = "Name is required";
    }

    if (!url().trim()) {
      newErrors.url = "URL is required";
    } else {
      try {
        new URL(url());
      } catch {
        newErrors.url = "Invalid URL format";
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: Event) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    props.onSubmit({
      name: name().trim(),
      url: url().trim(),
      priority: priority(),
      enabled: enabled(),
    });
  };

  const isEditing = () => !!props.initialData;

  return (
    <form onSubmit={handleSubmit} class="flex flex-col gap-6">
      <div class="space-y-2">
        <label class="text-sm font-medium" for="source-name">
          Name <span class="text-destructive">*</span>
        </label>
        <input
          id="source-name"
          type="text"
          class={`w-full px-3 py-2 bg-background border-2 rounded-md focus:outline-none transition-colors ${
            errors().name
              ? "border-destructive focus:border-destructive"
              : "border-border focus:border-primary"
          }`}
          placeholder="e.g., Ubuntu Releases"
          value={name()}
          onInput={(e) => {
            setName(e.target.value);
            if (errors().name) {
              setErrors((prev) => ({ ...prev, name: undefined }));
            }
          }}
        />
        <Show when={errors().name}>
          <p class="text-xs text-destructive">{errors().name}</p>
        </Show>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="source-url">
          URL <span class="text-destructive">*</span>
        </label>
        <input
          id="source-url"
          type="text"
          class={`w-full px-3 py-2 bg-background border-2 rounded-md focus:outline-none transition-colors font-mono text-sm ${
            errors().url
              ? "border-destructive focus:border-destructive"
              : "border-border focus:border-primary"
          }`}
          placeholder="https://example.com/releases"
          value={url()}
          onInput={(e) => {
            setUrl(e.target.value);
            if (errors().url) {
              setErrors((prev) => ({ ...prev, url: undefined }));
            }
          }}
        />
        <Show when={errors().url}>
          <p class="text-xs text-destructive">{errors().url}</p>
        </Show>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="source-priority">
          Priority
        </label>
        <input
          id="source-priority"
          type="number"
          class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary transition-colors"
          placeholder="0"
          value={priority()}
          onInput={(e) => setPriority(parseInt(e.target.value) || 0)}
        />
        <p class="text-xs text-muted-foreground">
          Lower values have higher priority. Sources are sorted by priority
          ascending.
        </p>
      </div>

      <div class="flex items-center gap-3">
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={enabled()}
            onChange={(e) => setEnabled(e.target.checked)}
            class="sr-only peer"
          />
          <div class="w-11 h-6 bg-muted rounded-full peer peer-checked:bg-primary peer-focus:ring-2 peer-focus:ring-primary/20 transition-colors after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-background after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full"></div>
        </label>
        <span class="text-sm font-medium">Enabled</span>
      </div>

      <nav class="flex justify-end gap-3 pt-4 border-t border-border">
        <button
          type="button"
          onClick={props.onCancel}
          disabled={props.isSubmitting}
          class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={props.isSubmitting}
          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <Show when={props.isSubmitting}>
            <Spinner size="sm" />
          </Show>
          <span>{isEditing() ? "Update Source" : "Add Source"}</span>
        </button>
      </nav>
    </form>
  );
};
