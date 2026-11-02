# 本 Fork 功能开发与维护流程

后续在本仓库增加功能、修改代码、同步上游，一律按本文执行。

本文管的是 **`CuylerChen/sub2api-kiro` 相对 `tamakiramimy/sub2api-kiro` 的独立改动**。  
`AGENTS.md` 其余章节是 tamakiramimy 相对 Wei-Shaw 的历史维护手册；日常功能开发不要按 `AGENTS.md` §4 对 Wei-Shaw 做 `reset --hard`。

---

## 0. 仓库关系（先看清再动手）

```
Wei-Shaw/sub2api                 ← 祖父上游，日常不同步
        ↓
tamakiramimy/sub2api-kiro        ← 直接上游（remote: tamakiramimy，只读）
        ↓
CuylerChen/sub2api-kiro          ← 本仓库（remote: origin，唯一可推送目标）
```

| 项 | 值 |
|---|---|
| 本仓库 `origin` | `git@github.com:CuylerChen/sub2api-kiro.git`（可推送） |
| 直接上游 `tamakiramimy` | `https://github.com/tamakiramimy/sub2api-kiro.git`（push URL 必须是 `no_push`） |
| 主分支 | `main` |
| Go module path | `github.com/Wei-Shaw/sub2api`（不要改） |
| 本地功能清单 | `docs/LOCAL_FEATURES.md` |
| 基线核对日 | 当前 `main` 曾与 `tamakiramimy/main` 同为 `7a1c528c2`（`v0.2.15`） |

---

## 1. 铁律

1. 只向 `origin` 推送。不要 `git push tamakiramimy`，也不要向 Wei-Shaw 推送。
2. 已有 Kiro 及 tamakiramimy 全部本地功能必须保留。不要删除 `backend/internal/kiro/`、`frontend/src/kiro/`，不要为了「干净」丢掉清单里的功能。
3. 新功能业务逻辑放独立目录；改上游/tamakiramimy 已有文件时只加注册、枚举、路由、DI、schema 接线。
4. 在功能分支上改，不要直接在 `main` 上堆大改动。
5. `git reset --hard`、`merge --abort`、force-push 会丢掉已有改动时，必须先向用户确认。
6. 用户当次的明确指令优先于本文和 `AGENTS.md`。文档过时就改文档。
7. 声称「已完成 / 已测试」必须有命令输出或浏览器操作证据。
8. 导入路径保持 `github.com/Wei-Shaw/sub2api`。

---

## 2. 每次开工前（约 1 分钟）

工作区必须干净，并确认上游 remote 仍在。

```bash
git status
git remote -v
git fetch origin
git fetch tamakiramimy
```

期望：

- 工作区干净（没有未提交改动，或先stash/提交）。
- `tamakiramimy` 的 push URL 是 `no_push`。
- 当前在最新 `main`，或明确知道自己在哪个功能分支上。

若 `main` 落后于 `tamakiramimy/main`，先按 **§7 同步上游**，再开新功能。  
若 `main` 已有未合入的本地功能，同步完成后再开新分支。

```bash
git checkout main
git pull origin main
git log --oneline main..tamakiramimy/main    # 上游多出来的提交
git log --oneline tamakiramimy/main..main    # 本 fork 多出来的提交
```

---

## 3. 新增一个功能：逐步执行

### 步骤 1 — 定范围

写清这四项再改代码：

| 项 | 要求 |
|---|---|
| 功能短名 | 英文 slug，如 `foo-bar`，用于分支、目录、commit scope |
| 要解决的问题 | 一句话 |
| 触及面 | 后端 / 前端 / 迁移 / 配置 / 文档 |
| 是否改已有文件 | 列出预计要接线的已有文件 |

同一功能不要混进无关重构。

### 步骤 2 — 开分支

```bash
git checkout main
git pull origin main
git checkout -b feat/<短名>
```

| 类型 | 分支 |
|---|---|
| 新功能 | `feat/<短名>` |
| 修 bug | `fix/<短名>` |
| 文档/流程 | `docs/<短名>` |
| 同步上游 | `sync/tamakiramimy-<YYYYMMDD>` |

### 步骤 3 — 按隔离规则写代码

见 **§4 代码放置规则**。

### 步骤 4 — 补测试

- 后端新逻辑：同包 `*_test.go`，能单测的不要只靠手工。
- 前端新逻辑：`*.spec.ts`，放在功能目录或邻近 `__tests__`。
- 改了 UI：按用户规则做浏览器端到端验证（点击、提交、相关路由、空态/错态）。没有浏览器工具时，用测试或 curl，并写明未验证的部分。
- 改了迁移：不能改写已经发布的 SQL 文件内容；只新增后续迁移。

### 步骤 5 — 更新本地功能清单

打开 `docs/LOCAL_FEATURES.md`，按模板追加一条：

- 短名、说明、分支/commit 标题
- 独立目录
- 改过的已有文件（接线点）
- 同步冲突时怎么处理

没有记入清单的功能，后续同步时容易丢。

### 步骤 6 — 本地验证（§6）

相关门禁通过后再提交。

### 步骤 7 — 提交

Commit 标题可搜索，一个功能一组提交：

```
feat(<短名>): <做什么>
fix(<短名>): <修什么>
docs(<短名>): <文档变更>
```

主体写清：为什么改、改了哪些接线、测试怎么跑。

不要把无关文件（`frontend/pnpm-workspace.yaml`、`backend/internal/web/dist/`、本地配置）加进提交。

### 步骤 8 — 推到自己的 GitHub

```bash
git push -u origin HEAD
```

只推 `origin`。

### 步骤 9 — 合入自己的 `main`

本地功能稳定后：

```bash
git checkout main
git pull origin main
git merge --no-ff feat/<短名>
git push origin main
```

需要线性历史时可用 rebase 后再 merge。不要 force-push `main`，除非用户明确要求，并且已经建了 backup 分支。

---

## 4. 代码放置规则

目标：同步 `tamakiramimy/main` 时，本地功能可以整目录带走，已有文件只需要重新打几行接线。

### 4.1 新功能放独立目录

| 侧 | 目录 |
|---|---|
| 后端业务 | `backend/internal/<短名>/` |
| 前端业务 | `frontend/src/<短名>/` |
| 说明文档 | `docs/` 下本功能自己的 md |

范例：Kiro 用 `backend/internal/kiro/` 与 `frontend/src/kiro/`。新功能照做。

目录内放：核心逻辑、该功能自己的测试、该功能自己的类型/组件/API 封装。

### 4.2 已有文件只加接线

不得不改的框架文件（每个文件尽量只加几行声明）：

- 平台/枚举：`backend/internal/domain/constants.go` 等
- Ent schema：`backend/ent/schema/*.go`
- 路由：`backend/internal/server/routes/`
- DI：`backend/internal/service/wire.go`
- 网关分发：对应 `*_gateway_*.go` 里加 case / 调用
- 前端路由、i18n 模块、管理页入口

接线形态：

```go
// 允许：注册、调用
case domain.PlatformFoo:
    return foopkg.Handle(ctx, req)
```

不要把大段 `foo` 业务写进已有的 `account.go`、`gateway_service.go`、`AccountsView.vue`。

### 4.3 生成代码

改了 Ent schema 或 Wire 后，从**合并后的源文件**重新生成：

```bash
cd backend
go generate ./ent
go generate ./internal/service
```

不要用旧的生成文件覆盖上游/tamakiramimy 的生成结果。

### 4.4 数据库迁移

- 已发布的 `backend/migrations/*.sql` 内容不得改写（checksum 会炸已有实例）。
- 只新增更大序号的迁移文件。
- 若与上游新迁移改同一约束，参考 `docs/UPSTREAM_SYNC_20261008_CN.md` 的 Kiro/TypeSafe 配额并集做法：执行层做并集，保留原文件 checksum。

### 4.5 不要动的默认项

- 不要改 Go module path。
- 不要为了跟功能无关去改 `.github/` 工作流。
- 不要在普通功能提交里改 `backend/cmd/server/VERSION`。
- `tamakiramimy/dev` 是旧历史分支，不要拿它当更新来源。

---

## 5. 修改已有功能（不是全新功能）

1. 仍开 `fix/<短名>` 或 `feat/<短名>` 分支。
2. 优先改独立目录里的代码。
3. 若必须扩大已有文件的接线，同步更新 `docs/LOCAL_FEATURES.md` 的「已有文件接线」列表。
4. 回归该功能相关测试，以及共用同一状态的页面/API。

---

## 6. 验证门禁

按改动范围跑，合入 `main` 前至少覆盖所改侧。全量同步上游时跑完整套。

在仓库根目录执行。后端：

```bash
cd backend
unset OPENAI_API_KEY
export PATH="$(go env GOPATH)/bin:$PATH"
go build ./...
go vet ./...
go test -tags=unit ./...
```

涉及仓库/数据库/Redis 时再加：

```bash
go test -tags=integration ./...
```

`go test ./...` 会跳过 `//go:build unit` 的测试，不能代替上面两条。

前端（用已有 binary，避免 pnpm 版本把 lockfile 改乱）：

```bash
cd frontend
export CI=true
./node_modules/.bin/eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts
./node_modules/.bin/vue-tsc --noEmit
./node_modules/.bin/vitest run
```

改了 UI 时：在浏览器走通主路径、相关路由、空态/错态；布局变化时看桌面和移动宽度。

构建若生成 `frontend/pnpm-workspace.yaml` 或改动了 `frontend/pnpm-lock.yaml` 且并非有意升级依赖：

```bash
git checkout frontend/pnpm-lock.yaml
rm -f frontend/pnpm-workspace.yaml
```

不要提交 `backend/internal/web/dist/`。

---

## 7. 同步 `tamakiramimy`（更新上游）

直接上游是 **tamakiramimy**，日常不要同步 Wei-Shaw。

### 7.1 开始前

```bash
git status                    # 必须干净
git fetch tamakiramimy
git checkout main
git pull origin main
git log --oneline main..tamakiramimy/main
```

无新提交则结束。有新提交则先备份：

```bash
git branch backup/pre-sync-$(date +%Y%m%d-%H%M%S) HEAD
```

打开 `docs/LOCAL_FEATURES.md`，列出本次必须保住的本地功能。

### 7.2 本地还没有独立功能时

```bash
git merge tamakiramimy/main
git push origin main
```

### 7.3 本地已有独立功能，且差异不大

仍可在 `main` 上 `git merge tamakiramimy/main`。

冲突处理：

1. 独立目录：保留本 fork 版本。
2. 已有文件冲突：先取 `tamakiramimy` 的实现，再把 `LOCAL_FEATURES.md` 里的接线打回去。
3. 不要用本 fork 的整文件覆盖 tamakiramimy 已重构的大文件。
4. Ent/Wire 从合并后源码重新生成。

### 7.4 差异大或 merge 冲突失控时

在隔离 worktree 里以 `tamakiramimy/main` 为底，重放本 fork 相对它的差异：

```bash
git fetch tamakiramimy
git worktree add /tmp/sub2api-kiro-sync tamakiramimy/main
cd /tmp/sub2api-kiro-sync
git checkout -b sync/tamakiramimy-$(date +%Y%m%d)

BASE=$(git merge-base main tamakiramimy/main)
git diff $BASE main > /tmp/local-fork.patch
git apply --3way /tmp/local-fork.patch
```

按 §7.3 解决冲突，跑 §6 全量门禁。通过后把结果合回本仓库 `main`，普通 `git push origin main`。

不要在未确认的情况下 `reset --hard tamakiramimy/main`。

### 7.5 同步后

- 更新 `docs/LOCAL_FEATURES.md` 的「最后一次同步」记录。
- 抽查清单里每个功能的接线还在。
- 需要发布再改 `backend/cmd/server/VERSION`。

---

## 8. 明确不要做的事

- 向 `tamakiramimy` 或 Wei-Shaw 开 PR、push（除非用户明确要求回馈上游）。
- 用 `git merge wei-shaw/main` 当日常同步。
- 按 `AGENTS.md` §4 对 Wei-Shaw 做 full-rewrite，除非用户明确要求跟 Wei-Shaw，并且已备份、已确认本地功能清单。
- 把本地业务写进上游大文件后靠注释「标记是 fork 改的」。
- 改写已发布迁移 SQL。
- 未备份就 force-push `origin/main`。
- 把 `tamakiramimy/dev` 当作新代码来源。

---

## 9. 单次功能开发检查表

复制到任务里逐项勾：

```
[ ] git status 干净；已 fetch origin 与 tamakiramimy
[ ] main 不落后于 tamakiramimy/main（或已按 §7 同步）
[ ] 已开 feat/<短名> 或 fix/<短名> 分支
[ ] 业务代码在 backend/internal/<短名>/ 和或 frontend/src/<短名>/
[ ] 已有文件只有注册/接线，无大段新业务
[ ] 测试已补；UI 已按规则验证
[ ] docs/LOCAL_FEATURES.md 已追加或更新
[ ] 相关门禁已跑过，有输出证据
[ ] commit 标题为 feat|fix(<短名>): ...
[ ] 只 push 到 origin
```

---

## 10. 给后续执行者（含 Agent）

1. 先读本文和 `docs/LOCAL_FEATURES.md`，再改代码。
2. 一次任务只做用户指定的功能，不顺手重构、不顺手同步上游（用户没点名同步就不要同步）。
3. 破坏性 git 操作先停，问用户。
4. 做完后在回复里写：改了什么、清单是否已更新、跑了哪些验证、还有什么没验证。
