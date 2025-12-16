import type { Component, JSX } from "solid-js";
import {
  createSignal,
  createContext,
  useContext,
  Show,
  children as resolveChildren,
} from "solid-js";

// Context for sharing tab state
interface TabsContextValue {
  value: () => string;
  setValue: (value: string) => void;
  orientation: () => TabsOrientation;
}

const TabsContext = createContext<TabsContextValue>();

export const useTabsContext = () => {
  const context = useContext(TabsContext);
  if (!context) {
    throw new Error("Tabs components must be used within a Tabs provider");
  }
  return context;
};

// Types
type TabsOrientation = "horizontal" | "vertical";

interface TabsProps {
  defaultValue?: string;
  value?: string;
  onValueChange?: (value: string) => void;
  orientation?: TabsOrientation;
  children?: JSX.Element;
  class?: string;
}

interface TabsListProps {
  children?: JSX.Element;
  class?: string;
}

interface TabsTriggerProps {
  value: string;
  disabled?: boolean;
  children?: JSX.Element;
  class?: string;
}

interface TabsContentProps {
  value: string;
  children?: JSX.Element;
  class?: string;
}

// Tabs Root Component
export const Tabs: Component<TabsProps> = (props) => {
  const [internalValue, setInternalValue] = createSignal(
    props.value ?? props.defaultValue ?? "",
  );

  const value = () => props.value ?? internalValue();

  const setValue = (newValue: string) => {
    if (props.value === undefined) {
      setInternalValue(newValue);
    }
    props.onValueChange?.(newValue);
  };

  const orientation = () => props.orientation ?? "horizontal";

  const contextValue: TabsContextValue = {
    value,
    setValue,
    orientation,
  };

  return (
    <TabsContext.Provider value={contextValue}>
      <section
        class={`flex ${orientation() === "vertical" ? "flex-row" : "flex-col"} ${props.class ?? ""}`}
        data-orientation={orientation()}
      >
        {props.children}
      </section>
    </TabsContext.Provider>
  );
};

// TabsList Component
export const TabsList: Component<TabsListProps> = (props) => {
  const context = useTabsContext();

  return (
    <nav
      role="tablist"
      class={`inline-flex items-center justify-center rounded bg-muted p-1 text-muted-foreground ${
        context.orientation() === "vertical" ? "flex-col h-auto" : "flex-row"
      } ${props.class ?? ""}`}
      data-orientation={context.orientation()}
    >
      {props.children}
    </nav>
  );
};

// TabsTrigger Component
export const TabsTrigger: Component<TabsTriggerProps> = (props) => {
  const context = useTabsContext();

  const isSelected = () => context.value() === props.value;

  const handleClick = () => {
    if (!props.disabled) {
      context.setValue(props.value);
    }
  };

  return (
    <button
      role="tab"
      type="button"
      aria-selected={isSelected()}
      disabled={props.disabled}
      onClick={handleClick}
      class={`inline-flex items-center justify-center whitespace-nowrap rounded px-3 py-1.5 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 ${
        isSelected()
          ? "bg-background text-foreground shadow-sm"
          : "hover:bg-background/50 hover:text-foreground"
      } ${props.class ?? ""}`}
      data-state={isSelected() ? "active" : "inactive"}
    >
      {props.children}
    </button>
  );
};

// TabsContent Component
export const TabsContent: Component<TabsContentProps> = (props) => {
  const context = useTabsContext();

  const isSelected = () => context.value() === props.value;
  const isVertical = () => context.orientation() === "vertical";

  return (
    <Show when={isSelected()}>
      <section
        role="tabpanel"
        class={`ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${
          isVertical() ? "flex-1" : "mt-2"
        } ${props.class ?? ""}`}
        data-state={isSelected() ? "active" : "inactive"}
      >
        {props.children}
      </section>
    </Show>
  );
};
