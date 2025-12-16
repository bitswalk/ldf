import type { Component, JSX } from "solid-js";
import { mergeProps, splitProps } from "solid-js";

interface SkeletonProps extends JSX.HTMLAttributes<HTMLElement> {
  /**
   * The semantic HTML tag to use for the skeleton element.
   * Defaults to "div" but should be set to the most appropriate semantic tag
   * (e.g., "article", "section", "header", "nav", "aside", "footer", "main")
   */
  as?: keyof JSX.IntrinsicElements;

  /**
   * Optional custom class names to apply
   */
  class?: string;
}

export const Skeleton: Component<SkeletonProps> = (props) => {
  const merged = mergeProps({ as: "div" as const }, props);
  const [local, others] = splitProps(merged, ["as", "class"]);

  const baseClass = "animate-pulse rounded-md bg-muted";
  const combinedClass = local.class ? `${baseClass} ${local.class}` : baseClass;

  // Dynamically create the element based on the 'as' prop
  const Element = local.as as any;

  return <Element class={combinedClass} {...others} />;
};
