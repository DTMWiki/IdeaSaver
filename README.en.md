# IdeaSaver

DTMWiki file upload and distribution platform.

Chinese README is the default: [README.md](./README.md)

## Overview

IdeaSaver is a lightweight platform for uploading, managing, and distributing files/videos.
Users log in via Authelia OAuth2, upload resources, and get direct URLs or Markdown references.

## Key Features

### File Management

- Chunked upload with pause/resume
- Folder operations (create/move/rename/copy/delete)
- Online preview (image/audio/text)
- Direct link generation and Markdown reference
- Trash and restore
- Share links with password/expiry
- File moderation and appeal workflow (ban/appeal/review)

### Video Management

- Video upload to DogeCloud VCloud
- Playback URL retrieval
- Video enable/disable/delete

### Admin

- User/admin role-based access control
- Quota management
- Global file/video management
- Appeal ticket review (approve/delete)
- Audit logs

## Tech Stack

- Backend: Go + Gin
- Frontend: React 18 + Ant Design 5 + Vite
- Database: PostgreSQL
- Storage: DogeCloud OSS (S3-compatible)
- Video: DogeCloud VCloud
- Auth: Authelia OAuth2

## Requirements

- Go 1.23+
- Node.js 20+
- PostgreSQL 15+

## Quick Start

```bash
git clone https://github.com/DTMWiki/IdeaSaver.git
cd IdeaSaver

# backend
cd server
cp .env.example .env
go mod download
go run cmd/main.go

# frontend
cd ../web
npm install
npm run dev
```

## Build

```bash
make build
make build-server
make build-web
```

## Deployment

See: [deploy/deploy.md](./deploy/deploy.md)

## License

MIT License
