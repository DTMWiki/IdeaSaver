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
- Playback info retrieval after transcode callback
- Web playback via DogeCloud JS Player (`vcode + userId`) with native `<video>` fallback
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

## Repository Governance and Release

- GitHub branch protection via Rulesets: [deploy/github-rulesets.md](./deploy/github-rulesets.md)
- Release policy (flow, gates, tagging, hotfix): [deploy/release-policy.md](./deploy/release-policy.md)
- Server release checklist: [deploy/release-checklist.md](./deploy/release-checklist.md)
- Rollback plan: [deploy/rollback-plan.md](./deploy/rollback-plan.md)
- Manual acceptance script: [deploy/manual-acceptance.md](./deploy/manual-acceptance.md)

## Contributing

- The default development branch is `dev`, `canary` is for pre-release validation, and `master` is the protected production branch.
- Run the minimum validation set before opening a PR:
  - `go test ./...`
  - `go build ./...`
  - `cd web && npm run lint && npm run build`
- If a change touches deployment, authentication, OSS/VCloud integration, or audit logging, update the related files under `deploy/` as part of the same change.
- Prefer source-package delivery plus on-server build for production releases instead of shipping local build artifacts.

## Contributors

- Maintained by DTMWiki
- Contributions are welcome through GitHub Issues and Pull Requests, especially for features, bug fixes, documentation, deployment notes, and security hardening.

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE) (`AGPL-3.0-only`).
