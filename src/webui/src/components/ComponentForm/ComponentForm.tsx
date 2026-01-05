import type { Component as SolidComponent } from "solid-js";
import { createSignal, Show, For } from "solid-js";
import { Spinner } from "../Spinner";
import {
  type Component,
  type CreateComponentRequest,
  type VersionRule,
  COMPONENT_CATEGORIES,
  VERSION_RULES,
  getCategoryDisplayName,
  getVersionRuleLabel,
} from "../../services/components";
import { t } from "../../services/i18n";

interface ComponentFormProps {
  onSubmit: (data: CreateComponentRequest) => void;
  onCancel: () => void;
  initialData?: Component;
  isSubmitting?: boolean;
}

export const ComponentForm: SolidComponent<ComponentFormProps> = (props) => {
  const [name, setName] = createSignal(props.initialData?.name || "");
  const [category, setCategory] = createSignal(
    props.initialData?.category || "core",
  );
  const [displayName, setDisplayName] = createSignal(
    props.initialData?.display_name || "",
  );
  const [description, setDescription] = createSignal(
    props.initialData?.description || "",
  );
  const [artifactPattern, setArtifactPattern] = createSignal(
    props.initialData?.artifact_pattern || "",
  );
  const [defaultUrlTemplate, setDefaultUrlTemplate] = createSignal(
    props.initialData?.default_url_template || "",
  );
  const [githubNormalizedTemplate, setGithubNormalizedTemplate] = createSignal(
    props.initialData?.github_normalized_template || "",
  );
  const [isOptional, setIsOptional] = createSignal(
    props.initialData?.is_optional ?? false,
  );
  const [defaultVersionRule, setDefaultVersionRule] = createSignal<VersionRule>(
    props.initialData?.default_version_rule || "latest-stable",
  );
  const [defaultVersion, setDefaultVersion] = createSignal(
    props.initialData?.default_version || "",
  );
  const [errors, setErrors] = createSignal<{
    name?: string;
    displayName?: string;
    category?: string;
    defaultVersion?: string;
  }>({});

  const validateForm = (): boolean => {
    const newErrors: {
      name?: string;
      displayName?: string;
      category?: string;
      defaultVersion?: string;
    } = {};

    if (!name().trim()) {
      newErrors.name = t("components.form.name.required");
    } else if (!/^[a-z0-9-]+$/.test(name().trim())) {
      newErrors.name = t("components.form.name.invalid");
    }

    if (!displayName().trim()) {
      newErrors.displayName = t("components.form.displayName.required");
    }

    if (!category()) {
      newErrors.category = t("components.form.category.required");
    }

    // If pinned version rule, require a version
    if (defaultVersionRule() === "pinned" && !defaultVersion().trim()) {
      newErrors.defaultVersion = t("components.form.defaultVersion.required");
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: Event) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    const request: CreateComponentRequest = {
      name: name().trim(),
      category: category(),
      display_name: displayName().trim(),
      is_optional: isOptional(),
    };

    if (description().trim()) {
      request.description = description().trim();
    }
    if (artifactPattern().trim()) {
      request.artifact_pattern = artifactPattern().trim();
    }
    if (defaultUrlTemplate().trim()) {
      request.default_url_template = defaultUrlTemplate().trim();
    }
    if (githubNormalizedTemplate().trim()) {
      request.github_normalized_template = githubNormalizedTemplate().trim();
    }

    // Add version fields
    request.default_version_rule = defaultVersionRule();
    if (defaultVersion().trim()) {
      request.default_version = defaultVersion().trim();
    }

    props.onSubmit(request);
  };

  const isEditing = () => !!props.initialData;

  return (
    <form onSubmit={handleSubmit} class="flex flex-col gap-6">
      <div class="space-y-2">
        <label class="text-sm font-medium" for="component-name">
          {t("components.form.name.label")}{" "}
          <span class="text-destructive">*</span>
        </label>
        <input
          id="component-name"
          type="text"
          class={`w-full px-3 py-2 bg-background border-2 rounded-md focus:outline-none transition-colors font-mono text-sm ${
            errors().name
              ? "border-destructive focus:border-destructive"
              : "border-border focus:border-primary"
          }`}
          placeholder={t("components.form.name.placeholder")}
          value={name()}
          onInput={(e) => {
            setName(e.target.value);
            if (errors().name) {
              setErrors((prev) => ({ ...prev, name: undefined }));
            }
          }}
        />
        <p class="text-xs text-muted-foreground">
          {t("components.form.name.help")}
        </p>
        <Show when={errors().name}>
          <p class="text-xs text-destructive">{errors().name}</p>
        </Show>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="component-display-name">
          {t("components.form.displayName.label")}{" "}
          <span class="text-destructive">*</span>
        </label>
        <input
          id="component-display-name"
          type="text"
          class={`w-full px-3 py-2 bg-background border-2 rounded-md focus:outline-none transition-colors ${
            errors().displayName
              ? "border-destructive focus:border-destructive"
              : "border-border focus:border-primary"
          }`}
          placeholder={t("components.form.displayName.placeholder")}
          value={displayName()}
          onInput={(e) => {
            setDisplayName(e.target.value);
            if (errors().displayName) {
              setErrors((prev) => ({ ...prev, displayName: undefined }));
            }
          }}
        />
        <Show when={errors().displayName}>
          <p class="text-xs text-destructive">{errors().displayName}</p>
        </Show>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="component-category">
          {t("components.form.category.label")}{" "}
          <span class="text-destructive">*</span>
        </label>
        <select
          id="component-category"
          class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary transition-colors"
          value={category()}
          onChange={(e) => setCategory(e.target.value)}
        >
          <For each={COMPONENT_CATEGORIES}>
            {(cat) => (
              <option value={cat}>{getCategoryDisplayName(cat)}</option>
            )}
          </For>
        </select>
        <p class="text-xs text-muted-foreground">
          {t("components.form.category.help")}
        </p>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="component-description">
          {t("components.form.description.label")}
        </label>
        <textarea
          id="component-description"
          class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary transition-colors resize-none"
          placeholder={t("components.form.description.placeholder")}
          rows={3}
          value={description()}
          onInput={(e) => setDescription(e.target.value)}
        />
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="component-artifact-pattern">
          {t("components.form.artifactPattern.label")}
        </label>
        <input
          id="component-artifact-pattern"
          type="text"
          class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary transition-colors font-mono text-sm"
          placeholder={t("components.form.artifactPattern.placeholder")}
          value={artifactPattern()}
          onInput={(e) => setArtifactPattern(e.target.value)}
        />
        <p class="text-xs text-muted-foreground">
          {t("components.form.artifactPattern.help")}
        </p>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="component-default-url-template">
          {t("components.form.defaultUrlTemplate.label")}
        </label>
        <input
          id="component-default-url-template"
          type="text"
          class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary transition-colors font-mono text-sm"
          placeholder={t("components.form.defaultUrlTemplate.placeholder")}
          value={defaultUrlTemplate()}
          onInput={(e) => setDefaultUrlTemplate(e.target.value)}
        />
        <p class="text-xs text-muted-foreground">
          {t("components.form.defaultUrlTemplate.help")}
        </p>
      </div>

      <div class="space-y-2">
        <label
          class="text-sm font-medium"
          for="component-github-normalized-template"
        >
          {t("components.form.githubNormalizedTemplate.label")}
        </label>
        <input
          id="component-github-normalized-template"
          type="text"
          class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary transition-colors font-mono text-sm"
          placeholder={t(
            "components.form.githubNormalizedTemplate.placeholder",
          )}
          value={githubNormalizedTemplate()}
          onInput={(e) => setGithubNormalizedTemplate(e.target.value)}
        />
        <p class="text-xs text-muted-foreground">
          {t("components.form.githubNormalizedTemplate.help")}
        </p>
      </div>

      <div class="flex items-center gap-3">
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={isOptional()}
            onChange={(e) => setIsOptional(e.target.checked)}
            class="sr-only peer"
          />
          <div class="w-11 h-6 bg-muted rounded-full peer peer-checked:bg-primary peer-focus:ring-2 peer-focus:ring-primary/20 transition-colors after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-background after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full"></div>
        </label>
        <span class="text-sm font-medium">
          {t("components.form.isOptional.label")}
        </span>
      </div>
      <p class="text-xs text-muted-foreground -mt-4">
        {t("components.form.isOptional.help")}
      </p>

      {/* Default Version Section */}
      <div class="border-t border-border pt-6 mt-2">
        <h3 class="text-sm font-semibold mb-4">
          {t("components.form.versionSection.title")}
        </h3>

        <div class="space-y-4">
          <div class="space-y-2">
            <label class="text-sm font-medium" for="component-version-rule">
              {t("components.form.versionRule.label")}
            </label>
            <select
              id="component-version-rule"
              class="w-full px-3 py-2 bg-background border-2 border-border rounded-md focus:outline-none focus:border-primary transition-colors"
              value={defaultVersionRule()}
              onChange={(e) => {
                setDefaultVersionRule(e.target.value as VersionRule);
                if (errors().defaultVersion) {
                  setErrors((prev) => ({ ...prev, defaultVersion: undefined }));
                }
              }}
            >
              <For each={VERSION_RULES}>
                {(rule) => <option value={rule.value}>{rule.label}</option>}
              </For>
            </select>
            <p class="text-xs text-muted-foreground">
              {t("components.form.versionRule.help")}
            </p>
          </div>

          <Show when={defaultVersionRule() === "pinned"}>
            <div class="space-y-2">
              <label
                class="text-sm font-medium"
                for="component-default-version"
              >
                {t("components.form.defaultVersion.label")}{" "}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="component-default-version"
                type="text"
                class={`w-full px-3 py-2 bg-background border-2 rounded-md focus:outline-none transition-colors font-mono text-sm ${
                  errors().defaultVersion
                    ? "border-destructive focus:border-destructive"
                    : "border-border focus:border-primary"
                }`}
                placeholder={t("components.form.defaultVersion.placeholder")}
                value={defaultVersion()}
                onInput={(e) => {
                  setDefaultVersion(e.target.value);
                  if (errors().defaultVersion) {
                    setErrors((prev) => ({
                      ...prev,
                      defaultVersion: undefined,
                    }));
                  }
                }}
              />
              <p class="text-xs text-muted-foreground">
                {t("components.form.defaultVersion.help")}
              </p>
              <Show when={errors().defaultVersion}>
                <p class="text-xs text-destructive">
                  {errors().defaultVersion}
                </p>
              </Show>
            </div>
          </Show>
        </div>
      </div>

      <nav class="flex justify-end gap-3 pt-4 border-t border-border">
        <button
          type="button"
          onClick={props.onCancel}
          disabled={props.isSubmitting}
          class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
        >
          {t("common.actions.cancel")}
        </button>
        <button
          type="submit"
          disabled={props.isSubmitting}
          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <Show when={props.isSubmitting}>
            <Spinner size="sm" />
          </Show>
          <span>
            {isEditing()
              ? t("components.form.actions.update")
              : t("components.form.actions.add")}
          </span>
        </button>
      </nav>
    </form>
  );
};
