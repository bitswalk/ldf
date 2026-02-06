---
paths:
  - "src/webui/**/*.{ts,tsx,css,json}"
---

# WebUI Rules

## Architecture

- SolidJS SPA with SolidJS reactive primitives
- Bun as runtime and package manager (binary at `/home/flint/.bun/bin/bun`). No npm/pnpm/yarn.
- Vite 7 as build tool with HMR -- no need to restart dev server after code changes
- TailwindCSS 4.x for all styling (themes, responsive, dark mode, filters)
- Phosphor Icons for iconography, Departure Mono as default font

## Structure

- `src/components/` -- reusable UI components (Badge, Card, Modal, Datagrid, etc.)
- `src/views/` -- page-level views injected into `<main id="viewport">`
- `src/services/` -- API client layer and state management
- `src/locales/{en,fr,de}/` -- i18n translation JSON files
- `src/themes/` -- theme definitions
- `src/lib/utils/` -- shared utility functions

## Layout

- `<header>` uses 10vh, 100% width
- `<main id="viewport">` uses 90vh, 100% width
- Views are injected into the viewport

## Conventions

- Semantic HTML5 only. Never use generic `<div>` or `<span>`. Use `<section>`, `<article>`, `<nav>`, `<aside>`, `<figure>`, or the empty SolidJS fragment `<>...</>`.
- Auth flow: check local profile -> validate JWT against ldfd -> fallback to anonymous defaults
- i18n via `@solid-primitives/i18n` with bundled dictionaries
- Dev server runs on port 3000

## Commands

```bash
cd src/webui
/home/flint/.bun/bin/bun install   # install deps
/home/flint/.bun/bin/bun run dev   # dev server with HMR
/home/flint/.bun/bin/bun run build # production build
/home/flint/.bun/bin/bun run serve # preview production build
```
