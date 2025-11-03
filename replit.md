# GoLANshare - Replit Project Documentation

## Overview
GoLANshare is a secure local file sharing application built with Go backend and vanilla JavaScript frontend. It enables fast file transfers over local networks with end-to-end encryption, real-time chat, and device discovery features.

## Project Status
- **Status**: Fully configured and running on Replit
- **Last Updated**: November 3, 2025
- **Port**: 5000 (frontend and backend both served by Go server)

## Architecture

### Backend (Go)
- **Location**: `/backend` directory
- **Entry Point**: `backend/main.go`
- **Go Version**: 1.21
- **Port**: 5000 (configured in `config.json`)
- **Host**: 0.0.0.0 (allows Replit proxy access)
- **Database**: SQLite (`backend/storage/database.db`)

### Frontend
- **Location**: `/frontend` directory
- **Technology**: Vanilla HTML/CSS/JavaScript
- **Served by**: Go backend (static file serving)
- **Entry Point**: `frontend/index.html`

### Key Features
- File upload/download with chunked transfer support
- Real-time WebSocket communication
- Device discovery via mDNS
- JWT-based authentication
- Built-in chat system
- Progressive Web App capabilities

## Dependencies

### Go Modules
```
github.com/golang-jwt/jwt/v4 v4.5.0
github.com/gorilla/websocket v1.5.0
github.com/hashicorp/mdns v1.0.5
github.com/mattn/go-sqlite3 v1.14.22
github.com/robfig/cron/v3 v3.0.0
golang.org/x/crypto v0.14.0
```

## Configuration

### config.json
```json
{
  "port": "5000",
  "host": "0.0.0.0",
  "enable_tls": false,
  "require_auth": true,
  "storage_path": "./storage",
  "upload_path": "./storage/uploads",
  "max_upload_size": 1073741824
}
```

### Workflow
- **Name**: golanshare
- **Command**: `cd backend && go run .`
- **Output**: webview
- **Port**: 5000

## Recent Changes (November 3, 2025)

1. **Port Configuration**: Changed from 8081 to 5000 for Replit compatibility
2. **Cache Control**: Added cache-control headers to prevent stale content in iframe preview
3. **JavaScript Fix**: Added null checks in `renderTransfers()` function to prevent errors
4. **Git Ignore**: Created comprehensive .gitignore for Go projects
5. **Workflow Setup**: Configured workflow to run Go server on port 5000

## Known Issues

### mDNS IPv6 Errors (Non-Critical)
The application shows mDNS IPv6 errors in logs. These are expected in containerized environments and don't affect core functionality:
- Device discovery may be limited in Replit environment
- All other features (file transfer, chat, auth) work normally

### Browser Caching
Due to aggressive browser caching in Replit's iframe preview:
- Cache-control headers are now set to prevent stale content
- If changes don't appear, users may need to do a hard refresh

## Default Credentials
- **Username**: admin
- **Password**: admin

Note: These can be changed after first login via the profile settings.

## File Structure
```
.
├── backend/
│   ├── storage/
│   │   └── database.db
│   ├── auth.go           # Authentication handlers
│   ├── chat.go           # Chat functionality
│   ├── db.go             # Database operations
│   ├── discovery.go      # Device discovery (mDNS)
│   ├── go.mod            # Go dependencies
│   ├── go.sum            # Go checksum
│   ├── handlers.go       # HTTP handlers
│   ├── main.go           # Main entry point
│   ├── transfer.go       # File transfer logic
│   └── utils.go          # Utility functions
├── frontend/
│   ├── css/
│   │   └── style.css
│   ├── js/
│   │   └── app.js
│   ├── index.html        # Main HTML
│   └── sw.js             # Service worker (PWA)
├── config.json           # Server configuration
├── .gitignore
└── README.md
```

## Running Locally (Outside Replit)
```bash
cd backend
go mod download
go run .
# Open browser to http://localhost:5000
```

## Deployment Notes
- Ready for deployment as a VM (stateful application)
- Database is SQLite-based (file storage)
- WebSocket connections require persistent server
- mDNS discovery may not work in all cloud environments
