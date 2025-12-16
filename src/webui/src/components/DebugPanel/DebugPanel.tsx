import type { Component } from "solid-js";
import { createSignal, onMount, Show } from "solid-js";
import { Icon } from "../Icon";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../DropdownMenu";

type ViewType =
  | "server-connection"
  | "distribution"
  | "login"
  | "register"
  | "settings";

interface DebugPanelProps {
  isLoggedIn: boolean;
  onToggleLogin: () => void;
  currentView: string;
  onViewChange?: (view: ViewType) => void;
}

export const DebugPanel: Component<DebugPanelProps> = (props) => {
  const [isDevMode, setIsDevMode] = createSignal(false);

  onMount(() => {
    const devMode = localStorage.getItem("dev-mode");
    setIsDevMode(devMode === "true");

    // Listen for storage changes from other tabs/windows
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "dev-mode") {
        setIsDevMode(e.newValue === "true");
      }
    };
    window.addEventListener("storage", handleStorage);
  });

  return (
    <Show when={isDevMode()}>
      <footer class="fixed bottom-2 left-2 right-2 min-h-[32px] bg-accent/50 border border-accent-foreground/20 rounded flex items-center justify-between px-4 py-2 text-xs font-mono z-[9999] backdrop-blur-sm">
        <section class="flex items-center gap-4">
          <span class="text-primary font-bold">DEV MODE</span>
          <DropdownMenu>
            <DropdownMenuTrigger class="text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1">
              View: <span class="text-foreground">{props.currentView}</span>
              <Icon name="caret-down" size="xs" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" side="top">
              <DropdownMenuItem
                onSelect={() => props.onViewChange?.("server-connection")}
                class="gap-2"
              >
                <Icon name="plug" size="sm" />
                <span>server-connection</span>
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => props.onViewChange?.("login")}
                class="gap-2"
              >
                <Icon name="sign-in" size="sm" />
                <span>login</span>
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => props.onViewChange?.("register")}
                class="gap-2"
              >
                <Icon name="user-plus" size="sm" />
                <span>register</span>
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => props.onViewChange?.("distribution")}
                class="gap-2"
              >
                <Icon name="package" size="sm" />
                <span>distribution</span>
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => props.onViewChange?.("settings")}
                class="gap-2"
              >
                <Icon name="gear" size="sm" />
                <span>settings</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <span class="text-muted-foreground">
            Auth:{" "}
            <span
              class={props.isLoggedIn ? "text-primary" : "text-destructive"}
            >
              {props.isLoggedIn ? "Logged In" : "Anonymous"}
            </span>
          </span>
        </section>
        <section class="flex items-center gap-2">
          <button
            onClick={props.onToggleLogin}
            class="px-2 py-0.5 bg-primary/20 hover:bg-primary/30 border border-primary/50 rounded text-primary transition-colors"
            title="Toggle login state"
          >
            Toggle Auth
          </button>
          <button
            onClick={() => {
              localStorage.removeItem("dev-mode");
              setIsDevMode(false);
            }}
            class="px-2 py-0.5 bg-destructive/20 hover:bg-destructive/30 border border-destructive/50 rounded text-destructive-foreground transition-colors"
            title="Disable dev mode"
          >
            <Icon name="x" size="sm" />
          </button>
        </section>
      </footer>
    </Show>
  );
};
