import type { Component, JSX } from "solid-js";
import { Icon } from "../Icon";
import {
  createSignal,
  splitProps,
  Show,
  children,
  createContext,
  useContext,
  onCleanup,
  onMount,
} from "solid-js";

// Context for managing dropdown state
interface DropdownContextValue {
  isOpen: () => boolean;
  setIsOpen: (open: boolean) => void;
  toggle: () => void;
}

const DropdownContext = createContext<DropdownContextValue>();

// DropdownMenu Root Component
interface DropdownMenuProps {
  children: JSX.Element;
  modal?: boolean;
}

export const DropdownMenu: Component<DropdownMenuProps> = (props) => {
  const [isOpen, setIsOpen] = createSignal(false);

  const toggle = () => setIsOpen(!isOpen());

  const contextValue: DropdownContextValue = {
    isOpen,
    setIsOpen,
    toggle,
  };

  return (
    <DropdownContext.Provider value={contextValue}>
      <section class="relative inline-block">{props.children}</section>
    </DropdownContext.Provider>
  );
};

// DropdownMenuTrigger Component
interface DropdownMenuTriggerProps {
  children: JSX.Element;
  class?: string;
  asChild?: boolean;
}

export const DropdownMenuTrigger: Component<DropdownMenuTriggerProps> = (
  props,
) => {
  const context = useContext(DropdownContext);
  if (!context)
    throw new Error("DropdownMenuTrigger must be used within DropdownMenu");

  const handleClick = (e: MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    context.toggle();
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      class={props.class}
      aria-haspopup="true"
      aria-expanded={context.isOpen()}
    >
      {props.children}
    </button>
  );
};

// DropdownMenuContent Component
interface DropdownMenuContentProps {
  children: JSX.Element;
  class?: string;
  align?: "start" | "center" | "end";
  side?: "top" | "bottom";
}

export const DropdownMenuContent: Component<DropdownMenuContentProps> = (
  props,
) => {
  const context = useContext(DropdownContext);
  if (!context)
    throw new Error("DropdownMenuContent must be used within DropdownMenu");

  const alignClass = () => {
    switch (props.align) {
      case "start":
        return "left-0";
      case "end":
        return "right-0";
      default:
        return "left-0";
    }
  };

  const sideClass = () => {
    switch (props.side) {
      case "top":
        return "bottom-full mb-2";
      case "bottom":
      default:
        return "mt-2";
    }
  };

  return (
    <Show when={context.isOpen()}>
      {() => {
        let contentRef: HTMLElement | undefined;

        const handleClickOutside = (e: MouseEvent) => {
          if (contentRef && !contentRef.contains(e.target as Node)) {
            context.setIsOpen(false);
          }
        };

        onMount(() => {
          document.addEventListener("click", handleClickOutside);
        });

        onCleanup(() => {
          document.removeEventListener("click", handleClickOutside);
        });

        return (
          <nav
            ref={contentRef}
            class={`absolute ${alignClass()} ${sideClass()} z-50 min-w-[8rem] overflow-hidden rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md ${
              props.class || ""
            }`}
            role="menu"
            onClick={(e) => e.stopPropagation()}
          >
            {props.children}
          </nav>
        );
      }}
    </Show>
  );
};

// DropdownMenuItem Component
interface DropdownMenuItemProps extends JSX.HTMLAttributes<HTMLButtonElement> {
  children: JSX.Element;
  class?: string;
  disabled?: boolean;
  onSelect?: () => void;
}

export const DropdownMenuItem: Component<DropdownMenuItemProps> = (props) => {
  const context = useContext(DropdownContext);
  const [local, others] = splitProps(props, [
    "children",
    "class",
    "disabled",
    "onSelect",
  ]);

  const handleClick = () => {
    if (!local.disabled && local.onSelect) {
      local.onSelect();
      context?.setIsOpen(false);
    }
  };

  return (
    <button
      type="button"
      role="menuitem"
      class={`relative flex w-full cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 ${
        local.class || ""
      }`}
      disabled={local.disabled}
      onClick={handleClick}
      {...others}
    >
      {local.children}
    </button>
  );
};

// DropdownMenuLabel Component
interface DropdownMenuLabelProps {
  children: JSX.Element;
  class?: string;
}

export const DropdownMenuLabel: Component<DropdownMenuLabelProps> = (props) => {
  return (
    <div class={`px-2 py-1.5 text-sm font-semibold ${props.class || ""}`}>
      {props.children}
    </div>
  );
};

// DropdownMenuSeparator Component
interface DropdownMenuSeparatorProps {
  class?: string;
}

export const DropdownMenuSeparator: Component<DropdownMenuSeparatorProps> = (
  props,
) => {
  return (
    <hr
      class={`-mx-1 my-1 h-px bg-muted ${props.class || ""}`}
      role="separator"
    />
  );
};

// DropdownMenuShortcut Component
interface DropdownMenuShortcutProps {
  children: JSX.Element;
  class?: string;
}

export const DropdownMenuShortcut: Component<DropdownMenuShortcutProps> = (
  props,
) => {
  return (
    <span
      class={`ml-auto text-xs tracking-widest opacity-60 ${props.class || ""}`}
    >
      {props.children}
    </span>
  );
};

// DropdownMenuGroup Component
interface DropdownMenuGroupProps {
  children: JSX.Element;
}

export const DropdownMenuGroup: Component<DropdownMenuGroupProps> = (props) => {
  return <>{props.children}</>;
};

// DropdownMenuCheckboxItem Component
interface DropdownMenuCheckboxItemProps {
  children: JSX.Element;
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  class?: string;
  disabled?: boolean;
}

export const DropdownMenuCheckboxItem: Component<
  DropdownMenuCheckboxItemProps
> = (props) => {
  const context = useContext(DropdownContext);

  const handleClick = () => {
    if (!props.disabled && props.onCheckedChange) {
      props.onCheckedChange(!props.checked);
    }
  };

  return (
    <button
      type="button"
      role="menuitemcheckbox"
      aria-checked={props.checked}
      class={`relative flex w-full cursor-pointer select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 ${
        props.class || ""
      }`}
      disabled={props.disabled}
      onClick={handleClick}
    >
      <span class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
        <Show when={props.checked}>
          <Icon name="check" size="xs" />
        </Show>
      </span>
      {props.children}
    </button>
  );
};

// DropdownMenuRadioGroup Component
interface DropdownMenuRadioGroupProps {
  children: JSX.Element;
  value?: string;
  onValueChange?: (value: string) => void;
}

const RadioGroupContext = createContext<{
  value: () => string | undefined;
  onValueChange?: (value: string) => void;
}>();

export const DropdownMenuRadioGroup: Component<DropdownMenuRadioGroupProps> = (
  props,
) => {
  const value = () => props.value;

  return (
    <RadioGroupContext.Provider
      value={{ value, onValueChange: props.onValueChange }}
    >
      <div role="group">{props.children}</div>
    </RadioGroupContext.Provider>
  );
};

// DropdownMenuRadioItem Component
interface DropdownMenuRadioItemProps {
  children: JSX.Element;
  value: string;
  class?: string;
  disabled?: boolean;
}

export const DropdownMenuRadioItem: Component<DropdownMenuRadioItemProps> = (
  props,
) => {
  const radioContext = useContext(RadioGroupContext);
  const isChecked = () => radioContext?.value() === props.value;

  const handleClick = () => {
    if (!props.disabled && radioContext?.onValueChange) {
      radioContext.onValueChange(props.value);
    }
  };

  return (
    <button
      type="button"
      role="menuitemradio"
      aria-checked={isChecked()}
      class={`relative flex w-full cursor-pointer select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 ${
        props.class || ""
      }`}
      disabled={props.disabled}
      onClick={handleClick}
    >
      <span class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
        <Show when={isChecked()}>
          <Icon name="circle" size="xs" />
        </Show>
      </span>
      {props.children}
    </button>
  );
};

// DropdownMenuSub Component (simplified for now)
interface DropdownMenuSubProps {
  children: JSX.Element;
}

export const DropdownMenuSub: Component<DropdownMenuSubProps> = (props) => {
  return <>{props.children}</>;
};

// DropdownMenuSubTrigger Component
interface DropdownMenuSubTriggerProps {
  children: JSX.Element;
  class?: string;
}

export const DropdownMenuSubTrigger: Component<DropdownMenuSubTriggerProps> = (
  props,
) => {
  return (
    <button
      type="button"
      class={`relative flex w-full cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground ${
        props.class || ""
      }`}
    >
      {props.children}
      <Icon name="caret-right" size="xs" class="ml-auto" />
    </button>
  );
};

// DropdownMenuSubContent Component
interface DropdownMenuSubContentProps {
  children: JSX.Element;
  class?: string;
}

export const DropdownMenuSubContent: Component<DropdownMenuSubContentProps> = (
  props,
) => {
  return (
    <nav
      class={`min-w-[8rem] overflow-hidden rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md ${
        props.class || ""
      }`}
      role="menu"
    >
      {props.children}
    </nav>
  );
};

// DropdownMenuPortal Component (simplified - just renders children)
interface DropdownMenuPortalProps {
  children: JSX.Element;
}

export const DropdownMenuPortal: Component<DropdownMenuPortalProps> = (
  props,
) => {
  return <>{props.children}</>;
};
