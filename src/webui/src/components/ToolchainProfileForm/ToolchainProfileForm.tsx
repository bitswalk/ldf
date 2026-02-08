import type { Component } from "solid-js";
import { createSignal, Show } from "solid-js";
import { Spinner } from "../Spinner";
import { t } from "../../services/i18n";
import type {
  ToolchainProfile,
  CreateToolchainProfileRequest,
  UpdateToolchainProfileRequest,
} from "../../services/toolchainProfiles";

interface ToolchainProfileFormProps {
  profile?: ToolchainProfile;
  onSubmit: (
    data: CreateToolchainProfileRequest | UpdateToolchainProfileRequest,
  ) => void;
  onCancel: () => void;
  submitting: boolean;
  error: string | null;
}

/**
 * Parse KEY=VALUE lines into a map, ignoring blank lines and comments.
 */
function parseEnvLines(text: string): Record<string, string> {
  const env: Record<string, string> = {};
  for (const line of text.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const idx = trimmed.indexOf("=");
    if (idx > 0) {
      env[trimmed.slice(0, idx)] = trimmed.slice(idx + 1);
    }
  }
  return env;
}

/**
 * Serialize a map to KEY=VALUE lines.
 */
function envToLines(env?: Record<string, string>): string {
  if (!env || Object.keys(env).length === 0) return "";
  return Object.entries(env)
    .map(([k, v]) => `${k}=${v}`)
    .join("\n");
}

export const ToolchainProfileForm: Component<ToolchainProfileFormProps> = (
  props,
) => {
  const isEditing = () => !!props.profile;

  const [name, setName] = createSignal(props.profile?.name ?? "");
  const [displayName, setDisplayName] = createSignal(
    props.profile?.display_name ?? "",
  );
  const [description, setDescription] = createSignal(
    props.profile?.description ?? "",
  );
  const [toolchainType, setToolchainType] = createSignal<"gcc" | "llvm">(
    props.profile?.type ?? "gcc",
  );
  const [crossCompilePrefix, setCrossCompilePrefix] = createSignal(
    props.profile?.config?.cross_compile_prefix ?? "",
  );
  const [extraEnvText, setExtraEnvText] = createSignal(
    envToLines(props.profile?.config?.extra_env),
  );
  const [compilerFlags, setCompilerFlags] = createSignal(
    props.profile?.config?.compiler_flags ?? "",
  );

  const handleSubmit = (e: SubmitEvent) => {
    e.preventDefault();

    const extraEnv = parseEnvLines(extraEnvText());
    const config = {
      cross_compile_prefix: crossCompilePrefix() || undefined,
      extra_env: Object.keys(extraEnv).length > 0 ? extraEnv : undefined,
      compiler_flags: compilerFlags() || undefined,
    };

    if (isEditing()) {
      const data: UpdateToolchainProfileRequest = {};
      if (name() !== props.profile!.name) data.name = name();
      if (displayName() !== props.profile!.display_name)
        data.display_name = displayName();
      if (description() !== props.profile!.description)
        data.description = description();
      data.config = config;
      props.onSubmit(data);
    } else {
      props.onSubmit({
        name: name(),
        display_name: displayName(),
        description: description() || undefined,
        type: toolchainType(),
        config,
      } as CreateToolchainProfileRequest);
    }
  };

  return (
    <form onSubmit={handleSubmit} class="flex flex-col gap-5">
      <Show when={props.error}>
        <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
          {props.error}
        </aside>
      </Show>

      {/* Name */}
      <fieldset class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="tp-name">
          {t("toolchainProfiles.form.name.label")}
        </label>
        <input
          id="tp-name"
          type="text"
          value={name()}
          onInput={(e) => setName(e.currentTarget.value)}
          required
          placeholder={t("toolchainProfiles.form.name.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
        />
        <p class="text-xs text-muted-foreground">
          {t("toolchainProfiles.form.name.description")}
        </p>
      </fieldset>

      {/* Display Name */}
      <fieldset class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="tp-display-name">
          {t("toolchainProfiles.form.displayName.label")}
        </label>
        <input
          id="tp-display-name"
          type="text"
          value={displayName()}
          onInput={(e) => setDisplayName(e.currentTarget.value)}
          required
          placeholder={t("toolchainProfiles.form.displayName.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
        />
        <p class="text-xs text-muted-foreground">
          {t("toolchainProfiles.form.displayName.description")}
        </p>
      </fieldset>

      {/* Description */}
      <fieldset class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="tp-description">
          {t("toolchainProfiles.form.description.label")}
        </label>
        <textarea
          id="tp-description"
          value={description()}
          onInput={(e) => setDescription(e.currentTarget.value)}
          rows={2}
          placeholder={t("toolchainProfiles.form.description.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent resize-none"
        />
        <p class="text-xs text-muted-foreground">
          {t("toolchainProfiles.form.description.description")}
        </p>
      </fieldset>

      {/* Type (only for create) */}
      <Show when={!isEditing()}>
        <fieldset class="flex flex-col gap-1.5">
          <label class="text-sm font-medium" for="tp-type">
            {t("toolchainProfiles.form.type.label")}
          </label>
          <select
            id="tp-type"
            value={toolchainType()}
            onChange={(e) =>
              setToolchainType(e.target.value as "gcc" | "llvm")
            }
            class="px-3 py-2 rounded-md border border-border bg-background text-foreground cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="gcc">GCC</option>
            <option value="llvm">LLVM/Clang</option>
          </select>
          <p class="text-xs text-muted-foreground">
            {t("toolchainProfiles.form.type.description")}
          </p>
        </fieldset>
      </Show>

      {/* Cross-Compile Prefix */}
      <fieldset class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="tp-cross-prefix">
          {t("toolchainProfiles.form.crossCompilePrefix.label")}
        </label>
        <input
          id="tp-cross-prefix"
          type="text"
          value={crossCompilePrefix()}
          onInput={(e) => setCrossCompilePrefix(e.currentTarget.value)}
          placeholder={t(
            "toolchainProfiles.form.crossCompilePrefix.placeholder",
          )}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono text-sm"
        />
        <p class="text-xs text-muted-foreground">
          {t("toolchainProfiles.form.crossCompilePrefix.description")}
        </p>
      </fieldset>

      {/* Extra Environment Variables */}
      <fieldset class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="tp-extra-env">
          {t("toolchainProfiles.form.extraEnv.label")}
        </label>
        <textarea
          id="tp-extra-env"
          value={extraEnvText()}
          onInput={(e) => setExtraEnvText(e.currentTarget.value)}
          rows={3}
          placeholder={t("toolchainProfiles.form.extraEnv.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent resize-none font-mono text-sm"
        />
        <p class="text-xs text-muted-foreground">
          {t("toolchainProfiles.form.extraEnv.description")}
        </p>
      </fieldset>

      {/* Compiler Flags */}
      <fieldset class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="tp-compiler-flags">
          {t("toolchainProfiles.form.compilerFlags.label")}
        </label>
        <input
          id="tp-compiler-flags"
          type="text"
          value={compilerFlags()}
          onInput={(e) => setCompilerFlags(e.currentTarget.value)}
          placeholder={t("toolchainProfiles.form.compilerFlags.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono text-sm"
        />
        <p class="text-xs text-muted-foreground">
          {t("toolchainProfiles.form.compilerFlags.description")}
        </p>
      </fieldset>

      {/* Actions */}
      <nav class="flex justify-end gap-3 pt-2">
        <button
          type="button"
          onClick={props.onCancel}
          disabled={props.submitting}
          class="px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors disabled:opacity-50"
        >
          {t("common.actions.cancel")}
        </button>
        <button
          type="submit"
          disabled={props.submitting || !name() || !displayName()}
          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <Show when={props.submitting}>
            <Spinner size="sm" />
          </Show>
          <span>
            {isEditing()
              ? t("common.actions.save")
              : t("common.actions.create")}
          </span>
        </button>
      </nav>
    </form>
  );
};
