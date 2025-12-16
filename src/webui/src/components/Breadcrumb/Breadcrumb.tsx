import type { Component, JSX } from "solid-js";
import { splitProps } from "solid-js";
import { Icon } from "../Icon";

// Breadcrumb Root Component
interface BreadcrumbProps extends JSX.HTMLAttributes<HTMLElement> {
  children: JSX.Element;
  class?: string;
}

export const Breadcrumb: Component<BreadcrumbProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <nav aria-label="breadcrumb" class={className} {...others}>
      {local.children}
    </nav>
  );
};

// BreadcrumbList Component
interface BreadcrumbListProps extends JSX.HTMLAttributes<HTMLOListElement> {
  children: JSX.Element;
  class?: string;
}

export const BreadcrumbList: Component<BreadcrumbListProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <ol
      class={`flex flex-wrap items-center gap-1.5 break-words text-sm text-muted-foreground sm:gap-2.5 ${className}`}
      {...others}
    >
      {local.children}
    </ol>
  );
};

// BreadcrumbItem Component
interface BreadcrumbItemProps extends JSX.HTMLAttributes<HTMLLIElement> {
  children: JSX.Element;
  class?: string;
}

export const BreadcrumbItem: Component<BreadcrumbItemProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <li class={`inline-flex items-center gap-1.5 ${className}`} {...others}>
      {local.children}
    </li>
  );
};

// BreadcrumbLink Component
interface BreadcrumbLinkProps
  extends JSX.AnchorHTMLAttributes<HTMLAnchorElement> {
  children: JSX.Element;
  class?: string;
  asChild?: boolean;
}

export const BreadcrumbLink: Component<BreadcrumbLinkProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children", "asChild"]);
  const className = local.class ? `${local.class}` : "";

  // If asChild is true, render children directly (for custom link components)
  if (local.asChild) {
    return <>{local.children}</>;
  }

  return (
    <a
      class={`transition-colors hover:text-foreground ${className}`}
      {...others}
    >
      {local.children}
    </a>
  );
};

// BreadcrumbPage Component (current page, non-interactive)
interface BreadcrumbPageProps extends JSX.HTMLAttributes<HTMLSpanElement> {
  children: JSX.Element;
  class?: string;
}

export const BreadcrumbPage: Component<BreadcrumbPageProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <span
      role="link"
      aria-disabled="true"
      aria-current="page"
      class={`font-normal text-foreground ${className}`}
      {...others}
    >
      {local.children}
    </span>
  );
};

// BreadcrumbSeparator Component
interface BreadcrumbSeparatorProps extends JSX.HTMLAttributes<HTMLLIElement> {
  children?: JSX.Element;
  class?: string;
}

export const BreadcrumbSeparator: Component<BreadcrumbSeparatorProps> = (
  props,
) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <li
      role="presentation"
      aria-hidden="true"
      class={`[&>svg]:size-3.5 ${className}`}
      {...others}
    >
      {local.children || <Icon name="caret-right" size="xs" />}
    </li>
  );
};

// BreadcrumbEllipsis Component
interface BreadcrumbEllipsisProps extends JSX.HTMLAttributes<HTMLSpanElement> {
  class?: string;
}

export const BreadcrumbEllipsis: Component<BreadcrumbEllipsisProps> = (
  props,
) => {
  const [local, others] = splitProps(props, ["class"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <span
      role="presentation"
      aria-hidden="true"
      class={`flex h-9 w-9 items-center justify-center ${className}`}
      {...others}
    >
      <Icon name="dots-three" size="lg" />
      <span class="sr-only">More</span>
    </span>
  );
};
