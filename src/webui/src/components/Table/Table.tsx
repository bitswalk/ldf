import type { Component, JSX } from "solid-js";
import { splitProps } from "solid-js";

// Table Root Component
interface TableProps extends JSX.TableHTMLAttributes<HTMLTableElement> {
  children: JSX.Element;
  class?: string;
}

export const Table: Component<TableProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <table class={`w-full caption-bottom text-sm ${className}`} {...others}>
      {local.children}
    </table>
  );
};

// Table Caption Component
interface TableCaptionProps
  extends JSX.HTMLAttributes<HTMLTableCaptionElement> {
  children: JSX.Element;
  class?: string;
}

export const TableCaption: Component<TableCaptionProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <caption
      class={`mt-4 text-sm text-muted-foreground ${className}`}
      {...others}
    >
      {local.children}
    </caption>
  );
};

// Table Header Component
interface TableHeaderProps extends JSX.HTMLAttributes<HTMLTableSectionElement> {
  children: JSX.Element;
  class?: string;
}

export const TableHeader: Component<TableHeaderProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <thead class={`[&_tr]:border-b ${className}`} {...others}>
      {local.children}
    </thead>
  );
};

// Table Body Component
interface TableBodyProps extends JSX.HTMLAttributes<HTMLTableSectionElement> {
  children: JSX.Element;
  class?: string;
}

export const TableBody: Component<TableBodyProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <tbody class={`[&_tr:last-child]:border-0 ${className}`} {...others}>
      {local.children}
    </tbody>
  );
};

// Table Footer Component
interface TableFooterProps extends JSX.HTMLAttributes<HTMLTableSectionElement> {
  children: JSX.Element;
  class?: string;
}

export const TableFooter: Component<TableFooterProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <tfoot
      class={`border-t bg-muted/50 font-medium [&>tr]:last:border-b-0 ${className}`}
      {...others}
    >
      {local.children}
    </tfoot>
  );
};

// Table Head Component
interface TableHeadProps extends JSX.ThHTMLAttributes<HTMLTableCellElement> {
  children?: JSX.Element;
  class?: string;
}

export const TableHead: Component<TableHeadProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <th
      class={`h-12 px-4 text-left align-middle font-medium text-muted-foreground [&:has([role=checkbox])]:pr-0 ${className}`}
      {...others}
    >
      {local.children}
    </th>
  );
};

// Table Row Component
interface TableRowProps extends JSX.HTMLAttributes<HTMLTableRowElement> {
  children: JSX.Element;
  class?: string;
}

export const TableRow: Component<TableRowProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <tr
      class={`border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted ${className}`}
      {...others}
    >
      {local.children}
    </tr>
  );
};

// Table Cell Component
interface TableCellProps extends JSX.TdHTMLAttributes<HTMLTableCellElement> {
  children?: JSX.Element;
  class?: string;
}

export const TableCell: Component<TableCellProps> = (props) => {
  const [local, others] = splitProps(props, ["class", "children"]);
  const className = local.class ? `${local.class}` : "";

  return (
    <td class={`p-4 align-middle ${className}`} {...others}>
      {local.children}
    </td>
  );
};
