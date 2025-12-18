import type { Component, JSX } from "solid-js";
import {
  createSignal,
  onMount,
  onCleanup,
  Show,
  For,
  createEffect,
  on,
} from "solid-js";
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

type LogLevel = "info" | "warn" | "error" | "success" | "debug";

interface ConsoleLogEntry {
  id: string;
  timestamp: Date;
  level: LogLevel;
  message: string;
  details?: string;
}

interface ConsoleProps {
  isLoggedIn: boolean;
  onToggleLogin: () => void;
  currentView: string;
  onViewChange?: (view: ViewType) => void;
}

// Helper to format timestamp as HH:MM:SS
const formatTime = (date: Date): string => {
  return date.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

// Helper to generate unique IDs
const generateId = (): string => {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
};

// Log level badge styling - using theme-compatible colors
const getLevelBadgeStyles = (level: LogLevel): string => {
  const baseStyles =
    "px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider";
  switch (level) {
    case "success":
      return `${baseStyles} bg-primary/20 text-primary border border-primary/30`;
    case "info":
      return `${baseStyles} bg-secondary text-secondary-foreground border border-border`;
    case "warn":
      return `${baseStyles} bg-primary/30 text-primary border border-primary/50`;
    case "error":
      return `${baseStyles} bg-destructive/20 text-destructive border border-destructive/30`;
    case "debug":
      return `${baseStyles} bg-muted text-muted-foreground border border-border`;
    default:
      return `${baseStyles} bg-muted text-muted-foreground`;
  }
};

const getLevelLabel = (level: LogLevel): string => {
  switch (level) {
    case "success":
      return "COMPLETED";
    case "info":
      return "INFO";
    case "warn":
      return "WARNING";
    case "error":
      return "ERROR";
    case "debug":
      return "DEBUG";
    default:
      return level.toUpperCase();
  }
};

// Constants for resize behavior
const HEADER_HEIGHT = 48; // h-12 (48px) of the header
const MAX_CONSOLE_HEIGHT_VH = 30; // 30vh max height
const MIN_DRAG_THRESHOLD = 50; // px - if dragged below this, collapse

export const Console: Component<ConsoleProps> = (props) => {
  const [isDevMode, setIsDevMode] = createSignal(false);
  const [isExpanded, setIsExpanded] = createSignal(false);
  const [logs, setLogs] = createSignal<ConsoleLogEntry[]>([]);
  const [newestFirst, setNewestFirst] = createSignal(true);
  const [consoleHeight, setConsoleHeight] = createSignal(0); // 0 means use default (30vh)
  const [isDragging, setIsDragging] = createSignal(false);

  // Get logs in the correct order based on user preference
  const orderedLogs = () => {
    const entries = logs();
    return newestFirst() ? [...entries].reverse() : entries;
  };

  const toggleOrder = () => {
    setNewestFirst(!newestFirst());
  };

  // Calculate max height in pixels
  const getMaxHeightPx = () => {
    return (window.innerHeight * MAX_CONSOLE_HEIGHT_VH) / 100;
  };

  // Get current console height (either custom or default)
  const getCurrentHeight = () => {
    if (!isExpanded()) return 0;
    const height = consoleHeight();
    return height > 0 ? height : getMaxHeightPx();
  };

  // Resize handlers
  const handleDragStart = (e: MouseEvent) => {
    e.preventDefault();
    setIsDragging(true);

    // Initialize height if not set
    if (consoleHeight() === 0) {
      setConsoleHeight(getMaxHeightPx());
    }
  };

  const handleDrag = (e: MouseEvent) => {
    if (!isDragging()) return;

    const windowHeight = window.innerHeight;
    const newHeight = windowHeight - e.clientY - HEADER_HEIGHT;
    const maxHeight = getMaxHeightPx();

    // Clamp between min threshold and max height
    if (newHeight < MIN_DRAG_THRESHOLD) {
      // Below threshold - will collapse on mouse up
      setConsoleHeight(MIN_DRAG_THRESHOLD);
    } else if (newHeight > maxHeight) {
      setConsoleHeight(maxHeight);
    } else {
      setConsoleHeight(newHeight);
    }
  };

  const handleDragEnd = () => {
    if (!isDragging()) return;
    setIsDragging(false);

    // If height is at or below threshold, collapse the console
    if (consoleHeight() <= MIN_DRAG_THRESHOLD) {
      setIsExpanded(false);
      setConsoleHeight(0); // Reset to default for next open
    }
  };

  // Set up global mouse listeners for drag
  onMount(() => {
    window.addEventListener("mousemove", handleDrag);
    window.addEventListener("mouseup", handleDragEnd);
  });

  onCleanup(() => {
    window.removeEventListener("mousemove", handleDrag);
    window.removeEventListener("mouseup", handleDragEnd);
  });

  // Add a log entry
  const addLog = (level: LogLevel, message: string, details?: string) => {
    const entry: ConsoleLogEntry = {
      id: generateId(),
      timestamp: new Date(),
      level,
      message,
      details,
    };
    setLogs((prev) => [...prev, entry]);
  };

  // Clear all logs
  const clearLogs = () => {
    setLogs([]);
  };

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

    // Listen for devmode changes from the same tab (custom event)
    const handleDevModeChange = (e: Event) => {
      const customEvent = e as CustomEvent<{ enabled: boolean }>;
      setIsDevMode(customEvent.detail.enabled);
      if (customEvent.detail.enabled) {
        addLog("success", "Dev console enabled.");
      }
    };
    window.addEventListener("devmode-change", handleDevModeChange);

    // Add initial log entries for demo
    if (devMode === "true") {
      addLog("success", "Dev console initialized successfully.");
      addLog("info", `Current view: ${props.currentView}`);
    }

    onCleanup(() => {
      window.removeEventListener("storage", handleStorage);
      window.removeEventListener("devmode-change", handleDevModeChange);
    });
  });

  // Log view changes - use `on` to explicitly track only currentView
  createEffect(
    on(
      () => props.currentView,
      (view, prevView) => {
        if (isDevMode() && prevView !== undefined && view !== prevView) {
          addLog("info", `View changed to: ${view}`);
        }
      },
    ),
  );

  // Log auth state changes - use `on` to explicitly track only isLoggedIn
  createEffect(
    on(
      () => props.isLoggedIn,
      (loggedIn, prevLoggedIn) => {
        if (
          isDevMode() &&
          prevLoggedIn !== undefined &&
          loggedIn !== prevLoggedIn
        ) {
          addLog(
            loggedIn ? "success" : "warn",
            `Auth state changed: ${loggedIn ? "Logged In" : "Anonymous"}`,
          );
        }
      },
    ),
  );

  const handleDisableDevMode = () => {
    localStorage.removeItem("dev-mode");
    setIsDevMode(false);
  };

  const toggleExpanded = () => {
    setIsExpanded(!isExpanded());
  };

  return (
    <Show when={isDevMode()}>
      <footer
        class="shrink-0 bg-card border-t border-border font-mono text-xs flex flex-col"
        classList={{ "select-none": isDragging() }}
      >
        {/* Resize Handle - Only visible when expanded */}
        <Show when={isExpanded()}>
          <aside
            onMouseDown={handleDragStart}
            class="h-1 bg-border hover:bg-primary cursor-ns-resize transition-colors flex items-center justify-center group"
            title="Drag to resize console"
          >
            <span class="w-8 h-0.5 bg-muted-foreground/30 group-hover:bg-primary-foreground/50 rounded-full" />
          </aside>
        </Show>

        {/* Console Header Bar - Always visible when dev mode is on */}
        <header class="flex items-center justify-between px-3 h-12 bg-muted border-b border-border">
          {/* Left section - Console toggle and info */}
          <section class="flex items-center gap-3">
            <button
              onClick={toggleExpanded}
              class="flex items-center gap-1.5 text-muted-foreground hover:text-foreground transition-colors"
              title={isExpanded() ? "Collapse console" : "Expand console"}
            >
              <Icon
                name={isExpanded() ? "caret-down" : "caret-right"}
                size="xs"
              />
              <span class="text-foreground">Console</span>
              <span class="text-muted-foreground">
                ({isExpanded() ? "Open" : "Closed"})
              </span>
            </button>

            <span class="text-border">|</span>

            <span class="text-primary font-bold text-[10px] tracking-wider">
              DEV MODE
            </span>

            <span class="text-border">|</span>

            {/* View Switcher */}
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

            <span class="text-border">|</span>

            {/* Auth Status */}
            <span class="text-muted-foreground">
              Auth:{" "}
              <span
                class={props.isLoggedIn ? "text-primary" : "text-destructive"}
              >
                {props.isLoggedIn ? "Logged In" : "Anonymous"}
              </span>
            </span>
          </section>

          {/* Right section - Actions */}
          <section class="flex items-center gap-2">
            <button
              onClick={props.onToggleLogin}
              class="px-2 py-0.5 bg-primary/20 hover:bg-primary/30 border border-primary/50 rounded text-primary transition-colors text-[10px]"
              title="Toggle login state"
            >
              Toggle Auth
            </button>
            <button
              onClick={toggleOrder}
              class="px-2 py-0.5 bg-secondary hover:bg-secondary/80 border border-border rounded text-secondary-foreground hover:text-foreground transition-colors text-[10px] flex items-center gap-1"
              title={
                newestFirst() ? "Showing newest first" : "Showing oldest first"
              }
            >
              <Icon
                name={newestFirst() ? "sort-descending" : "sort-ascending"}
                size="xs"
              />
              {newestFirst() ? "Newest" : "Oldest"}
            </button>
            <button
              onClick={clearLogs}
              class="px-2 py-0.5 bg-secondary hover:bg-secondary/80 border border-border rounded text-secondary-foreground hover:text-foreground transition-colors text-[10px]"
              title="Clear console logs"
            >
              Clear
            </button>
            <button
              onClick={handleDisableDevMode}
              class="px-2 py-0.5 bg-destructive/20 hover:bg-destructive/30 border border-destructive/50 rounded text-destructive-foreground transition-colors"
              title="Disable dev mode"
            >
              <Icon name="x" size="sm" />
            </button>
          </section>
        </header>

        {/* Console Log Area - Expandable with CSS transition */}
        <section
          class="overflow-hidden bg-card"
          classList={{
            "transition-[height] duration-200 ease-out": !isDragging(),
          }}
          style={{ height: isExpanded() ? `${getCurrentHeight()}px` : "0px" }}
        >
          <section
            class="overflow-y-auto"
            style={{ height: isExpanded() ? `${getCurrentHeight()}px` : "0px" }}
          >
            <Show
              when={logs().length > 0}
              fallback={
                <article class="flex items-center justify-center h-full text-muted-foreground py-8">
                  <span>No console entries yet.</span>
                </article>
              }
            >
              <ul class="divide-y divide-border">
                <For each={orderedLogs()}>
                  {(entry) => (
                    <li class="flex items-start gap-3 px-3 py-2 hover:bg-muted/50 transition-colors">
                      {/* Timestamp */}
                      <time class="text-primary shrink-0 tabular-nums">
                        {formatTime(entry.timestamp)}
                      </time>

                      {/* Level Badge */}
                      <span class={getLevelBadgeStyles(entry.level)}>
                        {getLevelLabel(entry.level)}
                      </span>

                      {/* Message */}
                      <p class="text-foreground flex-1">
                        {entry.message}
                        <Show when={entry.details}>
                          <span class="text-muted-foreground ml-2">
                            {entry.details}
                          </span>
                        </Show>
                      </p>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </section>
        </section>
      </footer>
    </Show>
  );
};
