import type { Component } from "solid-js";
import { createSignal, Show } from "solid-js";
import { Modal } from "../Modal";
import { Icon } from "../Icon";
import { Spinner } from "../Spinner";
import {
  startBuild,
  type TargetArch,
  type ImageFormat,
  getArchDisplayText,
  getFormatDisplayText,
} from "../../services/builds";
import { t } from "../../services/i18n";

interface BuildStartDialogProps {
  isOpen: boolean;
  onClose: () => void;
  distributionId: string;
  distributionName: string;
  onBuildStarted?: (buildId: string) => void;
}

export const BuildStartDialog: Component<BuildStartDialogProps> = (props) => {
  const [arch, setArch] = createSignal<TargetArch>("x86_64");
  const [format, setFormat] = createSignal<ImageFormat>("raw");
  const [isSubmitting, setIsSubmitting] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const architectures: TargetArch[] = ["x86_64", "aarch64"];
  const formats: ImageFormat[] = ["raw", "qcow2", "iso"];

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    const result = await startBuild(props.distributionId, arch(), format());

    setIsSubmitting(false);

    if (result.success) {
      props.onBuildStarted?.(result.build.id);
      props.onClose();
    } else {
      setError(result.message);
    }
  };

  const handleClose = () => {
    if (!isSubmitting()) {
      setError(null);
      props.onClose();
    }
  };

  return (
    <Modal
      isOpen={props.isOpen}
      onClose={handleClose}
      title={t("build.startDialog.title")}
    >
      <form onSubmit={handleSubmit} class="space-y-6">
        {/* Distribution info */}
        <div class="p-4 bg-muted rounded-lg">
          <div class="text-sm text-muted-foreground">
            {t("build.startDialog.distribution")}
          </div>
          <div class="font-medium">{props.distributionName}</div>
        </div>

        {/* Architecture selection */}
        <div class="space-y-2">
          <label class="block text-sm font-medium">
            {t("build.startDialog.architecture")}
          </label>
          <div class="grid grid-cols-2 gap-3">
            {architectures.map((a) => (
              <button
                type="button"
                class={`p-3 rounded-lg border text-left transition-colors ${
                  arch() === a
                    ? "border-primary bg-primary/10"
                    : "border-border hover:border-muted-foreground"
                }`}
                onClick={() => setArch(a)}
              >
                <div class="font-medium">{a}</div>
                <div class="text-xs text-muted-foreground">
                  {a === "x86_64" ? "AMD64 / Intel 64-bit" : "ARM 64-bit"}
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* Format selection */}
        <div class="space-y-2">
          <label class="block text-sm font-medium">
            {t("build.startDialog.format")}
          </label>
          <div class="space-y-2">
            {formats.map((f) => (
              <button
                type="button"
                class={`w-full p-3 rounded-lg border text-left transition-colors ${
                  format() === f
                    ? "border-primary bg-primary/10"
                    : "border-border hover:border-muted-foreground"
                }`}
                onClick={() => setFormat(f)}
              >
                <div class="flex items-center gap-3">
                  <Icon
                    name={
                      f === "iso"
                        ? "disc"
                        : f === "qcow2"
                          ? "hard-drives"
                          : "file"
                    }
                    size="sm"
                    class="text-muted-foreground"
                  />
                  <div>
                    <div class="font-medium">{getFormatDisplayText(f)}</div>
                    <div class="text-xs text-muted-foreground">
                      {f === "raw" && t("build.startDialog.formatDesc.raw")}
                      {f === "qcow2" && t("build.startDialog.formatDesc.qcow2")}
                      {f === "iso" && t("build.startDialog.formatDesc.iso")}
                    </div>
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* Error message */}
        <Show when={error()}>
          <div class="p-3 bg-destructive/10 border border-destructive rounded-lg text-destructive text-sm">
            {error()}
          </div>
        </Show>

        {/* Actions */}
        <div class="flex gap-3 pt-4 border-t border-border">
          <button
            type="button"
            class="flex-1 px-4 py-2 rounded-md border border-border hover:bg-muted transition-colors"
            onClick={handleClose}
            disabled={isSubmitting()}
          >
            {t("common.actions.cancel")}
          </button>
          <button
            type="submit"
            class="flex-1 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            disabled={isSubmitting()}
          >
            <Show when={isSubmitting()} fallback={<Icon name="hammer" size="sm" />}>
              <Spinner size="sm" />
            </Show>
            {isSubmitting()
              ? t("build.startDialog.starting")
              : t("build.startDialog.start")}
          </button>
        </div>
      </form>
    </Modal>
  );
};
