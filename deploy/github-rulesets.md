# IdeaSaver GitHub Rulesets 配置指南

> 适用于你当前仓库界面找不到 `Branch protection rules` 的情况。

## 为什么找不到旧入口

GitHub 新版仓库设置通常把旧的分支保护入口合并到了：

- `Settings` -> `Rules` -> `Rulesets`

如果你没看到 `Branch protection rules`，一般就是用了新界面，不是你操作错了。

## 建议的分支规则（生产可用）

分支职责：

- `master`：生产分支（只接收 `canary` 提升）
- `canary`：预发布分支（只接收 `dev` 提升）
- `dev`：开发集成分支

仓库内已提供 CI 约束工作流：`.github/workflows/branch-policy.yml`，会在 PR 时校验：

- 仅允许 `dev -> canary`
- 仅允许 `canary -> master`
- 阻止 `canary/master -> dev`

## 具体配置步骤

### 1. 进入规则页

- 打开仓库 -> `Settings` -> `Rules` -> `Rulesets`
- 点击 `New ruleset` -> `New branch ruleset`

### 2. 创建 `master` 规则

- Ruleset name：`protect-master`
- Enforcement status：`Active`
- Target branches：`master`
- 开启：
  - `Restrict deletions`
  - `Block force pushes`
  - `Require a pull request before merging`
  - `Require approvals`（建议至少 1）
  - `Require conversation resolution before merging`
  - `Require status checks to pass`
- Required status checks 添加：
  - `CI / backend (push)`
  - `CI / frontend (push)`
  - `Branch Policy / enforce-flow (pull_request)`

### 3. 创建 `canary` 规则

- Ruleset name：`protect-canary`
- Target branches：`canary`
- 开启：
  - `Restrict deletions`
  - `Block force pushes`
  - `Require a pull request before merging`
  - `Require status checks to pass`
- Required status checks 添加：
  - `CI / backend (push)`
  - `CI / frontend (push)`
  - `Branch Policy / enforce-flow (pull_request)`

### 4. 创建 `dev` 规则（轻保护）

- Ruleset name：`protect-dev`
- Target branches：`dev`
- 开启：
  - `Restrict deletions`
  - `Block force pushes`

如果团队希望 `dev` 也全部走 PR，可额外开启 `Require a pull request before merging`。

### 5. 验证规则生效

- 发起 `dev -> canary` PR：应允许
- 发起 `canary -> master` PR：应允许
- 发起 `master -> dev` 或 `canary -> dev` PR：应被 `Branch Policy` 拦截

## 常见问题

### Q1：公开仓库能否只让 `master` 公开，`dev/canary` 私有？

不能。GitHub 的公开仓库是“仓库级公开”，不是“分支级公开”。  
如果要隔离内部分支，建议：

- 方案 A：维护一个私有开发仓库 + 一个公开镜像仓库（仅同步 `master`）
- 方案 B：保持公开仓库，但通过 Rulesets + 最小权限控制写入
