import type { Component, JSX } from "solid-js";
import { splitProps, mergeProps } from "solid-js";
import { Icon } from "../Icon";

// Pagination Root Component
interface PaginationProps extends JSX.HTMLAttributes<HTMLElement> {
  children: JSX.Element;
  class?: string;
}

export const Pagination: Component<PaginationProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <nav
      role="navigation"
      aria-label="pagination"
      class={`mx-auto flex w-full justify-center ${className}`}
      {...others}
    >
      {local.children}
    </nav>
  );
};

// Pagination Content Component
interface PaginationContentProps extends JSX.HTMLAttributes<HTMLUListElement> {
  children: JSX.Element;
  class?: string;
}

export const PaginationContent: Component<PaginationContentProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <ul class={`flex flex-row items-center gap-1 ${className}`} {...others}>
      {local.children}
    </ul>
  );
};

// Pagination Item Component
interface PaginationItemProps extends JSX.HTMLAttributes<HTMLLIElement> {
  children: JSX.Element;
  class?: string;
}

export const PaginationItem: Component<PaginationItemProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <li class={className} {...others}>
      {local.children}
    </li>
  );
};

// Pagination Link Component
interface PaginationLinkProps
  extends JSX.AnchorHTMLAttributes<HTMLAnchorElement> {
  children?: JSX.Element;
  isActive?: boolean;
  size?: "default" | "icon";
  class?: string;
}

export const PaginationLink: Component<PaginationLinkProps> = (props) => {
  const merged = mergeProps({ size: "icon" as const, isActive: false }, props);
  const [local, others] = splitProps(merged, [
    "class",
    "children",
    "isActive",
    "size",
  ]);

  const baseClass =
    "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50";
  const sizeClass = local.size === "default" ? "h-10 px-4 py-2" : "h-10 w-10";
  const stateClass = local.isActive
    ? "border border-border bg-background hover:bg-accent hover:text-accent-foreground"
    : "hover:bg-accent hover:text-accent-foreground";
  const customClass = local.class || "";

  return (
    <a
      aria-current={local.isActive ? "page" : undefined}
      class={`${baseClass} ${sizeClass} ${stateClass} ${customClass}`}
      {...others}
    >
      {local.children}
    </a>
  );
};

// Pagination Previous Component
interface PaginationPreviousProps
  extends JSX.AnchorHTMLAttributes<HTMLAnchorElement> {
  class?: string;
}

export const PaginationPrevious: Component<PaginationPreviousProps> = (
  props,
) => {
  const [local, others] = splitProps(props, ["class"]);
  const customClass = local.class || "";

  return (
    <a
      aria-label="Go to previous page"
      class={`inline-flex items-center justify-center gap-1 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 h-10 px-4 py-2 hover:bg-accent hover:text-accent-foreground ${customClass}`}
      {...others}
    >
      <Icon name="caret-left" size="sm" />
      <span>Previous</span>
    </a>
  );
};

// Pagination Next Component
interface PaginationNextProps
  extends JSX.AnchorHTMLAttributes<HTMLAnchorElement> {
  class?: string;
}

export const PaginationNext: Component<PaginationNextProps> = (props) => {
  const [local, others] = splitProps(props, ["class"]);
  const customClass = local.class || "";

  return (
    <a
      aria-label="Go to next page"
      class={`inline-flex items-center justify-center gap-1 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 h-10 px-4 py-2 hover:bg-accent hover:text-accent-foreground ${customClass}`}
      {...others}
    >
      <span>Next</span>
      <Icon name="caret-right" size="sm" />
    </a>
  );
};

// Pagination Ellipsis Component
interface PaginationEllipsisProps extends JSX.HTMLAttributes<HTMLSpanElement> {
  class?: string;
}

export const PaginationEllipsis: Component<PaginationEllipsisProps> = (
  props,
) => {
  const [local, others] = splitProps(props, ["class"]);
  const customClass = local.class || "";

  return (
    <span
      aria-hidden="true"
      class={`flex h-9 w-9 items-center justify-center ${customClass}`}
      {...others}
    >
      <Icon name="dots-three" size="lg" />
      <span class="sr-only">More pages</span>
    </span>
  );
};
