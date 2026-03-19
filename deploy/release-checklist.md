# IdeaSaver 服务器发布清单

## 0. 前置准备

- 确认代码已拉到目标版本（包含 `go.mod/go.sum`、`server/migrations/002_file_moderation.sql`）。
- 确认服务器有 Go 1.23+、Node.js 20+（如果服务器本地构建前端）。
- 记录当前线上版本号（Git commit / 发布包名）。

## 1. 发布前检查

- 备份数据库（强烈建议）：
  - `pg_dump "$IDEASAVER_DATABASE_URL" -Fc -f /var/backups/ideasaver_$(date +%F_%H%M%S).dump`
- 备份当前二进制和前端静态文件目录。
- 确认 `.env` 中关键配置完整：数据库、OSS、Authelia、JWT、PublicBaseURL。

## 2. 构建与制品准备

- 后端：
  - `go test ./...`
  - `go build -o ideasaver ./server/cmd`
- 前端：
  - `cd web && npm ci && npm run build`

## 3. 停机与部署

- `sudo systemctl stop ideasaver`
- 替换后端二进制到部署目录（如 `/data/ideasaver/ideasaver`）。
- 替换前端静态资源（如 `web/dist` 到 Nginx/服务目录）。

## 4. 数据迁移

- 执行迁移：
  - `/data/ideasaver/ideasaver migrate`
- 确认迁移成功后再启动服务。

## 5. 启动与验证

- `sudo systemctl start ideasaver`
- `sudo systemctl status ideasaver --no-pager`
- 跑验收脚本：
  - `bash deploy/scripts/smoke_test.sh`

## 6. 发布后观察（建议 30 分钟）

- `journalctl -u ideasaver -f`
- 观察 5xx 比例、响应时间、登录成功率、上传/预览成功率。
- 重点抽查新增流程：
  - 文件封禁后直链是否重定向禁止图片
  - 用户提交申诉工单
  - 管理员审核通过/删除并产生审计日志
