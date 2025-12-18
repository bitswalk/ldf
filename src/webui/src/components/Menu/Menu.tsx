import type { Component, JSX } from "solid-js";
import { createSignal, For, Show } from "solid-js";
import { Icon, type IconName } from "../Icon";

type MenuOrientation = "vertical" | "horizontal";

export interface MenuItem {
  id: string;
  label: string;
  icon: IconName;
  onClick: () => void;
}

interface MenuProps {
  orientation?: MenuOrientation;
  children?: JSX.Element;
  items?: MenuItem[];
  activeItemId?: string;
}

export const Menu: Component<MenuProps> = (props) => {
  const orientation = () => props.orientation ?? "vertical";
  const [isExpanded, setIsExpanded] = createSignal(false);

  const isVertical = () => orientation() === "vertical";

  const toggleExpanded = () => {
    setIsExpanded(!isExpanded());
  };

  return (
    <nav
      class="bg-card border-border flex shrink-0 z-[60]"
      classList={{
        "flex-col h-full border-r": isVertical(),
        "flex-row h-12 w-full border-b": !isVertical(),
        "w-12": isVertical() && !isExpanded(),
        "w-48 shadow-lg": isVertical() && isExpanded(),
      }}
      style={{ transition: "width 200ms ease-out" }}
    >
      {/* Menu Content */}
      <section
        class="flex-1 overflow-hidden"
        classList={{
          "flex flex-col": isVertical(),
          "flex flex-row": !isVertical(),
        }}
      >
        <Show when={props.items && props.items.length > 0}>
          <For each={props.items}>
            {(item) => (
              <button
                onClick={item.onClick}
                class="h-12 flex items-center text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors shrink-0"
                classList={{
                  "w-12 justify-center": !isExpanded(),
                  "w-48 justify-start px-4 gap-3": isExpanded(),
                  "text-primary bg-muted/30": props.activeItemId === item.id,
                }}
                title={item.label}
              >
                <Icon name={item.icon} size="md" />
                <Show when={isExpanded()}>
                  <span class="text-sm">{item.label}</span>
                </Show>
              </button>
            )}
          </For>
        </Show>
        {props.children}
      </section>

      {/* Toggle Button - Only for vertical orientation */}
      {isVertical() && (
        <button
          onClick={toggleExpanded}
          class="h-12 flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors shrink-0"
          classList={{
            "w-12": !isExpanded(),
            "w-48 justify-start px-4 gap-2": isExpanded(),
          }}
          title={isExpanded() ? "Collapse menu" : "Expand menu"}
        >
          <Icon name="sidebar-simple" size="md" />
          {isExpanded() && <span class="text-sm">Collapse</span>}
        </button>
      )}
    </nav>
  );
};
