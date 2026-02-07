import type { Component, JSX } from "solid-js";

export type LabelVariant =
  | "muted"
  | "success"
  | "danger"
  | "warning"
  | "primary";

interface LabelProps {
  variant?: LabelVariant;
  children: JSX.Element;
  class?: string;
}

const variantClasses: Record<LabelVariant, string> = {
  muted: "bg-muted text-muted-foreground",
  success: "text-green-500 bg-current/10",
  danger: "text-red-500 bg-current/10",
  warning: "text-yellow-500 bg-current/10",
  primary: "text-primary bg-current/10",
};

export const Label: Component<LabelProps> = (props) => {
  return (
    <span
      class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${variantClasses[props.variant || "muted"]} ${props.class || ""}`}
    >
      {props.children}
    </span>
  );
};
