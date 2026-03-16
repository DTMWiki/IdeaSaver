# IdeaSaver 部署文档

## 环境要求

- Debian 12 (Bookworm)
- Go 1.23+ (仅编译时需要)
- PostgreSQL 15+
- Nginx (已安装)
- 域名 `i.dtmwiki.cn` 已解析

---

## 1. 创建系统用户

```bash
sudo useradd -r -s /usr/sbin/nologin -d /opt/ideasaver ideasaver
sudo mkdir -p /opt/ideasaver /etc/ideasaver
sudo chown ideasaver:ideasaver /opt/ideasaver
```

## 2. 构建项目

在开发机上编译（交叉编译 Linux amd64）：

```bash
# 后端
cd server
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ideasaver cmd/main.go

# CLI 工具
cd ../cli
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ideactl main.go

# 前端
cd ../web
npm install && npm run build
```

## 3. 上传文件到服务器

```bash
scp server/ideasaver user@server:/opt/ideasaver/
scp cli/ideactl user@server:/usr/local/bin/
scp -r web/dist user@server:/opt/ideasaver/web/
```

如果你使用源码包上传，可先在本地执行：

```powershell
powershell -ExecutionPolicy Bypass -File deploy/scripts/package_source.ps1
```

## 4. 配置数据库

```sql
CREATE USER ideasaver WITH PASSWORD 'your-secure-password';
CREATE DATABASE ideasaver OWNER ideasaver;
```

## 5. 配置环境变量

```bash
sudo cp server/.env.example /etc/ideasaver/.env
sudo chmod 600 /etc/ideasaver/.env
sudo chown ideasaver:ideasaver /etc/ideasaver/.env
sudo nano /etc/ideasaver/.env   # 填入实际配置
```

## 6. 执行数据库迁移

```bash
ideactl db migrate
```

## 7. 部署 systemd 服务

```bash
sudo cp deploy/systemd/ideasaver.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable ideasaver
ideactl start
ideactl status
```

## 8. 配置 Nginx

```bash
sudo cp deploy/nginx/ideasaver.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/ideasaver.conf /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## 9. 配置 Authelia

在 Authelia 配置中添加 IdeaSaver 作为 OIDC 客户端：

```yaml
identity_providers:
  oidc:
    clients:
      - client_id: ideasaver
        client_secret: 'your-hashed-secret'
        redirect_uris:
          - https://i.dtmwiki.cn/api/auth/callback
        scopes:
          - openid
          - profile
          - email
          - groups
```

## 10. 验证部署

```bash
ideactl status       # 检查服务运行状态
ideactl logs         # 查看实时日志
curl -s https://i.dtmwiki.cn/api/auth/login | jq  # 验证 API 可达
bash deploy/scripts/smoke_test.sh                  # 执行冒烟脚本
```

---

## 发布与回滚文档

- GitHub Rulesets 配置: [github-rulesets.md](./github-rulesets.md)
- 发布规范: [release-policy.md](./release-policy.md)
- 发布清单: [release-checklist.md](./release-checklist.md)
- 回滚预案: [rollback-plan.md](./rollback-plan.md)
- 手动验收脚本: [manual-acceptance.md](./manual-acceptance.md)

---

## 日常维护

```bash
ideactl start/stop/restart   # 服务管理
ideactl logs --lines=100     # 查看日志
ideactl config list          # 查看配置
ideactl config set KEY VAL   # 修改配置（需 restart 生效）
```
