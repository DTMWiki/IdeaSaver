# IdeaSaver 发布规范（Release Policy）

## 1. 分支与环境映射

- `dev`：开发集成环境
- `canary`：预发布/灰度环境
- `master`：生产环境

固定流向：

- `feature/*` 或 `fix/*` -> `dev`
- `dev` -> `canary`
- `canary` -> `master`

禁止反向直接回灌：

- `master` -> `dev`
- `canary` -> `dev`

如需回灌，必须使用 `cherry-pick` 并写清原因。

## 2. 合并门禁

进入 `canary`、`master` 的 PR 必须满足：

- CI 全绿：
  - `CI / backend (push)`
  - `CI / frontend (push)`
  - `Branch Policy / enforce-flow (pull_request)`
- 至少 1 位审批通过
- 所有 review comment 已解决

## 3. 版本号与标签

使用语义化版本：

- 正式版：`vMAJOR.MINOR.PATCH`（例：`v1.4.2`）
- 候选版：`vMAJOR.MINOR.PATCH-rc.N`（例：`v1.4.2-rc.1`）

规则：

- `canary` 验证通过后，从 `master` 打正式 tag
- 每个 tag 必须对应可回滚的发布包（7z 或二进制）

## 4. 发布流程

### Step 1：开发合入 `dev`

- 功能分支提交 PR 到 `dev`
- 自测 + CI 通过后合并

### Step 2：提升到 `canary`

- 发起 PR：`dev -> canary`
- 跑完整集成测试与手动验收
- 重点验证封禁/申诉/审核日志链路

### Step 3：提升到 `master`

- 发起 PR：`canary -> master`
- 审批并确认发布窗口
- 合并后打 tag，按发布清单上线

## 5. Hotfix 流程

生产紧急修复：

- 从 `master` 切 `hotfix/*`
- PR 合入 `master`，发布 `PATCH` 版本
- 将同一修复按顺序同步：
  - `master` -> `canary`（建议 cherry-pick）
  - `master` -> `dev`（建议 cherry-pick）

## 6. 发布准入与退出标准

发布准入（Go）：

- 冒烟脚本通过：`deploy/scripts/smoke_test.sh`
- 手动验收完成：`deploy/manual-acceptance.md`
- 数据迁移已验证可回滚

发布退出（No-Go）：

- 核心路径失败（登录、上传、下载、封禁、申诉、审核）
- 5xx 异常持续上升
- 数据一致性校验失败

## 7. 审计与留痕

每次发布必须保留：

- 对应 PR 链接
- 发布 commit 与 tag
- 数据迁移记录
- 回滚点（上一个稳定版本）
- 处理人和审批人

建议在 GitHub Release Notes 或 `deploy/release-checklist.md` 中记录以上信息。
