<p align="center">
  <h1 align="center">📦 IdeaSaver</h1>
  <p align="center">DTMWiki 文件上传分发平台</p>
  <p align="center"><a href="./README.en.md">English</a> | 中文（默认）</p>
</p>

<p align="center">
  <a href="#功能特性">功能特性</a> •
  <a href="#技术栈">技术栈</a> •
  <a href="#快速开始">快速开始</a> •
  <a href="#部署指南">部署指南</a> •
  <a href="#CLI-工具">CLI 工具</a> •
  <a href="#许可证">许可证</a>
</p>

---

## 简介

IdeaSaver 是由 DTMWiki 开发的轻量级文件上传分发平台。用户通过浏览器访问 `i.dtmwiki.cn`，使用 Authelia OAuth2 统一登录后，即可上传文件和视频，获取直链 URL 并方便地在 Markdown 中引用。

## 功能特性

### 文件管理

- 🚀 **快捷上传** — 拖拽或选择文件/文件夹，支持分片上传、断点续传、多任务并发
- 📁 **文件管理器** — 新建/上传/复制/剪切/粘贴/移动/重命名/删除/批量操作
- 👁️ **文件预览** — 文本文件（代码高亮）、图片、音频在线预览
- 🔗 **直链生成** — 上传完成后自动生成直链 URL 和 Markdown 引用
- 🗑️ **回收站** — 误删文件可恢复，30天自动清理
- 📤 **分享链接** — 带密码/有效期的分享短链
- 🚫 **资源封禁与申诉** — 管理员可封禁文件，用户可提交申诉，管理员复审通过或彻底删除

### 视频管理

- 🎬 **视频上传** — 视频文件自动上传到多吉云视频云
- ▶️ **播放地址** — 转码完成后自动获取播放信息（支持回调刷新）
- 📺 **Web 播放器** — 优先使用 DogeCloud JS Player（`vcode + userId`），失败自动降级原生播放器
- ⚙️ **视频管理** — 启用/禁用/删除/批量删除视频

### 管理功能

- 🔐 **Authelia OAuth2** — 统一登录认证
- 👥 **权限管理** — 管理员/普通用户角色
- 📊 **审计日志** — 所有操作留有日志
- 🧾 **申诉工单** — 封禁资源支持工单复审与处理意见记录
- 💾 **存储配额** — 管理员可管理用户存储配额

### 系统特性

- ⚡ **高性能** — Go 后端，高并发低占用
- 📡 **实时通知** — SSE 事件流推送上传/转码状态
- 🎛️ **流量控制** — 服务端上传并发限制与限速控制
- 🖥️ **CLI 管理** — `ideactl` 命令行工具快捷管理服务

## 技术栈

| 组件     | 技术                    |
| -------- | ----------------------- |
| 后端     | Go (Gin)                |
| 前端     | React 18 + Ant Design 5 |
| 构建     | Vite                    |
| 数据库   | PostgreSQL              |
| 对象存储 | 多吉云 OSS (S3 兼容)    |
| 视频云   | 多吉云 VCloud           |
| 认证     | Authelia OAuth2         |
| 反向代理 | Nginx                   |

## 项目结构

```
IdeaSaver/
├── server/              # Go 后端
│   ├── cmd/             # 程序入口
│   ├── internal/        # 内部包
│   │   ├── config/      # 配置管理
│   │   ├── handler/     # HTTP handlers
│   │   ├── middleware/   # 中间件
│   │   ├── model/       # 数据模型
│   │   ├── repository/  # 数据库操作
│   │   ├── service/     # 业务逻辑
│   │   └── storage/     # 多吉云封装
│   └── internal/repository/migrations/  # 数据库迁移（embed，版本表 schema_migrations）
├── web/                 # React 前端
│   └── src/
│       ├── api/         # API 请求
│       ├── components/  # 通用组件
│       ├── layouts/     # 布局组件
│       ├── pages/       # 页面
│       ├── hooks/       # 自定义 Hooks
│       ├── stores/      # 状态管理
│       └── utils/       # 工具函数
├── cli/                 # ideactl CLI 工具
├── deploy/              # 部署配置
│   ├── nginx/           # Nginx 配置
│   └── systemd/         # systemd 服务
├── .github/             # GitHub 模板与 CI
├── go.mod
├── go.sum
├── README.md
├── README.en.md
├── LICENSE
├── .gitignore
└── Makefile
```

## 快速开始

### 环境要求

- Go 1.23+
- Node.js 20+
- PostgreSQL 15+
- Nginx

### 开发环境

```bash
# 克隆仓库
git clone https://github.com/DTMWiki/IdeaSaver.git
cd IdeaSaver

# 后端
cd server
cp .env.example .env    # 编辑配置
go mod download
go run cmd/main.go

# 前端
cd web
npm install
npm run dev
```

### 构建

```bash
make build          # 构建后端 + 前端
make build-server   # 仅构建后端
make build-web      # 仅构建前端
```

## 部署指南

详见 [deploy/deploy.md](deploy/deploy.md)。

简要步骤：

1. 编译项目或下载 Release
2. 配置 `/etc/ideasaver/.env`
3. 部署 systemd 服务
4. 配置 Nginx 反向代理
5. 执行数据库迁移
6. 启动服务

## 仓库治理与发布规范

- GitHub 分支保护（新界面 Rulesets）：[deploy/github-rulesets.md](deploy/github-rulesets.md)
- 发布规范（分支流、门禁、打标、Hotfix）：[deploy/release-policy.md](deploy/release-policy.md)
- 服务器发布清单：[deploy/release-checklist.md](deploy/release-checklist.md)
- 回滚预案：[deploy/rollback-plan.md](deploy/rollback-plan.md)
- 手动验收脚本：[deploy/manual-acceptance.md](deploy/manual-acceptance.md)

## CLI 工具

```bash
ideactl start              # 启动服务
ideactl stop               # 停止服务
ideactl restart            # 重启服务
ideactl status             # 查看服务状态
ideactl logs               # 查看日志 (跟踪模式)
ideactl logs --lines=100   # 查看最近100行
ideactl config list        # 列出所有配置
ideactl config set KEY VAL # 设置配置
ideactl config get KEY     # 获取配置
ideactl db migrate         # 执行数据库迁移
ideactl db status          # 查看迁移状态
```

## 分支策略

| 分支     | 用途                 |
| -------- | -------------------- |
| `master` | 生产分支（保护分支） |
| `canary` | 预发布/测试分支      |
| `dev`    | 日常开发分支         |

## 贡献与协作

- 默认开发分支为 `dev`，预发布使用 `canary`，生产发布使用 `master`
- 提交前至少运行：
  - `go test ./...`
  - `go build ./...`
  - `cd web && npm run lint && npm run build`
- 涉及部署、鉴权、OSS/VCloud、审计日志的改动，请同步更新 `deploy/` 文档与回滚说明
- 生产发布建议使用源码包 + 服务器构建流程，避免把本地环境产物直接带上服务器

## 贡献者

- 维护团队：DTMWiki
- 欢迎通过 GitHub Issue / Pull Request 参与改进，贡献内容包括功能开发、Bug 修复、文档完善、部署经验补充与安全加固

## 许可证

本项目采用 [MIT License](LICENSE) 开源。

---

<p align="center">Made with ❤️ by <a href="https://dtmwiki.cn">DTMWiki</a></p>
