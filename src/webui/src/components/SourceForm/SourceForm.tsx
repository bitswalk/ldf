import type { Component as SolidComponent } from "solid-js";
import {
  createSignal,
  createResource,
  createEffect,
  createMemo,
  Show,
  For,
} from "solid-js";
import { Spinner } from "../Spinner";
import { Icon } from "../Icon";
import type { CreateSourceRequest, Source } from "../../services/sources";
import {
  listComponents,
  groupByCategory,
  getCategoryDisplayName,
  type Component,
} from "../../services/components";
import { t } from "../../services/i18n";

// Template variable definitions for the help tooltip
const TEMPLATE_VARIABLES = [
  { name: "{base_url}", example: "(source URL)", desc: "The source base URL" },
  { name: "{version}", example: "6.12.5", desc: "Full version string" },
  { name: "{tag}", example: "v6.12.5", desc: "Version with 'v' prefix" },
  { name: "{tag_short}", example: "v6.12", desc: "Major.minor only" },
  {
    name: "{tag_compact}",
    example: "v6125",
    desc: "Without dots (systemd style)",
  },
  { name: "{major}", example: "6", desc: "Major version number" },
  { name: "{minor}", example: "12", desc: "Minor version number" },
  { name: "{patch}", example: "5", desc: "Patch version number" },
  {
    name: "{major_x}",
    example: "6.x",
    desc: "Major with .x (kernel.org style)",
  },
];

interface SourceFormProps {
  onSubmit: (data: CreateSourceRequest) => void;
  onCancel: () => void;
  initialData?: Source;
  isSubmitting?: boolean;
}

export const SourceForm: SolidComponent<SourceFormProps> = (props) => {
  const [name, setName] = createSignal(props.initialData?.name || "");
  const [url, setUrl] = createSignal(props.initialData?.url || "");
  const [componentIds, setComponentIds] = createSignal<string[]>(
    props.initialData?.component_ids || [],
  );
  const [retrievalMethod, setRetrievalMethod] = createSignal(
    props.initialData?.retrieval_method || "release",
  );
  const [urlTemplate, setUrlTemplate] = createSignal(
    props.initialData?.url_template || "",
  );
  const [priority, setPriority] = createSignal(
    props.initialData?.priority ?? 0,
  );
  const [enabled, setEnabled] = createSignal(
    props.initialData?.enabled ?? true,
  );
  const [errors, setErrors] = createSignal<{ name?: string; url?: string }>({});

  // Fetch components for the dropdown
  const [components] = createResource(async () => {
    const result = await listComponents();
    if (result.success) {
      return result.components;
    }
    return [];
  });

  const groupedComponents = () => {
    const comps = components();
    if (!comps) return {};
    return groupByCategory(comps);
  };

  // Get selected components for display and URL template preview
  const selectedComponents = createMemo(() => {
    const comps = components();
    const ids = componentIds();
    if (!comps || ids.length === 0) return [];
    return comps.filter((c) => ids.includes(c.id));
  });

  // Get the first selected component for URL template preview
  const firstSelectedComponent = () => {
    const selected = selectedComponents();
    return selected.length > 0 ? selected[0] : null;
  };

  // Toggle component selection
  const toggleComponent = (id: string) => {
    setComponentIds((prev) => {
      if (prev.includes(id)) {
        return prev.filter((cid) => cid !== id);
      }
      return [...prev, id];
    });
  };

  // Check if a component is selected
  const isComponentSelected = (id: string) => componentIds().includes(id);

  const previewUrl = () => {
    const baseUrl = url().trim();
    if (!baseUrl) return "";

    const comp = firstSelectedComponent();
    let template = urlTemplate().trim();

    if (!template && comp) {
      // Use GitHub template if URL is GitHub
      if (baseUrl.includes("github.com") && comp.github_normalized_template) {
        template = comp.github_normalized_template;
      } else if (comp.default_url_template) {
        template = comp.default_url_template;
      }
    }

    if (!template) return baseUrl;

    // Apply template with example version 6.12.5
    const normalizedBase = baseUrl.replace(/\/$/, "").replace(/\.git$/, "");
    const exampleVersion = "6.12.5";

    return template
      .replace(/{base_url}/g, normalizedBase)
      .replace(/{version}/g, exampleVersion)
      .replace(/{tag}/g, "v" + exampleVersion)
      .replace(/{tag_short}/g, "v6.12")
      .replace(/{tag_compact}/g, "v6125")
      .replace(/{major}/g, "6")
      .replace(/{minor}/g, "12")
      .replace(/{patch}/g, "5")
      .replace(/{major_x}/g, "6.x");
  };

  const validateForm = (): boolean => {
    const newErrors: { name?: string; url?: string } = {};

    if (!name().trim()) {
      newErrors.name = t("sources.form.name.required");
    }

    if (!url().trim()) {
      newErrors.url = t("sources.form.url.required");
    } else {
      try {
        new URL(url());
      } catch {
        newErrors.url = t("sources.form.url.invalid");
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

    const request: CreateSourceRequest = {
      name: name().trim(),
      url: url().trim(),
      priority: priority(),
      enabled: enabled(),
      component_ids: componentIds(),
    };

    if (retrievalMethod()) {
      request.retrieval_method = retrievalMethod();
    }
    if (urlTemplate().trim()) {
      request.url_template = urlTemplate().trim();
    }

    props.onSubmit(request);
  };

  const isEditing = () => !!props.initialData;

  return (
    <form onSubmit={handleSubmit} class="flex flex-col gap-6">
      <div class="space-y-2">
        <label class="text-sm font-medium" for="source-name">
          {t("sources.form.name.label")} <span class="text-destructive">*</span>
        </label>
        <input
          id="source-name"
          type="text"
          class={`w-full px-3 py-2 bg-background border-2 rounded-md focus:outline-none transition-colors ${
            errors().name
              ? "border-destructive focus:border-destructive"
              : "border-border focus:border-primary"
          }`}
          placeholder={t("sources.form.name.placeholder")}
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
        <label class="text-sm font-medium">
          {t("sources.form.component.label")}
        </label>
        <div class="border-2 border-border rounded-md max-h-64 overflow-y-auto">
          <For each={Object.entries(groupedComponents())}>
            {([category, comps]) => (
              <div class="border-b border-border last:border-b-0">
                <div class="px-3 py-2 bg-muted/50 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                  {getCategoryDisplayName(category)}
                </div>
                <div class="divide-y divide-border/50">
                  <For each={comps as Component[]}>
                    {(comp) => (
                      <label class="flex items-center gap-3 px-3 py-2 cursor-pointer hover:bg-muted/30 transition-colors">
                        <input
                          type="checkbox"
                          checked={isComponentSelected(comp.id)}
                          onChange={() => toggleComponent(comp.id)}
                          class="w-4 h-4 rounded border-border text-primary focus:ring-primary/20"
                        />
                        <span class="text-sm flex-1">
                          {comp.display_name}
                          {comp.is_optional && (
                            <span class="text-muted-foreground ml-1">
                              (optional)
                            </span>
                          )}
                        </span>
                      </label>
                    )}
                  </For>
                </div>
              </div>
            )}
          </For>
          <Show when={Object.keys(groupedComponents()).length === 0}>
            <div class="px-3 py-4 text-center text-sm text-muted-foreground">
              Loading components...
            </div>
          </Show>
        </div>
        <Show when={selectedComponents().length > 0}>
          <div class="flex flex-wrap gap-1.5">
            <For each={selectedComponents()}>
              {(comp) => (
                <span class="inline-flex items-center gap-1 px-2 py-0.5 bg-primary/10 text-primary text-xs rounded-full">
                  {comp.display_name}
                  <button
                    type="button"
                    onClick={() => toggleComponent(comp.id)}
                    class="hover:bg-primary/20 rounded-full p-0.5"
                  >
                    <Icon name="x" size="xs" />
                  </button>
                </span>
              )}
            </For>
          </div>
        </Show>
        <p class="text-xs text-muted-foreground">
          {t("sources.form.component.help")}
        </p>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="source-url">
          {t("sources.form.url.label")} <span class="text-destructive">*</span>
        </label>
        <input
          id="source-url"
          type="text"
          class={`w-full px-3 py-2 bg-background border-2 rounded-md focus:outline-none transition-colors font-mono text-sm ${
            errors().url
              ? "border-destructive focus:border-destructive"
              : "border-border focus:border-primary"
          }`}
          placeholder={t("sources.form.url.placeholder")}
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
        <label class="text-sm font-medium">
          {t("sources.form.retrievalMethod.label")}
        </label>
        <div class="flex gap-4">
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="retrieval-method"
              value="release"
              checked={retrievalMethod() === "release"}
              onChange={() => setRetrievalMethod("release")}
              class="w-4 h-4 text-primary border-border focus:ring-primary"
            />
            <span class="text-sm">
              {t("sources.form.retrievalMethod.release")}
            </span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="retrieval-method"
              value="git"
              checked={retrievalMethod() === "git"}
              onChange={() => setRetrievalMethod("git")}
              class="w-4 h-4 text-primary border-border focus:ring-primary"
            />
            <span class="text-sm">{t("sources.form.retrievalMethod.git")}</span>
          </label>
        </div>
        <p class="text-xs text-muted-foreground">
          {t("sources.form.retrievalMethod.help")}
        </p>
      </div>

      <div class="space-y-2">
        <div class="flex items-center gap-2">
          <label class="text-sm font-medium" for="source-template">
            URL Template (optional)
          </label>
          <div class="relative group">
            <button
              type="button"
              class="w-5 h-5 rounded-full bg-muted hover:bg-muted/80 flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Template variables help"
            >
              <Icon name="question" size="xs" />
            </button>
            <div class="absolute left-0 top-full mt-2 z-50 hidden group-hover:block">
              <div class="bg-popover border border-border rounded-lg shadow-lg p-4 w-80">
                <h4 class="font-semibold text-sm mb-3">Template Variables</h4>
                <p class="text-xs text-muted-foreground mb-3">
                  Use these placeholders in your URL template. Example version:
                  6.12.5
                </p>
                <div class="space-y-2">
                  <For each={TEMPLATE_VARIABLES}>
                    {(variable) => (
                      <div class="flex items-start gap-2 text-xs">
                        <code class="px-1.5 py-0.5 bg-muted rounded font-mono text-primary whitespace-nowrap">
                          {variable.name}
                        </code>
                        <span class="text-muted-foreground flex-1">
                          {variable.desc}
                        </span>
                        <span class="font-mono text-foreground whitespace-nowrap">
                          {variable.example}
                        </span>
                      </div>
                    )}
                  </For>
                </div>
                <div class="mt-4 pt-3 border-t border-border">
                  <p class="text-xs font-medium mb-2">Examples:</p>
                  <div class="space-y-1.5 text-xs font-mono text-muted-foreground">
                    <p class="break-all">
                      <span class="text-foreground">kernel.org:</span>{" "}
                      {"{base_url}/{major_x}/linux-{version}.tar.xz"}
                    </p>
                    <p class="break-all">
                      <span class="text-foreground">systemd:</span>{" "}
                      {"{base_url}/archive/refs/tags/{tag_compact}.tar.gz"}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <input
          id="source-template"
          type="text"
          class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary transition-colors font-mono text-sm"
          placeholder="{base_url}/archive/refs/tags/v{version}.tar.gz"
          value={urlTemplate()}
          onInput={(e) => setUrlTemplate(e.target.value)}
        />
        <p class="text-xs text-muted-foreground">
          Custom URL template. Hover the <span class="font-medium">?</span> icon
          for available variables. Leave empty to use component defaults.
        </p>
      </div>

      <Show when={previewUrl()}>
        <div class="space-y-2 p-3 bg-muted/50 rounded-md border border-border">
          <label class="text-xs font-medium text-muted-foreground">
            URL Preview (example with version 6.12.5)
          </label>
          <p class="font-mono text-xs break-all">{previewUrl()}</p>
        </div>
      </Show>

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
