# 本 Fork 相对 tamakiramimy 的本地功能清单

每增加一个本仓库独有功能，按文末模板追加一条。  
同步 `tamakiramimy/main` 时，以本清单为准保留功能。流程见 `docs/FORK_FEATURE_WORKFLOW_CN.md`。

## 同步记录

| 日期 | 本仓库 HEAD | tamakiramimy/main | 说明 |
|---|---|---|---|
| 2026-11-03 | `7a1c528c2` | `7a1c528c2`（`v0.2.15`） | 初次核对，两边相同；尚无本 fork 独有功能 |
| 2026-11-03 | `feat/kiro-bonuses` | `7a1c528c2` | 增加 Kiro `bonuses[]` 解析与账号用量展示 |

## 必须随上游一起保留的已有能力

这些已经在 `tamakiramimy/main` 里，不是本 fork 新增，但同步或改代码时不得删除：

- Kiro 平台：`backend/internal/kiro/`、`frontend/src/kiro/`，以及账号/分组/网关接线
- OpenAI OAuth refresh_token_reused 软处理与管理员 reauth 提示
- 全平台 stale-stream watchdog
- Harmony `<|channel|>` 中和与 `invalid_prompt` 可观测性
- 2026-10-08 同步保留项：见 `docs/UPSTREAM_SYNC_20261008_CN.md`（含 Image 2.5、GPT-6 prompt-cache、Responses 数组 tool output、Claude 5.5 相关优化）

## 本 Fork 独有功能

### `kiro-bonuses`

- 状态：进行中
- 说明：Kiro `getUsageLimits` 的 `bonuses[]` 解析到 `kiro_bonuses`，管理后台账号用量格分条展示赠额名称、用量和剩余天数。`freeTrialInfo` 仍走原有 `kiro_bonus`。
- 分支 / commit 标题：`feat(kiro-bonuses): show Kiro usage bonuses in account quota`
- 独立目录：无（扩展已有 Kiro 用量路径）
- 已有文件接线：
  - `backend/internal/service/kiro_usage_fetcher.go` — 解析 CREDIT breakdown 或响应顶层 `bonuses`，额度耗尽判断计入赠额剩余
  - `backend/internal/service/account_usage_service.go` — `UsageInfo.KiroBonuses`、`KiroCreditProgress.DisplayName`
  - `frontend/src/components/account/AccountUsageCell.vue` — 分条展示 `kiro_bonuses`
  - `frontend/src/types/index.ts` — `kiro_bonuses` 类型
- 测试：`kiro_alignment_test.go` 映射/耗尽；`AccountUsageCell.spec.ts` 展示 Welcome Bonus
- 同步冲突处理：保留本 fork 的 `bonuses` 解析与 `kiro_bonuses` 展示；已有文件先取 tamakiramimy 再打回上述接线

<!--
### `<短名>`

- 状态：进行中 / 已合入 main
- 说明：
- 分支 / commit 标题：`feat(<短名>): ...`
- 独立目录：
  - `backend/internal/<短名>/`
  - `frontend/src/<短名>/`
- 已有文件接线：
  - `path/to/file` — 加了什么（枚举 / 路由 / DI / schema 字段）
- 测试：
- 同步冲突处理：独立目录保留本 fork；已有文件先取 tamakiramimy 再打回本条接线
-->
