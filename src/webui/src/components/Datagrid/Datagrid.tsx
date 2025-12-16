import type { Component, JSX } from "solid-js";
import { createSignal, For, Show } from "solid-js";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../Table/Table";
import { Icon } from "../Icon";
import { compareValues } from "../../lib/utils";

interface Column<T> {
  key: keyof T;
  label: string;
  sortable?: boolean;
  filterable?: boolean;
  render?: (value: T[keyof T], row: T) => JSX.Element;
  component?: Component<{ value: T[keyof T]; row: T }>;
  class?: string;
}

interface DatagridProps<T> {
  columns: Column<T>[];
  data: T[];
  rowKey: keyof T;
  selectable?: boolean;
  onSelectionChange?: (selectedRows: T[]) => void;
  class?: string;
}

export function Datagrid<T extends Record<string, any>>(
  props: DatagridProps<T>,
): JSX.Element {
  const [selectedRows, setSelectedRows] = createSignal<Set<string | number>>(
    new Set(),
  );
  const [filters, setFilters] = createSignal<Partial<Record<keyof T, string>>>(
    {},
  );
  const [sortColumn, setSortColumn] = createSignal<keyof T | null>(null);
  const [sortDirection, setSortDirection] = createSignal<"asc" | "desc">("asc");

  const toggleRowSelection = (rowId: string | number) => {
    const current = new Set(selectedRows());
    if (current.has(rowId)) {
      current.delete(rowId);
    } else {
      current.add(rowId);
    }
    setSelectedRows(current);

    if (props.onSelectionChange) {
      const selected = props.data.filter((row) =>
        current.has(row[props.rowKey] as string | number),
      );
      props.onSelectionChange(selected);
    }
  };

  const toggleSelectAll = () => {
    const current = selectedRows();
    if (current.size === filteredAndSortedData().length) {
      setSelectedRows(new Set());
      props.onSelectionChange?.([]);
    } else {
      const allIds = new Set(
        filteredAndSortedData().map(
          (row) => row[props.rowKey] as string | number,
        ),
      );
      setSelectedRows(allIds);
      props.onSelectionChange?.(filteredAndSortedData());
    }
  };

  const isAllSelected = () => {
    const data = filteredAndSortedData();
    return data.length > 0 && selectedRows().size === data.length;
  };

  const handleSort = (column: keyof T) => {
    if (sortColumn() === column) {
      setSortDirection(sortDirection() === "asc" ? "desc" : "asc");
    } else {
      setSortColumn(column);
      setSortDirection("asc");
    }
  };

  const filteredAndSortedData = () => {
    let result = [...props.data];

    // Apply filters
    const currentFilters = filters();
    Object.keys(currentFilters).forEach((key) => {
      const filterValue = currentFilters[key as keyof T];
      if (filterValue) {
        result = result.filter((row) => {
          const cellValue = String(row[key as keyof T]).toLowerCase();
          return cellValue.includes(filterValue.toLowerCase());
        });
      }
    });

    // Apply sorting
    const sortCol = sortColumn();
    if (sortCol) {
      result.sort((a, b) =>
        compareValues(a[sortCol], b[sortCol], sortDirection()),
      );
    }

    return result;
  };

  return (
    <Table class={props.class}>
      <TableHeader>
        <TableRow>
          <Show when={props.selectable}>
            <TableHead class="w-12">
              <input
                type="checkbox"
                checked={isAllSelected()}
                onChange={toggleSelectAll}
                class="cursor-pointer accent-primary"
              />
            </TableHead>
          </Show>
          <For each={props.columns}>
            {(column) => (
              <TableHead class={column.class}>
                <article
                  class={`flex items-center justify-between gap-2 ${column.sortable ? "cursor-pointer select-none" : ""}`}
                  onClick={() => column.sortable && handleSort(column.key)}
                >
                  <span>{column.label}</span>
                  <Show when={column.sortable}>
                    <section class="flex flex-col">
                      <Icon
                        name="caret-up"
                        size="xs"
                        class={
                          sortColumn() === column.key &&
                          sortDirection() === "asc"
                            ? "text-primary"
                            : "text-muted-foreground"
                        }
                      />
                      <Icon
                        name="caret-down"
                        size="xs"
                        class={`-mt-1 ${
                          sortColumn() === column.key &&
                          sortDirection() === "desc"
                            ? "text-primary"
                            : "text-muted-foreground"
                        }`}
                      />
                    </section>
                  </Show>
                </article>
              </TableHead>
            )}
          </For>
        </TableRow>
      </TableHeader>
      <TableBody>
        <For each={filteredAndSortedData()}>
          {(row) => {
            const rowId = row[props.rowKey] as string | number;

            return (
              <TableRow
                data-state={selectedRows().has(rowId) ? "selected" : undefined}
                class={selectedRows().has(rowId) ? "bg-muted" : ""}
              >
                <Show when={props.selectable}>
                  <TableCell class="w-12">
                    <input
                      type="checkbox"
                      checked={selectedRows().has(rowId)}
                      onChange={() => toggleRowSelection(rowId)}
                      class="cursor-pointer accent-primary"
                      onClick={(e) => e.stopPropagation()}
                    />
                  </TableCell>
                </Show>
                <For each={props.columns}>
                  {(column) => (
                    <TableCell class={column.class}>
                      {column.component ? (
                        <column.component value={row[column.key]} row={row} />
                      ) : column.render ? (
                        column.render(row[column.key], row)
                      ) : (
                        String(row[column.key])
                      )}
                    </TableCell>
                  )}
                </For>
              </TableRow>
            );
          }}
        </For>
      </TableBody>
    </Table>
  );
}
