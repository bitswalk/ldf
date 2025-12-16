import type { Component, JSX } from "solid-js";
import { Show } from "solid-js";

interface CardProps {
  children: JSX.Element;
  borderStyle?: "solid" | "dashed";
  header?: {
    title?: string;
    description?: string;
  };
  footer?: JSX.Element;
}

export const Card: Component<CardProps> = (props) => {
  const borderClass = () =>
    props.borderStyle === "dashed" ? "border-dashed" : "border-solid";

  return (
    <article
      class={`border-2 ${borderClass()} border-border rounded-lg bg-card text-card-foreground`}
    >
      <Show when={props.header}>
        <header class={`px-6 py-4 border-b ${borderClass()} border-border`}>
          <Show when={props.header?.title}>
            <h2 class="text-xl font-bold mb-1">{props.header?.title}</h2>
          </Show>
          <Show when={props.header?.description}>
            <p class="text-sm text-muted-foreground">
              {props.header?.description}
            </p>
          </Show>
        </header>
      </Show>

      <section class="p-6">{props.children}</section>

      <Show when={props.footer}>
        <footer class={`px-6 py-4 border-t ${borderClass()} border-border`}>
          {props.footer}
        </footer>
      </Show>
    </article>
  );
};
