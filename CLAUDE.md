# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**WaTools** is a cross-platform desktop application launcher built with **Wails v2** (Go backend + React/TypeScript frontend). It provides a command palette-style interface for launching applications and executing system operations via global hotkeys, inspired by uTools and Alfred.

Key features:
- Global access via hotkey (Alt+Space)
- Application launcher with fuzzy search
- System operations (calculator, etc.)
- Cross-platform support (macOS primary target)

## Architecture

### High-Level Data Flow
1. **Global Hotkey** → App window shows/hides
2. **User Input** → Fuzzy search across applications and operations
3. **Command Selection** → Backend executes via coordinator pattern
4. **Results** → Window hides automatically after execution

### Backend (Go)
- **Framework**: Wails v2 for desktop app development
- **Language**: Go 1.23+
- **Database**: SQLite with sqlc for type-safe queries
- **Architecture**: Coordinator pattern with singleton instances

### Frontend (React/TypeScript)
- **Framework**: React 19 with TypeScript
- **Build Tool**: Vite 7.1.5
- **Package Manager**: Yarn 1.22.22
- **Styling**: Tailwind CSS 4
- **UI Components**: shadcn/ui, cmdk (command palette)

## Key Components

### Core Application (`internal/app/`)
- `app.go`: Main application lifecycle and window management
- `hotkey.go`: Global hotkey registration (platform-specific)
- Singleton pattern for application state

### Command System (`internal/command/`)
- `command.go`: Command execution engine with interfaces
- `application/`: Application discovery and launching logic
- `operator/`: Built-in system operations
- `watcher/`: File system monitoring for application changes

### Coordinator Pattern (`internal/coordinator/`)
- `coordinator.go`: Central orchestrator binding frontend to backend
- Aggregates all subsystems (app, commands) into single Wails-bound interface
- All frontend API calls go through coordinator methods

### Database Layer (`pkg/db/`)
- `wa_db.go`: Database connection, migration management, high-level operations
- `queries/*.sql`: Raw SQL queries for sqlc generation
- `migrations/`: Schema migrations with embedded files (`go:embed`)
- `*.sql.go`: Generated type-safe query functions

### Models (`pkg/models/`)
- `command.go`: Core interfaces (ApplicationCommand, OperationCommand)
- Command categories and execution patterns
- Type definitions shared between frontend and backend

### Frontend Architecture
- `components/watools/`: Main application components (WaCommand, search UI)
- `components/ui/`: Reusable shadcn/ui components
- `api/`: Wails backend communication layer
- `lib/search.ts`: Fuzzy search with Fuse.js and Pinyin support for Chinese
- `stores/`: Zustand state management stores

### Plugin System (`pkg/models/plugin.go`, `internal/plugin/`)
- **Plugin Types**: Execute plugins (backend) and UI plugins (iframe-based)
- **Plugin API**: Selective Wails runtime exposure to iframe plugins via function mounting
- **Content Handling**: Unified input system supporting text, images, files with smart transmission strategies
- **Plugin Matching**: Dynamic content-based plugin discovery and execution

## Development Workflow

### Essential Commands
```bash
# Development mode with hot reload
wails dev

# Production build
wails build -clean

# Frontend only (for UI development)
cd frontend && yarn dev

# Install frontend dependencies
cd frontend && yarn install

# Database code generation after SQL changes
sqlc generate

# Run Go tests
go test ./...

# Frontend build (TypeScript compilation + Vite)
cd frontend && yarn build
```

### Database Development
```bash
# Create new migration
migrate create -ext sql -dir pkg/db/migrations -seq migration_name

# After modifying SQL queries in pkg/db/queries/
sqlc generate

# Migrations run automatically on app startup via embedded files
```

### Build System Notes
- **Wails**: Handles both Go compilation and frontend bundling
- **Frontend**: Vite builds to `frontend/dist`, embedded via `go:embed`
- **Assets**: Static assets served through custom handler in `internal/handler/`

## Key Technologies & Dependencies

### Backend
- `github.com/wailsapp/wails/v2`: Desktop framework with Go-JS bridge
- `modernc.org/sqlite`: Pure Go SQLite driver
- `github.com/golang-migrate/migrate/v4`: Database migrations
- `github.com/rs/zerolog`: Structured logging
- `github.com/fsnotify/fsnotify`: File system watching
- `golang.design/x/hotkey`: Cross-platform global hotkeys

### Frontend
- `cmdk`: Command palette component (core UI)
- `fuse.js`: Fuzzy search implementation
- `pinyin-pro`: Chinese character search support
- `wouter`: Lightweight React router
- `zustand`: State management
- `lucide-react`: Icon system

## Platform-Specific Implementation

### macOS (Primary Target)
- Uses Cocoa APIs for application discovery in `internal/command/application/`
- Global hotkey integration with macOS system
- File system monitoring for `/Applications` and `~/Applications`
- Platform-specific files use `_darwin.go` suffix

### Cross-Platform Strategy
- Platform-specific code isolated in separate files (`_darwin.go`, `_windows.go`)
- Wails handles window management across platforms
- Database and core logic platform-agnostic
- Build system supports targeting multiple platforms

## Common Development Tasks

### Adding New System Operations
1. Create operation in `internal/command/operator/`
2. Implement command interface from `pkg/models/command.go`
3. Register in coordinator's operation command list
4. Add frontend matching logic in `components/watools/`

### Adding New Application Sources
1. Extend `internal/command/application/` with new discovery logic
2. Update database schema if needed (migrations + queries)
3. Modify watcher patterns in file system monitoring
4. Test across target platforms

### Database Schema Changes
1. Create migration: `migrate create -ext sql -dir pkg/db/migrations -seq description`
2. Add queries in `pkg/db/queries/*.sql`
3. Regenerate Go code: `sqlc generate`
4. Update database layer methods in `wa_db.go`

### Frontend Component Development
1. Follow shadcn/ui patterns for reusable components in `components/ui/`
2. Application-specific components in `components/watools/`
3. Use Tailwind classes, leverage `cn()` utility for conditional styling
4. State management through Zustand stores

### Plugin Development
1. **UI Plugins**: Create iframe-based plugins with access to limited Wails runtime API
2. **Execute Plugins**: Backend plugins with full system access via Go coordinator
3. **Content Processing**: Handle various input types (text, images, files) with 50KB boundary for performance
4. **API Exposure**: Use function mounting to iframe `window` object for secure runtime access

### Input System Design
1. **Unified Input Sources**: Manual typing, clipboard auto-read, paste events, mixed input
2. **Content Type Detection**: Automatic classification of text, long text, images, files
3. **Performance Optimization**: Direct transmission for small content (< 50KB), function access for large content
4. **Snapshot Mechanism**: Preserve input content consistency between plugin matching and execution

## Configuration Files

- `wails.json`: Wails project configuration
- `sqlc.yaml`: Database code generation settings
- `tailwind.config.ts`: Tailwind CSS configuration
- `go.mod`: Go dependencies and module definition

## Troubleshooting

### Common Issues
- **Build failures**: Ensure Go 1.23+ and Node.js 18+
- **Hotkey conflicts**: Check macOS System Preferences > Keyboard
- **Migration errors**: Verify `go:embed` paths and file permissions
- **Frontend errors**: Clear `frontend/dist` and rebuild

### Development Debugging
- Use `wails dev` for live reload and browser devtools access
- Go backend logs via zerolog (structured JSON output)
- Frontend debugging through Chrome DevTools when in dev mode

### State Management Best Practices
- **Zustand Patterns**: Use direct state access for reactivity, avoid computed functions in stores
- **Performance**: Use `useMemo` for object creation to prevent infinite re-renders
- **Debouncing**: Implement UI responsiveness with business logic debouncing (separate concerns)
- **Content Transmission**: Follow 50KB rule - direct pass for small content, function access for large content

### Plugin System Patterns
- **Security**: Whitelist safe Wails runtime APIs (clipboard, window controls, browser operations)
- **Content Strategy**: Serialize small content, provide URLs for large binary content
- **Function Mounting**: Direct iframe window attachment for maximum performance
- **Input Processing**: Snapshot clipboard content at application activation to ensure consistency