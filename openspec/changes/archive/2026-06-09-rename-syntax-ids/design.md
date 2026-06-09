## Context

`go-macro-contrib` 中三个 Call 宏包（`inline`、`try`、`with`）当前在 `//macro:` 指令与注册表中使用带 `syntax-` 前缀的 syntax-id。同仓库的 `derive`（`derive`）与 `wirejson`（`wire-json`）已采用与包名对齐的短标识。本次变更仅重命名 syntax-id，不触及 Expander 逻辑、桩签名或展开语义。

受影响位置：
- 各包 `stubs.go`、`expand.go` 上的 `//macro:` 注释
- `expand_test.go` 中 `mactest.ExpandCall` 的第三个参数（syntax-id）
- `README.md` 官方宏库表格
- `openspec/specs/` 中相关规格文本
- `.cursor/skills/openspec-viewer/generate_viewer.py` 依赖图节点键名

## Goals / Non-Goals

**Goals:**

- 将 `syntax-inline`、`syntax-try`、`syntax-with` 统一更名为 `inline`、`try`、`with`
- 保持与 `derive` / `wire-json` 命名风格一致：syntax-id 与包目录名对应
- 全仓库内旧 id 零残留（代码、测试、文档、规格）

**Non-Goals:**

- 不重命名 OpenSpec 规格目录名（`syntax-inline/` 等保持不变，仅更新内容中的 id 引用）
- 不修改 Expander 实现、桩 API、展开语义
- 不在 `go-macro` 核心仓库做变更
- 不提供运行时兼容层或双注册（旧 id 与新 id 并存）

## Decisions

### 1. 一次性硬切换，无兼容别名

**选择**：直接替换旧 syntax-id，不保留 `syntax-*` 作为别名。

**理由**：contrib 尚未广泛对外发布旧 id；双注册增加维护成本且易混淆。与 `derive` 从未使用 `syntax-derive` 前缀的做法一致。

**备选**：同时注册新旧 id → 拒绝，增加测试与文档负担。

### 2. 机械替换范围

**选择**：全局搜索 `syntax-inline`、`syntax-try`、`syntax-with` 并替换为对应新 id；不改动 `syntax-derive`、`syntax-wire-json` 等规格目录名或 `derive`/`wire-json` syntax-id。

**理由**：用户明确要求仅更名这三个 id；规格目录名属于 OpenSpec 组织方式，与 runtime syntax-id 解耦（`derive` 已是先例）。

### 3. 规格以 MODIFIED delta 更新

**选择**：在 change 的 `specs/` 下为四个 capability 提交 MODIFIED Requirements，归档时合并入主规格。

**理由**：syntax-id 属于规格级可观测标识，必须在 spec 中体现 BREAKING 变更。

## Risks / Trade-offs

- **[BREAKING 外部用户]** 已手写 `//macro: syntax-try` 的宏主文件将停止被 expand 发现 → 在 README 与变更说明中标注迁移对照表
- **[遗漏引用]** 归档 change 或 viewer 脚本中可能残留旧 id → 任务包含 `rg 'syntax-(inline|try|with)'` 零匹配验收
- **[mactest 参数]** 测试第三个参数必须与桩注释一致 → 每个包 `go test` 作为门禁

## Migration Plan

1. 合并并发布 `go-macro-contrib` 新版本
2. 用户将宏主文件及相关文档中的 `//macro: syntax-*` 改为 `//macro: inline|try|with`
3. 无需 `go-macro` 核心升级（expand 按 provider 注释发现 syntax-id）
4. 回滚：revert 本 change 即可恢复旧 id

## Open Questions

（无）
