# IdeaSaver Image Web

IdeaSaver 的独立 Svelte 5 文件管理前端。它与原有 `web/` React 前端并存，复用现有 Go API，不包含新的后端服务。

## Local development

```bash
cp .env.example .env.local
npm install
npm run dev
```

`VITE_DEV_API_ORIGIN` 指向正在运行的 IdeaSaver Go 服务。Vite 会将 `/api` 与 `/s` 代理到该服务，因此浏览器仍以同源方式发送会话 Cookie。

## Production deployment

构建产物位于 `dist/`。独立域名应同时提供静态文件和以下反向代理：

```nginx
server {
    server_name image.example.com;
    root /srv/ideasaver-image-web;

    location /api/ {
        proxy_pass https://ideasaver-api.example.com/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /s/ {
        proxy_pass https://ideasaver-api.example.com/s/;
        proxy_set_header Host $host;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

保持 `VITE_API_BASE=/api`。这样不需要为浏览器跨域请求修改 Go 服务的 CORS。

OAuth 提供方的回调允许列表与 Go 服务的 `IDEASAVER_AUTHELIA_REDIRECT_URL` 必须同时设置为图床域名下的回调地址，例如：

```text
https://image.example.com/login/callback
```

Svelte 前端会读取回调中的 `code` 与 `state`，再通过同源代理调用 `/api/auth/callback` 建立 HttpOnly Cookie 会话。

## Upload recovery

上传任务元数据保存在 LocalStorage，文件对象保存在 IndexedDB。刷新或浏览器异常退出后，任务会以暂停状态恢复；用户可以继续上传。服务端仍是进度与分片状态的最终依据。

## License

本前端随仓库以 [GNU Affero General Public License v3.0](../LICENSE) 开源（`AGPL-3.0-only`）。
