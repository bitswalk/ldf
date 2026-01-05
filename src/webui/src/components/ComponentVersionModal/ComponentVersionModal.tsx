import type { Component as SolidComponent } from "solid-js";
import { createSignal, createResource, Show, For } from "solid-js";
import { Modal } from "../Modal";
import { Spinner } from "../Spinner";
import { Icon } from "../Icon";
import {
  getComponentVersions,
  resolveVersionRule,
  getVersionTypeLabel,
  type Component,
  type SourceVersion,
  type VersionRule,
} from "../../services/components";
import { t } from "../../services/i18n";

interface ComponentVersionModalProps {
  isOpen: boolean;
  onClose: () => void;
  component: Component;
  currentVersion?: string;
  onSelectVersion: (version: string | undefined) => void;
}

export const ComponentVersionModal: SolidComponent<
  ComponentVersionModalProps
> = (props) => {
  const [selectedVersion, setSelectedVersion] = createSignal<string | undefined>(
    props.currentVersion
  );
  const [useDefault, setUseDefault] = createSignal(!props.currentVersion);
  const [searchQuery, setSearchQuery] = createSignal("");
  const [versionFilter, setVersionFilter] = createSignal<string>("all");
  const [offset, setOffset] = createSignal(0);
  const limit = 20;

  // Fetch versions for this component
  const [versions] = createResource(
    () => ({
      componentId: props.component.id,
      versionType: versionFilter(),
      offset: offset(),
    }),
    async ({ componentId, versionType, offset }) => {
      const result = await getComponentVersions(componentId, {
        limit,
        offset,
        version_type: versionType as "all" | "stable" | "longterm" | "mainline",
      });
      if (result.success) {
        return result.data;
      }
      return { versions: [], total: 0, limit, offset };
    }
  );

  // Resolve the default version based on the component's rule
  const [resolvedDefault] = createResource(
    () => props.component,
    async (component) => {
      if (!component.default_version_rule) return null;
      if (component.default_version_rule === "pinned") {
        return component.default_version || null;
      }
      const result = await resolveVersionRule(
        component.id,
        component.default_version_rule
      );
      if (result.success) {
        return result.data.resolved_version;
      }
      return null;
    }
  );

  const filteredVersions = () => {
    const query = searchQuery().toLowerCase().trim();
    const versionsList = versions()?.versions || [];
    if (!query) return versionsList;
    return versionsList.filter((v) => v.version.toLowerCase().includes(query));
  };

  const handleApply = () => {
    if (useDefault()) {
      props.onSelectVersion(undefined);
    } else {
      props.onSelectVersion(selectedVersion());
    }
    props.onClose();
  };

  const handleSelectVersion = (version: string) => {
    setUseDefault(false);
    setSelectedVersion(version);
  };

  const handleUseDefault = () => {
    setUseDefault(true);
    setSelectedVersion(undefined);
  };

  const getVersionRuleDisplay = (rule: VersionRule | undefined): string => {
    switch (rule) {
      case "latest-stable":
        return t("distribution.versionModal.ruleLatestStable");
      case "latest-lts":
        return t("distribution.versionModal.ruleLatestLts");
      case "pinned":
        return t("distribution.versionModal.rulePinned");
      default:
        return t("distribution.versionModal.ruleLatestStable");
    }
  };

  return (
    <Modal
      isOpen={props.isOpen}
      onClose={props.onClose}
      title={t("distribution.versionModal.title", {
        component: props.component.display_name,
      })}
    >
      <div class="flex flex-col gap-4 max-h-[70vh]">
        {/* Default version option */}
        <div
          class={`p-4 border-2 rounded-lg cursor-pointer transition-colors ${
            useDefault()
              ? "border-primary bg-primary/5"
              : "border-border hover:border-muted-foreground"
          }`}
          onClick={handleUseDefault}
        >
          <div class="flex items-center gap-3">
            <div
              class={`w-5 h-5 rounded-full border-2 flex items-center justify-center ${
                useDefault() ? "border-primary" : "border-muted-foreground"
              }`}
            >
              <Show when={useDefault()}>
                <div class="w-2.5 h-2.5 rounded-full bg-primary" />
              </Show>
            </div>
            <div class="flex-1">
              <div class="font-medium">
                {t("distribution.versionModal.useDefault")}
              </div>
              <div class="text-sm text-muted-foreground">
                {getVersionRuleDisplay(props.component.default_version_rule)}
                <Show when={resolvedDefault()}>
                  {" "}
                  <span class="font-mono">({resolvedDefault()})</span>
                </Show>
                <Show when={resolvedDefault.loading}>
                  <span class="ml-2">
                    <Spinner size="sm" />
                  </span>
                </Show>
              </div>
            </div>
          </div>
        </div>

        {/* Separator */}
        <div class="flex items-center gap-3">
          <div class="flex-1 border-t border-border" />
          <span class="text-sm text-muted-foreground">
            {t("distribution.versionModal.orSelectSpecific")}
          </span>
          <div class="flex-1 border-t border-border" />
        </div>

        {/* Search and filter */}
        <div class="flex gap-2">
          <div class="flex-1 relative">
            <Icon
              name="search"
              size="sm"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <input
              type="text"
              placeholder={t("distribution.versionModal.searchPlaceholder")}
              class="w-full pl-9 pr-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary text-sm"
              value={searchQuery()}
              onInput={(e) => setSearchQuery(e.target.value)}
            />
          </div>
          <select
            class="px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:border-primary text-sm"
            value={versionFilter()}
            onChange={(e) => {
              setVersionFilter(e.target.value);
              setOffset(0);
            }}
          >
            <option value="all">
              {t("distribution.versionModal.filterAll")}
            </option>
            <option value="stable">
              {t("distribution.versionModal.filterStable")}
            </option>
            <option value="longterm">
              {t("distribution.versionModal.filterLts")}
            </option>
            <option value="mainline">
              {t("distribution.versionModal.filterMainline")}
            </option>
          </select>
        </div>

        {/* Version list */}
        <div class="flex-1 overflow-y-auto min-h-[200px] max-h-[300px] border border-border rounded-md">
          <Show
            when={!versions.loading}
            fallback={
              <div class="flex items-center justify-center py-8">
                <Spinner size="md" />
              </div>
            }
          >
            <Show
              when={filteredVersions().length > 0}
              fallback={
                <div class="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <Icon name="package" size="lg" class="mb-2" />
                  <p class="text-sm">
                    {t("distribution.versionModal.noVersions")}
                  </p>
                </div>
              }
            >
              <div class="divide-y divide-border">
                <For each={filteredVersions()}>
                  {(version) => (
                    <div
                      class={`flex items-center gap-3 p-3 cursor-pointer transition-colors ${
                        !useDefault() && selectedVersion() === version.version
                          ? "bg-primary/10"
                          : "hover:bg-muted"
                      }`}
                      onClick={() => handleSelectVersion(version.version)}
                    >
                      <div
                        class={`w-4 h-4 rounded-full border-2 flex items-center justify-center ${
                          !useDefault() &&
                          selectedVersion() === version.version
                            ? "border-primary"
                            : "border-muted-foreground"
                        }`}
                      >
                        <Show
                          when={
                            !useDefault() &&
                            selectedVersion() === version.version
                          }
                        >
                          <div class="w-2 h-2 rounded-full bg-primary" />
                        </Show>
                      </div>
                      <div class="flex-1">
                        <span class="font-mono text-sm">{version.version}</span>
                      </div>
                      <span
                        class={`text-xs px-2 py-0.5 rounded-full ${
                          version.version_type === "longterm"
                            ? "bg-green-500/10 text-green-600"
                            : version.version_type === "stable"
                              ? "bg-blue-500/10 text-blue-600"
                              : "bg-orange-500/10 text-orange-600"
                        }`}
                      >
                        {getVersionTypeLabel(version.version_type)}
                      </span>
                      <Show when={version.release_date}>
                        <span class="text-xs text-muted-foreground">
                          {new Date(version.release_date!).toLocaleDateString()}
                        </span>
                      </Show>
                    </div>
                  )}
                </For>
              </div>
            </Show>
          </Show>
        </div>

        {/* Pagination */}
        <Show when={(versions()?.total || 0) > limit}>
          <div class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">
              {t("distribution.versionModal.showing", {
                start: offset() + 1,
                end: Math.min(offset() + limit, versions()?.total || 0),
                total: versions()?.total || 0,
              })}
            </span>
            <div class="flex gap-2">
              <button
                type="button"
                disabled={offset() === 0}
                onClick={() => setOffset((prev) => Math.max(0, prev - limit))}
                class="px-3 py-1 text-sm border border-border rounded-md hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t("distribution.versionModal.previous")}
              </button>
              <button
                type="button"
                disabled={offset() + limit >= (versions()?.total || 0)}
                onClick={() => setOffset((prev) => prev + limit)}
                class="px-3 py-1 text-sm border border-border rounded-md hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t("distribution.versionModal.next")}
              </button>
            </div>
          </div>
        </Show>

        {/* Actions */}
        <div class="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={props.onClose}
            class="px-4 py-2 text-sm border border-border rounded-md hover:bg-muted transition-colors"
          >
            {t("common.actions.cancel")}
          </button>
          <button
            type="button"
            onClick={handleApply}
            class="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
          >
            {t("distribution.versionModal.apply")}
          </button>
        </div>
      </div>
    </Modal>
  );
};
