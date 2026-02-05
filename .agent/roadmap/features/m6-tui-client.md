# M6 -- TUI Client

**Priority**: Medium
**Status**: Not started
**Depends on**: M3 (reuses HTTP client), M4 (build progress display)

## Goal

Implement the Terminal User Interface using Bubble Tea, launched via `ldfctl --tui`.

## Tasks

### 6.1 TUI Foundation
- [ ] Set up Bubble Tea framework in `src/ldfctl/internal/tui/`
- [ ] Implement navigation model (main menu, views, modals)
- [ ] Reuse ldfctl HTTP client for API communication
- [ ] Implement auth flow in TUI
- **Files**: `src/ldfctl/internal/tui/`

### 6.2 TUI Views
- [ ] Dashboard: distributions overview, recent builds, system status
- [ ] Distribution list/detail/create/edit
- [ ] Component browser with version selection
- [ ] Source management
- [ ] Build progress with real-time log streaming
- [ ] Settings panel
- **Files**: `src/ldfctl/internal/tui/views/`

### 6.3 TUI Polish
- [ ] Keyboard shortcuts and help overlay
- [ ] Theme support (consistent with WebUI theming)
- [ ] Responsive layout for different terminal sizes
- [ ] Wire as `ldfctl --tui` entry point
- **Files**: `src/ldfctl/internal/tui/`, `src/ldfctl/internal/cmd/`

## Acceptance Criteria

- `ldfctl --tui` launches a navigable terminal interface
- All major CRUD operations available
- Build progress visible in real-time
- Works in standard 80x24 terminal and scales to larger sizes
