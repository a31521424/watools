# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**WaTools** is a cross-platform desktop application launcher built with **Wails v2** (Go backend + React/TypeScript
frontend). It provides a command palette-style interface for launching applications and executing system operations via
global hotkeys, inspired by uTools and Alfred.

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
- **Architecture Philosophy**: Plugins are self-contained and communicate via well-defined context objects

#### Plugin Context Pattern

```typescript
// Frontend: frontend/src/schemas/plugin.ts
export type PluginContext = {
    input: AppInput           // User input data
    clipboard: AppClipboardContent | null  // Clipboard snapshot
}

export type PluginEntry = {
    type: "executable" | "ui"
    match: (context: PluginContext) => boolean
    execute?: (context: PluginContext) => Promise<void>
}
```

**Design Principles**:

- ✅ Match and execute use the same context (React closure guarantees consistency)
- ✅ No manual snapshot needed - useMemo handles re-rendering
- ✅ Simple object reference passing (no function callbacks)
- ❌ Avoid over-engineering - trust React's rendering lifecycle

#### Plugin API Exposure

Plugins run in iframes and access backend APIs via `window.watools`:

```javascript
// Frontend: frontend/src/api/api.ts
export const WaApi = {
    OpenFolder,           // Open folder in file manager
    SaveBase64Image,      // Save base64 image to downloads
    HttpProxy            // Generic HTTP proxy (bypasses CORS)
}

// Plugin usage:
await window.watools.HttpProxy({
    url: 'https://api.example.com',
    method: 'POST',
    headers: {'Authorization': 'Bearer token'},
    body: JSON.stringify(data),
    timeout: 30000
});
```

**Security Model**:

- Only whitelisted APIs exposed to iframe plugins
- No direct access to `window.go` internals
- Backend validates all requests

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

#### Creating a New Plugin

```bash
# 1. Create plugin directory
mkdir fronted-plugin/watools.plugin.yourplugin

# 2. Create manifest.json
{
  "packageId": "watools.plugin.yourplugin",
  "name": "Your Plugin",
  "description": "Plugin description",
  "version": "0.0.1",
  "author": "Your Name",
  "uiEnabled": true,
  "entry": "app.js"
}

# 3. Create app.js (entry point)
const entry = [{
    type: "ui",  // or "executable"
    subTitle: "Plugin action",
    icon: "icon-name",
    match: (context) => {
        // Return true if plugin should handle this input
        return context.input.value.startsWith('keyword');
    },
    file: "index.html"  // For UI plugins
}];
export default entry;

# 4. For UI plugins, create index.html
# Use window.watools.HttpProxy for external API calls
```

#### HTTP Proxy for CORS-Free API Calls

**Problem**: Plugins run in iframes and face CORS restrictions when calling external APIs.

**Solution**: Use the generic `HttpProxy` backend API that acts as a proxy:

```javascript
// ❌ Direct fetch (CORS error)
fetch('https://api.external.com/data')  // CORS blocked!

// ✅ Use HttpProxy (no CORS)
const response = await window.watools.HttpProxy({
    url: 'https://api.external.com/data',
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer your-api-key'
    },
    body: JSON.stringify({ query: 'data' }),
    timeout: 30000  // milliseconds
});

if (response.status_code === 200) {
    const data = JSON.parse(response.body);
    // Process data
}
```

**Architecture**:

```
Frontend (iframe) → window.watools.HttpProxy()
                 → coordinator.HttpProxyApi()
                 → api.HttpProxy()
                 → External API (no CORS)
```

**Key Benefits**:

- ✅ Generic API - works for ANY external service
- ✅ No need to add backend code for each new API
- ✅ Plugins remain self-contained
- ✅ Follows Open/Closed Principle (open for extension, closed for modification)

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

#### Architecture Decisions

1. **Generic vs Specialized APIs**:
    - ✅ Prefer generic APIs (e.g., `HttpProxy`) over specialized ones (e.g., `TranslateTextApi`)
    - ✅ Let plugins implement business logic, backend provides infrastructure
    - ❌ Avoid adding backend code for each new plugin requirement

2. **Plugin Context Pattern**:
    - ✅ Use simple object passing: `match(context)` and `execute(context)`
    - ✅ Trust React's useMemo for consistency - no manual snapshots needed
    - ❌ Avoid complex closure patterns or function callbacks

3. **API Organization**:
    - Plugin APIs belong in `internal/api/api.go` (infrastructure layer)
    - Exposed to frontend via `frontend/src/api/api.ts`
    - Mounted to iframe via `window.watools` (security boundary)

#### Security & Best Practices

- **Security**: Whitelist safe Wails runtime APIs (clipboard, window controls, browser operations)
- **Content Strategy**: Serialize small content, provide URLs for large binary content
- **Function Mounting**: Direct iframe window attachment for maximum performance
- **Input Processing**: Snapshot clipboard content at application activation to ensure consistency
- **CORS Handling**: Use generic HttpProxy for all external API calls - never expose API keys in frontend code

## Example Plugins

### Translator Plugin (`watools.plugin.translator`)

A production-ready example demonstrating plugin best practices.

**Features**:

- Multi-language translation via DeepL API
- Auto language detection
- Translation history
- Keyboard shortcuts (Cmd+Enter to translate, Cmd+C to copy)

**Key Implementation Details**:

```javascript
// app.js - Plugin entry
const entry = [{
    type: "ui",
    subTitle: "Translate with DeepL",
    icon: "languages",
    match: (context) => {
        // Match on keywords or any text input
        const keywords = ['translate', 'trans', '翻译'];
        return keywords.some(kw =>
            context.input.value.toLowerCase().startsWith(kw)
        );
    },
    file: "index.html"
}];

// index.html - API call using HttpProxy
async function translate() {
    const formData = new URLSearchParams({
        auth_key: apiKey,
        text: sourceText,
        target_lang: targetLang
    });

    const response = await window.watools.HttpProxy({
        url: 'https://api-free.deepl.com/v2/translate',
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: formData.toString(),
        timeout: 30000
    });

    if (response.status_code === 200) {
        const data = JSON.parse(response.body);
        // Display translation
    }
}
```

**Lessons Learned**:

- Store API keys in localStorage (user-provided)
- Use generic HttpProxy instead of specialized backend APIs
- Implement proper error handling and loading states
- Follow the plugin context pattern for consistency
