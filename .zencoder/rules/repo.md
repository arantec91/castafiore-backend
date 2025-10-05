---
description: Repository Information Overview
alwaysApply: true
---

# Castafiore Backend Information

## Summary
Castafiore Backend is a Go-based music streaming server with Subsonic API compatibility. It provides advanced features including user subscription plans, concurrent connection limits, download tracking, and a PostgreSQL-driven architecture.

## Structure
- **cmd/**: Contains main entry points (server and scanner)
- **internal/**: Core application code organized by domain
- **migrations/**: SQL database schema and sample data
- **web/**: Web interface templates
- **scripts/**: Utility scripts for administration
- **config/**: Configuration files

## Language & Runtime
**Language**: Go
**Version**: 1.21
**Build System**: Go modules
**Package Manager**: Go modules

## Dependencies
**Main Dependencies**:
- github.com/gin-gonic/gin: Web framework
- github.com/golang-jwt/jwt/v5: JWT authentication
- github.com/lib/pq: PostgreSQL driver
- golang.org/x/crypto: Cryptography functions
- github.com/dhowden/tag: Audio file metadata parsing

## Build & Installation
```bash
# Install dependencies
go mod download

# Build the server
go build -o bin/castafiore.exe cmd/server/main.go

# Build the scanner
go build -o bin/scanner.exe cmd/scanner/main.go

# Run the server
go run cmd/server/main.go
```

## Docker
**Configuration**: docker-compose.yml
**Services**:
- PostgreSQL 15 (database)
- Redis 7 (optional caching)
- Adminer (database management)

**Run Command**:
```bash
docker-compose up -d
```

## Database
**Type**: PostgreSQL
**Schema**: migrations/001_initial_schema.sql
**Main Tables**:
- users: User accounts and subscription plans
- artists: Music artists
- albums: Music albums
- songs: Individual tracks
- user_sessions: Active sessions for concurrency control
- downloads: Download history for daily limits

## Testing
**Run Command**:
```bash
go test ./...
```

## Environment Configuration
**File**: .env (from .env.example)
**Key Variables**:
- HOST/PORT: Network configuration
- DATABASE_URL: PostgreSQL connection string
- JWT_SECRET: Authentication secret
- MUSIC_PATH: Path to music library
- MAX_CONCURRENT_STREAMS: Limit per user
- MAX_DOWNLOADS_PER_DAY: Daily download limit

## API
**Type**: Subsonic API v1.16.1
**Authentication**: JWT + Subsonic salt/token
**Key Endpoints**:
- /rest/ping: Connectivity test
- /rest/getLicense: License information
- /rest/getMusicFolders: Music folders
- /rest/getIndexes: Artist index
- /rest/stream: Music streaming
- /rest/download: File downloads