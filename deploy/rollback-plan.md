# IdeaSaver 回滚预案

## 触发条件（任一满足即执行回滚）

- 服务启动失败或持续 CrashLoop。
- 登录、上传、文件预览核心链路异常。
- 新增封禁/申诉流程导致大面积 5xx。
- 数据迁移后出现严重兼容性问题。

## 回滚步骤

## 1. 立刻止损

- `sudo systemctl stop ideasaver`
- 冻结新版本发布动作，通知相关人员进入回滚窗口。

## 2. 回滚应用

- 恢复上一个稳定版二进制：
  - `cp /opt/ideasaver/releases/<previous>/ideasaver /opt/ideasaver/ideasaver`
- 恢复上一个稳定版前端静态资源目录。

## 3. 数据处理策略

- 本次新增迁移为新增字段/表，通常可向后兼容旧版本。
- 若发生数据层不兼容：
  - 使用发布前备份执行数据库恢复（`pg_restore`）。
  - 恢复前先确认业务可接受的数据回退窗口。

## 4. 启动并验证旧版本

- `sudo systemctl start ideasaver`
- `sudo systemctl status ideasaver --no-pager`
- 执行：
  - `bash deploy/scripts/smoke_test.sh`

## 5. 事后复盘

- 记录故障时间线、影响范围、根因。
- 形成修复 PR 与二次发布条件（测试补充、灰度策略、监控补齐）。
