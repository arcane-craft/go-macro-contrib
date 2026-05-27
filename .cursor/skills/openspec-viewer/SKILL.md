---
name: openspec-viewer
description: "Generate a dark-themed HTML viewer for OpenSpec spec files — collapsible cards, RFC 2119 keyword highlighting, dependency graph, scenarios with Given/When/Then."
triggers:
  - "openspec viewer"
  - "spec可视化"
  - "spec to html"
  - "openspec html"
  - "spec viewer"
---

# OpenSpec Spec → HTML Viewer

将 OpenSpec 的 `.md` spec 文件转为可交互的单文件 HTML 页面，提升人类阅读体验。

## 输入

一份 OpenSpec spec Markdown 文件，结构为：

```markdown
# <name> Specification
## Purpose
## Requirements
### Requirement: <title>
<描述，含 RFC 2119 关键词>
#### Scenario: <场景名>
- GIVEN ...
- WHEN ...
- THEN ...
```

## 输出

单个 `index.html`，无外部依赖，暗色主题，可直接浏览器打开。

## 页面结构（6 个区块）

### 1. Header（粘性顶栏）
- spec 标题（从 `# xxx Specification` 提取）
- Purpose 一句话摘要
- 统计条：Requirements 数 / Scenarios 数 / 关键词标签

### 2. Summary Bar（统计摘要）
- 4 个卡片：Requirements 总数、Scenarios 总数、MUST 约束数、SHOULD 数
- 让读者 2 秒内判断 spec 的约束密度

### 3. Legend（关键词图例）
- MUST = 红色 / MUST NOT = 深红 / SHOULD = 黄色 / MAY = 绿色
- 固定在 Summary 下方，全局可见

### 4. Sidebar（左侧导航，桌面端 sticky）
- 列出所有 Requirement 标题，锚链接跳转
- 底部放 Quick Ref（图例、依赖图）

### 5. Dependency Graph（需求依赖关系）
- 纯 SVG 手绘风格，节点 = 圆角矩形 + 箭头
- 从 spec 文字中**人工推断**依赖关系后绘制
- 无依赖则省略此区块

### 6. Requirement Cards（核心内容区）
每个 Requirement 是一张**可折叠卡片**：

```
┌─ [序号圆圈] 标题 ── [MUST标签] [N scenarios标签] ─ [▸折叠箭头] ─┐
│                                                                   │
│  ┌─ 描述区（深色底，RFC关键词高亮）──────────────────────────┐    │
│  │ 原文描述，MUST/MUST NOT/SHOULD/MAY 用对应颜色 <span>    │    │
│  └───────────────────────────────────────────────────────────┘    │
│                                                                   │
│  ┌─ 表格（如有）─────────────────────────────────────────────┐    │
│  │ 桩名 | 签名 | 载荷数  （按原spec中的表格还原）           │    │
│  └───────────────────────────────────────────────────────────┘    │
│                                                                   │
│  ┌─ Scenario 1 ──────────────────────────────────────────────┐    │
│  │ ▶ 场景标题                                                │    │
│  │ [When] 触发条件                                           │    │
│  │ [Then] 期望行为（含 MUST 等高亮）                         │    │
│  └───────────────────────────────────────────────────────────┘    │
│  ┌─ Scenario 2 ──────────────────────────────────────────────┐    │
│  │ ...                                                       │    │
│  └───────────────────────────────────────────────────────────┘    │
└───────────────────────────────────────────────────────────────────┘
```

## 视觉设计规范

### 色板（CSS 变量）
```css
--bg: #0d1117           /* 页面背景 */
--surface: #161b22      /* 卡片背景 */
--surface2: #1c2129     /* 描述区/表头背景 */
--border: #30363d       /* 边框 */
--text: #e6edf3         /* 正文 */
--text-muted: #8b949e   /* 次要文字 */
--accent: #58a6ff       /* 强调色（蓝） */

/* RFC 2119 关键词 */
--must: #f85149         /* MUST — 红 */
--mustnot: #da3633      /* MUST NOT — 深红 */
--should: #d29922       /* SHOULD — 黄 */
--may: #3fb950          /* MAY — 绿 */

/* Scenario 步骤标签 */
--when: #58a6ff         /* When — 蓝 */
--then: #3fb950         /* Then — 绿 */
--given: #d29922        /* Given — 黄 */
```

### 字体
```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans SC', sans-serif;
```

### 代码高亮
```css
code {
  background: rgba(110,118,129,0.15);
  color: #ffa657;       /* 橙色 */
  border-radius: 3px;
  padding: 0.15em 0.4em;
}
```

### 关键词标签样式
- 小圆角 pill（4px radius）
- 半透明背景 + 实色文字 + 细边框
- 字号 0.7rem，大写，加粗

### Scenario 步骤标签
- GIVEN/WHEN/THEN 各用对应颜色的半透明背景 pill
- 左侧对齐，文字紧随其后

## 交互行为

1. **折叠/展开**：点击 `req-header` 切换父级 `.req-card` 的 `.open` class，`req-body` 通过 `display:none/block` 控制
2. **第一个卡片默认展开**，其余默认折叠
3. **侧边栏锚链接**：点击跳转到对应卡片（CSS scroll-behavior: smooth）
4. **hover 效果**：卡片边框变蓝、侧边栏左侧出现蓝色竖线

**折叠 JS 约束（重要）**：整页只绑定**一种**点击处理——使用 `templates/viewer-template.html` 底部的 `addEventListener`，**禁止**在 `req-header` 上再加 `onclick`。二者并存会导致一次点击 `toggle` 两次、折叠失效。

## 生成步骤

1. **解析 spec markdown**：
   - 提取标题、Purpose
   - 遍历每个 `### Requirement:` → 提取标题、描述文本、关键词
   - 遍历每个 `#### Scenario:` → 提取场景名、GIVEN/WHEN/THEN 步骤
   - 统计 MUST/MUST NOT/SHOULD/MAY 出现次数

2. **推断依赖关系**：
   - 分析描述文本中的交叉引用（如"外层函数"暗示依赖"桩族定义"）
   - 绘制 SVG 依赖图（无明显依赖则省略）

3. **生成 HTML**：
   - 以 `templates/viewer-template.html` 为骨架（CSS、布局、底部折叠脚本）
   - 写入 Header + Summary + Legend + Sidebar
   - 写入 Dependency Graph SVG（如有）
   - 遍历每个 Requirement 生成卡片 HTML（`req-header` **无** `onclick`）
   - 关键词替换：`MUST` → `<span class="kw-must">MUST</span>` 等；`MUST NOT` 须先于 `MUST` 替换，避免嵌套错误
   - Scenario 步骤用 Given/When/Then 色块标签包裹

4. **写入文件**：`${用户指定路径}/index.html`

## 注意事项

- **单文件**：所有 CSS 内联，不依赖外部资源
- **纯静态**：不需要 JS 框架，vanilla JS 处理折叠即可（仅底部一处 `addEventListener`，勿重复绑定）
- **中英混排**：spec 内容保持原样，不翻译
- **表格还原**：spec 中如有 markdown 表格，转为 `<table>` 保持结构
- **SVG 依赖图**：手写 SVG，节点位置硬编码，不用 JS 动态计算
- **移动端适配**：800px 以下隐藏 sidebar，单列布局
