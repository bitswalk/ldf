import type { Component, JSX } from "solid-js";
import {
  createSignal,
  createContext,
  useContext,
  Show,
  For,
} from "solid-js";
import { Icon } from "../Icon";

// Context for sharing summary state
interface SummaryContextValue {
  activeSection: () => string;
  setActiveSection: (section: string) => void;
  expandedCategories: () => Set<string>;
  toggleCategory: (category: string) => void;
}

const SummaryContext = createContext<SummaryContextValue>();

const useSummaryContext = () => {
  const context = useContext(SummaryContext);
  if (!context) {
    throw new Error("Summary components must be used within a Summary provider");
  }
  return context;
};

// Types
interface SummaryProps {
  defaultSection?: string;
  defaultExpanded?: string[];
  children?: JSX.Element;
  class?: string;
}

interface SummaryNavProps {
  children?: JSX.Element;
  class?: string;
}

interface SummaryCategoryProps {
  id: string;
  label: string;
  icon?: string;
  children?: JSX.Element;
  class?: string;
}

interface SummaryNavItemProps {
  id: string;
  label: string;
  icon?: string;
  class?: string;
}

interface SummaryContentProps {
  children?: JSX.Element;
  class?: string;
}

interface SummarySectionProps {
  id: string;
  title: string;
  description?: string;
  children?: JSX.Element;
  class?: string;
}

interface SummaryItemProps {
  title: string;
  description?: string;
  icon?: string;
  children?: JSX.Element;
  class?: string;
}

interface SummaryToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  class?: string;
}

interface SummarySelectProps {
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
  disabled?: boolean;
  class?: string;
}

interface SummaryButtonProps {
  onClick: () => void;
  disabled?: boolean;
  children?: JSX.Element;
  class?: string;
}

// Summary Root Component
export const Summary: Component<SummaryProps> = (props) => {
  const [activeSection, setActiveSection] = createSignal(props.defaultSection ?? "");
  const [expandedCategories, setExpandedCategories] = createSignal<Set<string>>(
    new Set(props.defaultExpanded ?? [])
  );

  const toggleCategory = (category: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  };

  const contextValue: SummaryContextValue = {
    activeSection,
    setActiveSection,
    expandedCategories,
    toggleCategory,
  };

  return (
    <SummaryContext.Provider value={contextValue}>
      <section class={`flex h-full ${props.class ?? ""}`}>
        {props.children}
      </section>
    </SummaryContext.Provider>
  );
};

// Summary Navigation Sidebar
export const SummaryNav: Component<SummaryNavProps> = (props) => {
  return (
    <nav class={`w-56 shrink-0 border-r border-border overflow-y-auto ${props.class ?? ""}`}>
      <ul class="flex flex-col py-2">
        {props.children}
      </ul>
    </nav>
  );
};

// Summary Category (expandable group)
export const SummaryCategory: Component<SummaryCategoryProps> = (props) => {
  const context = useSummaryContext();

  const isExpanded = () => context.expandedCategories().has(props.id);

  const handleToggle = () => {
    context.toggleCategory(props.id);
  };

  return (
    <li class={`flex flex-col ${props.class ?? ""}`}>
      <button
        onClick={handleToggle}
        class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/50 transition-colors w-full text-left"
      >
        <Icon
          name={isExpanded() ? "caret-down" : "caret-right"}
          size="xs"
          class="text-muted-foreground"
        />
        <Show when={props.icon}>
          <Icon name={props.icon!} size="sm" class="text-muted-foreground" />
        </Show>
        <span>{props.label}</span>
      </button>
      <Show when={isExpanded()}>
        <ul class="flex flex-col">
          {props.children}
        </ul>
      </Show>
    </li>
  );
};

// Summary Navigation Item
export const SummaryNavItem: Component<SummaryNavItemProps> = (props) => {
  const context = useSummaryContext();

  const isActive = () => context.activeSection() === props.id;

  const handleClick = () => {
    context.setActiveSection(props.id);
  };

  return (
    <li>
      <button
        onClick={handleClick}
        class={`flex items-center gap-2 pl-10 pr-4 py-1.5 text-sm w-full text-left transition-colors ${
          isActive()
            ? "bg-primary/10 text-primary border-l-2 border-primary"
            : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
        } ${props.class ?? ""}`}
      >
        <Show when={props.icon}>
          <Icon name={props.icon!} size="sm" />
        </Show>
        <span>{props.label}</span>
      </button>
    </li>
  );
};

// Summary Content Area
export const SummaryContent: Component<SummaryContentProps> = (props) => {
  return (
    <article class={`flex-1 overflow-y-auto ${props.class ?? ""}`}>
      {props.children}
    </article>
  );
};

// Summary Section (shown based on active section)
export const SummarySection: Component<SummarySectionProps> = (props) => {
  const context = useSummaryContext();

  const isActive = () => context.activeSection() === props.id;

  return (
    <Show when={isActive()}>
      <section class={`p-6 ${props.class ?? ""}`}>
        <header class="mb-6">
          <h2 class="text-xl font-semibold">{props.title}</h2>
          <Show when={props.description}>
            <p class="text-sm text-muted-foreground mt-1">{props.description}</p>
          </Show>
        </header>
        <section class="flex flex-col divide-y divide-border">
          {props.children}
        </section>
      </section>
    </Show>
  );
};

// Summary Item (individual setting row)
export const SummaryItem: Component<SummaryItemProps> = (props) => {
  return (
    <article class={`flex items-center justify-between py-4 gap-4 ${props.class ?? ""}`}>
      <section class="flex items-start gap-3 flex-1 min-w-0">
        <Show when={props.icon}>
          <Icon name={props.icon!} size="sm" class="text-muted-foreground mt-0.5 shrink-0" />
        </Show>
        <section class="flex flex-col min-w-0">
          <span class="font-medium">{props.title}</span>
          <Show when={props.description}>
            <span class="text-sm text-muted-foreground">{props.description}</span>
          </Show>
        </section>
      </section>
      <section class="shrink-0">
        {props.children}
      </section>
    </article>
  );
};

// Summary Toggle (on/off switch)
export const SummaryToggle: Component<SummaryToggleProps> = (props) => {
  const handleClick = () => {
    if (!props.disabled) {
      props.onChange(!props.checked);
    }
  };

  return (
    <button
      type="button"
      role="switch"
      aria-checked={props.checked}
      disabled={props.disabled}
      onClick={handleClick}
      class={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${
        props.checked ? "bg-primary" : "bg-muted"
      } ${props.class ?? ""}`}
    >
      <span
        class={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-background shadow-lg ring-0 transition-transform ${
          props.checked ? "translate-x-5" : "translate-x-0"
        }`}
      />
    </button>
  );
};

// Summary Select (dropdown)
export const SummarySelect: Component<SummarySelectProps> = (props) => {
  const handleChange = (e: Event) => {
    const target = e.target as HTMLSelectElement;
    props.onChange(target.value);
  };

  return (
    <select
      value={props.value}
      onChange={handleChange}
      disabled={props.disabled}
      class={`px-3 py-1.5 text-sm border border-border rounded bg-background text-foreground cursor-pointer hover:bg-muted/50 focus:outline-none focus:ring-2 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50 ${props.class ?? ""}`}
    >
      <For each={props.options}>
        {(option) => (
          <option value={option.value}>{option.label}</option>
        )}
      </For>
    </select>
  );
};

// Summary Button
export const SummaryButton: Component<SummaryButtonProps> = (props) => {
  return (
    <button
      type="button"
      onClick={props.onClick}
      disabled={props.disabled}
      class={`px-3 py-1.5 text-sm border border-border rounded bg-background text-foreground hover:bg-muted/50 transition-colors focus:outline-none focus:ring-2 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50 ${props.class ?? ""}`}
    >
      {props.children}
    </button>
  );
};
