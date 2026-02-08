import type { Component } from "solid-js";
import { Icon } from "../Icon";

interface NotificationProps {
  type: "success" | "error";
  message: string;
}

export const Notification: Component<NotificationProps> = (props) => {
  return (
    <aside
      class={`p-3 rounded-md ${
        props.type === "success"
          ? "bg-green-500/10 border border-green-500/20 text-green-500"
          : "bg-red-500/10 border border-red-500/20 text-red-500"
      }`}
    >
      <output class="flex items-center gap-2">
        <Icon
          name={props.type === "success" ? "check-circle" : "warning-circle"}
          size="md"
        />
        <span>{props.message}</span>
      </output>
    </aside>
  );
};
