import type { Component } from "solid-js";
import { createSignal, Show } from "solid-js";
import { Icon } from "../Icon";
import { Spinner } from "../Spinner";
import { t } from "../../services/i18n";
import type {
  BoardProfile,
  CreateBoardProfileRequest,
  UpdateBoardProfileRequest,
} from "../../services/boardProfiles";

interface BoardProfileFormProps {
  profile?: BoardProfile; // If provided, we're editing
  onSubmit: (
    data: CreateBoardProfileRequest | UpdateBoardProfileRequest,
  ) => void;
  onCancel: () => void;
  submitting: boolean;
  error: string | null;
}

export const BoardProfileForm: Component<BoardProfileFormProps> = (props) => {
  const isEditing = () => !!props.profile;

  const [name, setName] = createSignal(props.profile?.name ?? "");
  const [displayName, setDisplayName] = createSignal(
    props.profile?.display_name ?? "",
  );
  const [description, setDescription] = createSignal(
    props.profile?.description ?? "",
  );
  const [arch, setArch] = createSignal<"x86_64" | "aarch64">(
    props.profile?.arch ?? "x86_64",
  );
  const [kernelDefconfig, setKernelDefconfig] = createSignal(
    props.profile?.config?.kernel_defconfig ?? "",
  );
  const [kernelCmdline, setKernelCmdline] = createSignal(
    props.profile?.config?.kernel_cmdline ?? "",
  );

  const handleSubmit = (e: SubmitEvent) => {
    e.preventDefault();

    const config = {
      kernel_defconfig: kernelDefconfig() || undefined,
      kernel_cmdline: kernelCmdline() || undefined,
      kernel_overlay: props.profile?.config?.kernel_overlay,
      device_trees: props.profile?.config?.device_trees,
      boot_params: props.profile?.config?.boot_params,
      firmware: props.profile?.config?.firmware,
    };

    if (isEditing()) {
      const data: UpdateBoardProfileRequest = {};
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
        arch: arch(),
        config,
      } as CreateBoardProfileRequest);
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
      <div class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="bp-name">
          {t("boardProfiles.form.name.label")}
        </label>
        <input
          id="bp-name"
          type="text"
          value={name()}
          onInput={(e) => setName(e.currentTarget.value)}
          required
          placeholder={t("boardProfiles.form.name.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
        />
        <p class="text-xs text-muted-foreground">
          {t("boardProfiles.form.name.description")}
        </p>
      </div>

      {/* Display Name */}
      <div class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="bp-display-name">
          {t("boardProfiles.form.displayName.label")}
        </label>
        <input
          id="bp-display-name"
          type="text"
          value={displayName()}
          onInput={(e) => setDisplayName(e.currentTarget.value)}
          required
          placeholder={t("boardProfiles.form.displayName.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
        />
        <p class="text-xs text-muted-foreground">
          {t("boardProfiles.form.displayName.description")}
        </p>
      </div>

      {/* Description */}
      <div class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="bp-description">
          {t("boardProfiles.form.description.label")}
        </label>
        <textarea
          id="bp-description"
          value={description()}
          onInput={(e) => setDescription(e.currentTarget.value)}
          rows={2}
          placeholder={t("boardProfiles.form.description.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent resize-none"
        />
        <p class="text-xs text-muted-foreground">
          {t("boardProfiles.form.description.description")}
        </p>
      </div>

      {/* Architecture (only for create) */}
      <Show when={!isEditing()}>
        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium" for="bp-arch">
            {t("boardProfiles.form.arch.label")}
          </label>
          <select
            id="bp-arch"
            value={arch()}
            onChange={(e) =>
              setArch(e.target.value as "x86_64" | "aarch64")
            }
            class="px-3 py-2 rounded-md border border-border bg-background text-foreground cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="x86_64">x86_64</option>
            <option value="aarch64">aarch64</option>
          </select>
          <p class="text-xs text-muted-foreground">
            {t("boardProfiles.form.arch.description")}
          </p>
        </div>
      </Show>

      {/* Kernel Defconfig */}
      <div class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="bp-defconfig">
          {t("boardProfiles.form.kernelDefconfig.label")}
        </label>
        <input
          id="bp-defconfig"
          type="text"
          value={kernelDefconfig()}
          onInput={(e) => setKernelDefconfig(e.currentTarget.value)}
          placeholder={t("boardProfiles.form.kernelDefconfig.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono text-sm"
        />
        <p class="text-xs text-muted-foreground">
          {t("boardProfiles.form.kernelDefconfig.description")}
        </p>
      </div>

      {/* Kernel Command Line */}
      <div class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="bp-cmdline">
          {t("boardProfiles.form.kernelCmdline.label")}
        </label>
        <input
          id="bp-cmdline"
          type="text"
          value={kernelCmdline()}
          onInput={(e) => setKernelCmdline(e.currentTarget.value)}
          placeholder={t("boardProfiles.form.kernelCmdline.placeholder")}
          class="px-3 py-2 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono text-sm"
        />
        <p class="text-xs text-muted-foreground">
          {t("boardProfiles.form.kernelCmdline.description")}
        </p>
      </div>

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
            {isEditing() ? t("common.actions.save") : t("common.actions.create")}
          </span>
        </button>
      </nav>
    </form>
  );
};
