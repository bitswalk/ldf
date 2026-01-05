import type { Component, JSX } from "solid-js";
import {
  mergeProps,
  splitProps,
  createResource,
  Show,
  createEffect,
} from "solid-js";

interface IconProps extends JSX.HTMLAttributes<HTMLOrSVGElement> {
  name: string;
  class?: string;
  size?: "xs" | "sm" | "md" | "lg" | "xl" | "2xl";
}

const loadIcon = async (name: string): Promise<string> => {
  const response = await fetch(`/icons/${name}.svg`);
  if (!response.ok) {
    throw new Error(`Failed to load icon: ${name}`);
  }
  return response.text();
};

export const Icon: Component<IconProps> = (props) => {
  const merged = mergeProps({ size: "md" as const }, props);
  const [local, others] = splitProps(merged, ["name", "class", "size"]);

  const [svgContent] = createResource(() => local.name, loadIcon);
  let iconRef: HTMLElement | undefined;

  const sizeClasses = {
    xs: "w-3 h-3",
    sm: "w-4 h-4",
    md: "w-5 h-5",
    lg: "w-6 h-6",
    xl: "w-8 h-8",
    "2xl": "w-12 h-12",
  };

  createEffect(() => {
    const svg = svgContent();
    if (iconRef && svg) {
      iconRef.innerHTML = svg;
      const svgElement = iconRef.querySelector("svg");
      if (svgElement) {
        svgElement.removeAttribute("width");
        svgElement.removeAttribute("height");
      }
    }
  });

  return (
    <Show
      when={!svgContent.loading}
      fallback={
        <i class={`inline-block flex-shrink-0 ${sizeClasses[local.size]}`} />
      }
    >
      <i
        ref={iconRef}
        class={`icon-container ${sizeClasses[local.size]} ${local.class || ""}`}
        {...others}
      />
    </Show>
  );
};
