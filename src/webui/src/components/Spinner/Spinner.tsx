import type { Component, JSX } from "solid-js";
import { mergeProps, splitProps } from "solid-js";
import { Icon } from "../Icon";

interface SpinnerProps extends JSX.HTMLAttributes<HTMLOrSVGElement> {
  class?: string;
  size?: "sm" | "md" | "lg" | "xl";
}

export const Spinner: Component<SpinnerProps> = (props) => {
  const merged = mergeProps({ size: "md" as const }, props);
  const [local, others] = splitProps(merged, ["class", "size"]);

  const sizeMap = {
    sm: "xs" as const,
    md: "sm" as const,
    lg: "md" as const,
    xl: "lg" as const,
  };

  return (
    <Icon
      name="hourglass"
      size={sizeMap[local.size]}
      class={`animate-spin ${local.class || ""}`}
      role="status"
      aria-label="Loading"
      {...others}
    />
  );
};
