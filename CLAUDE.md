# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ImageSpider is a web image crawler with a Go backend and Vue 3 frontend. It crawls image boards (like yande.re), downloads images, converts them to formats like AVIF, and provides a web interface for browsing and managing images.

## Tech Stack

- **Backend**: Go 1.23+ with Gin web framework, PostgreSQL database
- **Frontend**: Vue 3 + TypeScript + Vite + Ant Design Vue 4 + Pinia
- **Build**: Shell script (`build.sh`)

## Development Commands

### Full Build
```bash
./build.sh           # Build both frontend and backend
./build.sh frontend  # Build only frontend
./build.sh backend   # Build only backend
./build.sh debug     # Debug build (unminified, source maps)
./build.sh clean     # Clean build artifacts
```

### Frontend Development
```bash
cd frontend
npm install
npm run dev          # Dev server at http://localhost:5173
npm run build        # Production build to embed/www/
npm run build:debug  # Debug build with source maps
```

### Backend Development
```bash
go run . -c config.yml    # Run with config
go build -o imagespider . # Build binary
```

### Run Built Binary
```bash
./imagespider -c config.yml
```

## Architecture

### Plugin System

The backend uses a plugin-based architecture. Each plugin:
- Registers itself in `init()` via `interfaces.Plugins[plugin.ID()] = plugin`
- Implements `interfaces.IPlugin` (Name, ID, Load, Unload, GetService)
- Can depend on other plugins via `app.GetService(callerID, targetID, serviceID)`

Plugins are loaded in order defined by `plugins:` in `config.yml`. The `GetService` call creates a dependency graph; plugins are unloaded in reverse dependency order on shutdown.

Key plugins (see `interfaces/IPlugin.go` for IDs):
- `DB` - Database service (PostgreSQL)
- `spider` - Web crawler for image boards
- `imageDownloader` - Downloads images from URLs
- `imageConvert` - Converts images to AVIF/WebP with thumbnails
- `dataChecker` - Validates database integrity
- `fileIntegrityChecker` - Validates image files on disk
- `API` - HTTP REST API and static file serving

### Frontend Integration

The frontend is built to `embed/www/` and embedded into the Go binary via `//go:embed` directives in `embed/embed.go`. The API serves static files from this embedded filesystem at `/www/`.

During development, Vite dev server proxies `/api` and `/image` paths to `http://localhost:8080` (configured in `vite.config.ts`).

### Configuration

Configuration is loaded from a YAML file (`-c config.yml` or `CONFIG_PATH` env var). Key sections:
- `plugins` - List of plugin IDs to load
- `spiders` - Crawler configurations per source (selectors, headers, retry settings)
- `imageConverter` - Output format, quality, thumbnail sizes
- `database` - PostgreSQL connection string
- `api` - HTTP port (default 8080)
- `imageDir` - Where images are stored
- `workDir` - Runtime state directory

Runtime config (paging state, retry counts) is stored in `workDir/runtimeConfig.yaml`.

## Key File Locations

- Interfaces: `interfaces/` - Plugin and service interfaces
- Plugins: `plugins/` - Plugin implementations (API.go, Spider.go, DB.go, etc.)
- Models: `models/` - Data structures and config types
- Frontend: `frontend/src/` - Vue components, stores, API clients
- Embedded assets: `embed/` - SQL init scripts, web content

## API Structure

The REST API (`plugins/API.go`) serves:
- `/api/sources` - List available image sources
- `/api/:sourceid/tags` - Get tags for a source
- `/api/:sourceid/images` - List/filter images
- `/api/:sourceid/image/:id` - Get image details
- `/api/:sourceid/images/redownload` - Batch redownload
- `/api/:sourceid/tags/:tag/cover` - Set cover image for tag
- `/image/*` - Serve image files
- `/www/*` - Serve frontend (embedded)

## Database

PostgreSQL is used. Schema is initialized from `embed/res/dbinit.sql`. The DB plugin provides `interfaces.IDBService` for other plugins.
