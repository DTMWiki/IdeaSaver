# IdeaSaver 安全漏洞与代码问题审查报告

> 审查时间：2026-07-23
> 审查范围：Go 后端（server/）、React 前端（web/src/）、CLI（cli/）、部署配置（deploy/）
> 总体评价：代码质量整体不错——SQL 全部参数化、鉴权/越权检查基本到位、分享密码 bcrypt、OAuth state 与下载 token 处理规范、`.env` 未进入 git 历史。

---

## 高危

### 1. 存储型 XSS：用户上传的 HTML/SVG/JS 以同源内联方式返回

- `server/internal/service/upload_service.go:469` 的 `getMimeType` 把 `.html→text/html`、`.svg→image/svg+xml`、`.js/.css` 映射为可执行类型，上传时写入 OSS。
- `server/internal/handler/direct_sse.go:41`（`/s/:user_id/:filename` 公开直链）和 `server/internal/handler/files.go:244`（预览）直接使用 OSS 的 Content-Type 返回，无 `Content-Disposition`、无 CSP。
- 更严重的是 `upload_service.go:101` multipart 初始化时直接信任客户端传入的 `req.MimeType`，连扩展名限制都可绕过。

**攻击场景**：任何注册用户上传一个 `.html` 文件，把公开直链发给其他登录用户。该页面与主站同源，其中的脚本可读取页面 DOM、并以受害者身份（HttpOnly cookie 自动携带）调用全部 API——删文件、建分享、改视频状态，等同账户接管。同时这也是免费的钓鱼页托管。

**修复建议**：
- 对 `text/html`、`image/svg+xml`、`text/javascript` 等危险类型强制 `Content-Disposition: attachment` 或降级为 `application/octet-stream`；
- 忽略客户端传入的 `mime_type`，仅按扩展名白名单决定；
- 长期方案：给 `/s/` 直链使用独立域名，并加 `Content-Security-Policy: sandbox`。

---

## 中危

### 2. 分享密码缺少针对性限流/锁定

- 位置：`server/internal/handler/routes.go:28-29`
- `/api/shares/:code` 和 `/download` 只有全局 300 次/分/IP 限制，分布式 IP 下短密码可被爆破；每次 bcrypt 校验本身也是 CPU 消耗。
- 建议：对单个 code 加失败计数锁定，或更严格的独立限流（如 10 次/分/IP/code）。

### 3. 视频上传绕过单文件大小限制、配额按客户端声明值扣

- 位置：`server/internal/service/video_service.go:56-63`
- 只查配额，不像分片上传那样校验 `MaxUploadSizeMB`；`header.Size` 是客户端可伪造的声明值，直接用于配额扣减和 `ContentLength`。

### 4. 缺安全响应头

- 位置：`deploy/nginx/ideasaver.conf` 及应用层
- 未设置 `Content-Security-Policy`、`X-Content-Type-Options`、`X-Frame-Options`、`Strict-Transport-Security`、`Referrer-Policy`。
- 其中 `X-Content-Type-Options: nosniff` 与高危 #1 直接相关，应优先加。

### 5. SSE 连接数无上限

- 位置：`server/internal/handler/direct_sse.go:60`
- 已认证用户可无限挂长连接（每条一个 goroutine + channel），全局限流挡不住已建立连接。
- 建议：限制每用户并发 SSE 连接数。

---

## 低危 / 代码质量

### 6. 分页 `limit` 无上限

- 位置：`server/internal/handler/auth.go:126`（`/user/history`）及 `admin.go` 各列表接口
- `limit=100000000` 直接传到 SQL，可拖慢数据库。建议 clamp 到 1–200。

### 7. JWT 未显式限定签名算法

- 位置：`server/internal/middleware/middleware.go:61`
- `jwt.ParseWithClaims` 未加 `jwt.WithValidMethods([]string{"HS256"})`。当前因密钥类型不匹配不可利用，但属纵深防御缺项（分享下载 token 的校验 `share_service.go:243` 反而做了，建议对齐）。

### 8. 登出无吊销

- 位置：`server/internal/handler/auth.go:72` `handleLogout`
- 只清 cookie，已泄露的 JWT 在 24h 内仍有效。可考虑缩短有效期 + refresh token，或维护吊销列表。

### 9. 视频回调 secret 允许 query 传参

- 位置：`server/internal/handler/videos.go:216`
- `?secret=` 会留在 nginx access log 里。建议仅接受 header。

### 10. 批量操作部分失败被吞

- 位置：`server/internal/handler/files.go:153`（批量删除逐条忽略错误）、`server/internal/service/video_service.go:278`（批量删视频跳过不存在的 ID、且多条 DELETE 无事务）
- 调用方无法感知部分失败。

### 11. 第三方播放器脚本无完整性校验

- 位置：`web/src/utils/dogePlayer.ts:3`
- 从 `player.dogecloud.com` 动态加载 JS，无 SRI；该 CDN 被控即在站内执行任意脚本。

### 12. 错误白名单用子串匹配

- 位置：`server/internal/handler/respond.go:32`
- `isClientFacingError` 只要错误消息含"分享/上传/无效"等词就原样返回给客户端，内部错误恰好含这些词时会泄露实现细节。建议用 sentinel error 类型判断。

### 13. 其他小问题

- `cli/main.go:151` `config set` 未校验 value 中的换行符，可注入额外配置行（本地工具，影响有限）。
- `server/internal/middleware/ratelimit.go` 的 visitors map 无硬上限，靠 5 分钟清理兜底，极端 IP 数量下内存可增长。
- `handleUploadChunk`（`server/internal/handler/upload.go:40`）忽略 `strconv.Atoi` 错误（服务层兜底了范围校验，属坏味道）。

---

## 值得肯定的地方

- 仓储层全部参数化查询，无 SQL 注入；排序/过滤拼接均为固定常量。
- 文件/分享/视频操作几乎都有所有者校验，未见明显 IDOR。
- OAuth state 一次性 cookie + 常量时间比较；分享密码 bcrypt；下载 token 强制 HS256 + `typ` 检查；分享密码只经 header 传输、不落日志。
- `.env` 已被 gitignore 且未进 git 历史；JWT secret 为启动必检项；trusted proxies 默认不信任。

---

## 优先修复顺序建议

1. #1 同源 XSS（最高优先）
2. #4 安全响应头（半天工作量）
3. #2 分享密码限流、#3 视频上传大小校验
4. 其余择机处理

---

## 修复状态（2026-07-23 落地）

| # | 状态 | 说明 |
|---|------|------|
| 1 | **已修** | 危险 MIME 存为 octet-stream；直链/预览/分享下载强制 attachment + nosniff；忽略客户端 mime_type |
| 2 | **已修** | `SharePasswordLimit` 10 次/分/IP+code，失败计数，成功清零 |
| 3 | **已修** | 视频上传校验 `MaxUploadSizeMB` 与 size>0 |
| 4 | **已修** | Go `SecurityHeaders` 中间件 + nginx 头 |
| 5 | **已修** | 每用户最多 5 条 SSE 连接 |
| 6 | **已修** | `parseOffsetLimit` clamp 1–200 |
| 7 | **已修** | JWT `WithValidMethods(HS256)` + method 检查 |
| 8 | **已修** | `token_version` 登出递增，旧 JWT 失效 |
| 9 | **已修** | 回调 secret 仅 header |
| 10 | **部分** | 批量软删返回 failed 列表；视频已 validatedIDs |
| 11 | **部分** | Doge 脚本 `crossOrigin`+`referrerPolicy`；SRI 因 CDN 无版本哈希未钉死；CSP 限 host |
| 12 | **已修** | `ClientError` + 短中文消息白名单（禁 failed/sql 等） |
| 13 | **已修** | CLI 禁换行；ratelimit map 硬上限；chunk_index Atoi 校验 |
