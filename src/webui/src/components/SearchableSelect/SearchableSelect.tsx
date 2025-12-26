import type { Component, JSX } from "solid-js";
import {
  createSignal,
  createEffect,
  onCleanup,
  For,
  Show,
} from "solid-js";
import { Icon } from "../Icon";

export interface SearchableSelectOption {
  value: string;
  label: string;
  sublabel?: string;
}

interface SearchableSelectProps {
  value: string;
  options: SearchableSelectOption[];
  onChange: (value: string) => void;
  placeholder?: string;
  searchPlaceholder?: string;
  loading?: boolean;
  onSearch?: (query: string) => void;
  maxDisplayed?: number;
  class?: string;
}

export const SearchableSelect: Component<SearchableSelectProps> = (props) => {
  const [isOpen, setIsOpen] = createSignal(false);
  const [searchQuery, setSearchQuery] = createSignal("");
  const [highlightedIndex, setHighlightedIndex] = createSignal(0);
  let containerRef: HTMLDivElement | undefined;
  let inputRef: HTMLInputElement | undefined;

  const maxDisplayed = () => props.maxDisplayed ?? 10;

  const filteredOptions = () => {
    const query = searchQuery().toLowerCase().trim();
    if (!query) {
      return props.options.slice(0, maxDisplayed());
    }
    return props.options
      .filter(
        (opt) =>
          opt.value.toLowerCase().includes(query) ||
          opt.label.toLowerCase().includes(query)
      )
      .slice(0, maxDisplayed());
  };

  const selectedOption = () =>
    props.options.find((opt) => opt.value === props.value);

  const handleSelect = (option: SearchableSelectOption) => {
    props.onChange(option.value);
    setIsOpen(false);
    setSearchQuery("");
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (!isOpen()) {
      if (e.key === "Enter" || e.key === " " || e.key === "ArrowDown") {
        e.preventDefault();
        setIsOpen(true);
      }
      return;
    }

    const options = filteredOptions();

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setHighlightedIndex((i) => Math.min(i + 1, options.length - 1));
        break;
      case "ArrowUp":
        e.preventDefault();
        setHighlightedIndex((i) => Math.max(i - 1, 0));
        break;
      case "Enter":
        e.preventDefault();
        if (options[highlightedIndex()]) {
          handleSelect(options[highlightedIndex()]);
        }
        break;
      case "Escape":
        e.preventDefault();
        setIsOpen(false);
        setSearchQuery("");
        break;
    }
  };

  const handleSearchInput = (e: Event) => {
    const value = (e.target as HTMLInputElement).value;
    setSearchQuery(value);
    setHighlightedIndex(0);
    props.onSearch?.(value);
  };

  // Close on click outside
  createEffect(() => {
    if (!isOpen()) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef && !containerRef.contains(e.target as Node)) {
        setIsOpen(false);
        setSearchQuery("");
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    onCleanup(() => document.removeEventListener("mousedown", handleClickOutside));
  });

  // Focus search input when opened
  createEffect(() => {
    if (isOpen() && inputRef) {
      inputRef.focus();
    }
  });

  // Reset highlighted index when options change
  createEffect(() => {
    filteredOptions();
    setHighlightedIndex(0);
  });

  return (
    <div
      ref={containerRef}
      class={`relative ${props.class || ""}`}
      onKeyDown={handleKeyDown}
    >
      {/* Trigger button */}
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen())}
        class="flex items-center justify-between gap-2 px-2 py-1 text-sm bg-background border border-border rounded-md hover:border-primary focus:outline-none focus:border-primary min-w-[140px] text-right"
      >
        <span class="font-mono truncate">
          {selectedOption()?.label || props.value || props.placeholder || "Select..."}
        </span>
        <Icon
          name={isOpen() ? "caret-up" : "caret-down"}
          size="sm"
          class="text-muted-foreground shrink-0"
        />
      </button>

      {/* Dropdown */}
      <Show when={isOpen()}>
        <div class="absolute right-0 top-full mt-1 w-64 bg-popover border border-border rounded-md shadow-lg z-50 overflow-hidden">
          {/* Search input */}
          <div class="p-2 border-b border-border">
            <div class="relative">
              <Icon
                name="magnifying-glass"
                size="sm"
                class="absolute left-2 top-1/2 -translate-y-1/2 text-muted-foreground"
              />
              <input
                ref={inputRef}
                type="text"
                class="w-full pl-8 pr-3 py-1.5 text-sm bg-background border border-border rounded-md focus:outline-none focus:border-primary"
                placeholder={props.searchPlaceholder || "Search..."}
                value={searchQuery()}
                onInput={handleSearchInput}
              />
            </div>
          </div>

          {/* Options list */}
          <div class="max-h-60 overflow-y-auto">
            <Show
              when={!props.loading}
              fallback={
                <div class="flex items-center justify-center py-4">
                  <Icon
                    name="spinner-gap"
                    size="md"
                    class="animate-spin text-muted-foreground"
                  />
                </div>
              }
            >
              <Show
                when={filteredOptions().length > 0}
                fallback={
                  <div class="px-3 py-4 text-sm text-center text-muted-foreground">
                    No results found
                  </div>
                }
              >
                <For each={filteredOptions()}>
                  {(option, index) => (
                    <button
                      type="button"
                      class={`w-full px-3 py-2 text-left text-sm hover:bg-muted transition-colors flex items-center justify-between ${
                        index() === highlightedIndex() ? "bg-muted" : ""
                      } ${option.value === props.value ? "text-primary" : ""}`}
                      onClick={() => handleSelect(option)}
                      onMouseEnter={() => setHighlightedIndex(index())}
                    >
                      <span class="font-mono">{option.label}</span>
                      <Show when={option.sublabel}>
                        <span class="text-xs text-muted-foreground">
                          {option.sublabel}
                        </span>
                      </Show>
                      <Show when={option.value === props.value}>
                        <Icon name="check" size="sm" class="text-primary" />
                      </Show>
                    </button>
                  )}
                </For>
              </Show>
            </Show>
          </div>

          {/* Footer hint */}
          <Show when={filteredOptions().length > 0 && props.options.length > maxDisplayed()}>
            <div class="px-3 py-2 border-t border-border text-xs text-muted-foreground text-center">
              Showing {filteredOptions().length} of {props.options.length} results
            </div>
          </Show>
        </div>
      </Show>
    </div>
  );
};
