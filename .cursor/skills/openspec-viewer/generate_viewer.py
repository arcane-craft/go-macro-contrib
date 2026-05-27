#!/usr/bin/env python3
"""Generate OpenSpec HTML viewers from spec markdown files."""

from __future__ import annotations

import html
import re
import sys
from pathlib import Path

TEMPLATE_CSS_START = """<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>"""

TEMPLATE_HEAD_END = """</title>
<style>
  :root {
    --bg: #0d1117; --surface: #161b22; --surface2: #1c2129; --border: #30363d;
    --text: #e6edf3; --text-muted: #8b949e; --accent: #58a6ff;
    --must: #f85149; --must-bg: rgba(248,81,73,0.1);
    --mustnot: #da3633; --mustnot-bg: rgba(218,54,51,0.1);
    --should: #d29922; --should-bg: rgba(210,153,34,0.1);
    --may: #3fb950; --may-bg: rgba(63,185,80,0.1);
    --scenario-bg: rgba(31,111,235,0.06); --tag-radius: 4px;
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans SC', sans-serif; background: var(--bg); color: var(--text); line-height: 1.7; }
  .header { background: linear-gradient(135deg, #0d1117 0%, #161b22 100%); border-bottom: 1px solid var(--border); padding: 2rem 2rem 1.5rem; position: sticky; top: 0; z-index: 100; backdrop-filter: blur(12px); }
  .header h1 { font-size: 1.6rem; font-weight: 700; color: var(--accent); margin-bottom: 0.25rem; }
  .header .purpose { color: var(--text-muted); font-size: 0.95rem; }
  .header .meta { display: flex; gap: 1.5rem; margin-top: 0.75rem; font-size: 0.85rem; color: var(--text-muted); flex-wrap: wrap; }
  .header .meta a { color: var(--accent); text-decoration: none; }
  .header .meta a:hover { text-decoration: underline; }
  .container { max-width: 1100px; margin: 0 auto; padding: 1.5rem 2rem 4rem; }
  .layout { display: grid; grid-template-columns: 240px 1fr; gap: 2rem; align-items: start; }
  .sidebar { position: sticky; top: 120px; max-height: calc(100vh - 140px); overflow-y: auto; }
  .sidebar h3 { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-muted); margin-bottom: 0.75rem; }
  .sidebar a { display: block; padding: 0.35rem 0.75rem; color: var(--text-muted); text-decoration: none; font-size: 0.85rem; border-left: 2px solid transparent; border-radius: 0 var(--tag-radius) var(--tag-radius) 0; transition: all 0.15s; }
  .sidebar a:hover { color: var(--text); background: var(--surface); border-left-color: var(--accent); }
  .dep-graph { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1.25rem 1.5rem; margin-bottom: 2rem; }
  .dep-graph h3 { font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--text-muted); margin-bottom: 1rem; }
  .dep-graph svg { width: 100%; height: auto; }
  .req-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; margin-bottom: 1.25rem; overflow: hidden; transition: border-color 0.2s; scroll-margin-top: 130px; }
  .req-card:hover { border-color: var(--accent); }
  .req-card.open { border-color: var(--accent); }
  .req-header { display: flex; align-items: center; gap: 0.75rem; padding: 1rem 1.25rem; cursor: pointer; user-select: none; }
  .req-header:hover { background: var(--surface2); }
  .req-num { width: 28px; height: 28px; border-radius: 50%; background: var(--accent); color: #000; font-size: 0.8rem; font-weight: 700; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
  .req-title { font-size: 1rem; font-weight: 600; flex: 1; }
  .req-tags { display: flex; gap: 0.4rem; flex-shrink: 0; flex-wrap: wrap; justify-content: flex-end; }
  .req-toggle { color: var(--text-muted); font-size: 1.2rem; transition: transform 0.2s; flex-shrink: 0; }
  .req-card.open .req-toggle { transform: rotate(90deg); }
  .req-body { display: none; padding: 0 1.25rem 1.25rem; border-top: 1px solid var(--border); }
  .req-card.open .req-body { display: block; }
  .req-desc { font-size: 0.92rem; line-height: 1.8; color: var(--text); margin-bottom: 1rem; padding: 0.75rem 1rem; background: var(--surface2); border-radius: 6px; }
  .req-desc p { margin-bottom: 0.5rem; }
  .req-desc p:last-child { margin-bottom: 0; }
  .req-desc ul { margin: 0.5rem 0 0.5rem 1.25rem; }
  .req-desc .kw-must { color: var(--must); font-weight: 700; }
  .req-desc .kw-mustnot { color: var(--mustnot); font-weight: 700; }
  .req-desc .kw-should { color: var(--should); font-weight: 700; }
  .req-desc .kw-may { color: var(--may); font-weight: 700; }
  .req-desc .kw-rec { color: var(--should); font-weight: 700; }
  .req-desc code, .scenario-steps code { background: rgba(110,118,129,0.15); padding: 0.15em 0.4em; border-radius: 3px; font-size: 0.88em; color: #ffa657; }
  .req-desc pre { background: rgba(110,118,129,0.1); padding: 0.75rem 1rem; border-radius: 6px; overflow-x: auto; margin: 0.5rem 0; font-size: 0.85rem; line-height: 1.5; }
  .req-desc pre code { background: none; padding: 0; color: var(--text); }
  .req-table { width: 100%; border-collapse: collapse; margin-bottom: 1rem; font-size: 0.88rem; }
  .req-table th { text-align: left; padding: 0.5rem 0.75rem; background: var(--surface2); color: var(--text-muted); font-weight: 600; font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.04em; border-bottom: 1px solid var(--border); }
  .req-table td { padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--border); vertical-align: top; }
  .req-table tr:last-child td { border-bottom: none; }
  .scenario { background: var(--scenario-bg); border: 1px solid rgba(31,111,235,0.15); border-radius: 6px; padding: 0.85rem 1rem; margin-bottom: 0.75rem; }
  .scenario-title { font-size: 0.82rem; font-weight: 600; color: var(--accent); margin-bottom: 0.5rem; display: flex; align-items: center; gap: 0.4rem; }
  .scenario-title::before { content: '▶'; font-size: 0.65rem; }
  .scenario-steps { list-style: none; font-size: 0.88rem; }
  .scenario-steps li { padding: 0.2rem 0; display: flex; gap: 0.5rem; }
  .step-label { font-weight: 700; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.1rem 0.45rem; border-radius: 3px; flex-shrink: 0; margin-top: 0.15rem; }
  .step-when .step-label { background: rgba(88,166,255,0.15); color: var(--accent); }
  .step-then .step-label { background: rgba(63,185,80,0.15); color: var(--may); }
  .step-given .step-label { background: rgba(210,153,34,0.15); color: var(--should); }
  .tag { display: inline-block; padding: 0.15rem 0.5rem; border-radius: var(--tag-radius); font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; }
  .tag-must { background: var(--must-bg); color: var(--must); border: 1px solid rgba(248,81,73,0.25); }
  .tag-mustnot { background: var(--mustnot-bg); color: var(--mustnot); border: 1px solid rgba(218,54,51,0.25); }
  .tag-should { background: var(--should-bg); color: var(--should); border: 1px solid rgba(210,153,34,0.25); }
  .tag-may { background: var(--may-bg); color: var(--may); border: 1px solid rgba(63,185,80,0.25); }
  .tag-count { background: rgba(88,166,255,0.1); color: var(--accent); border: 1px solid rgba(88,166,255,0.2); }
  .summary-bar { display: flex; gap: 1rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
  .summary-item { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 0.75rem 1.25rem; display: flex; align-items: center; gap: 0.75rem; font-size: 0.85rem; }
  .summary-item .num { font-size: 1.5rem; font-weight: 700; color: var(--accent); }
  .legend { display: flex; gap: 1rem; margin-bottom: 1.5rem; flex-wrap: wrap; font-size: 0.8rem; color: var(--text-muted); }
  .legend-item { display: flex; align-items: center; gap: 0.35rem; }
  @media (max-width: 800px) { .layout { grid-template-columns: 1fr; } .sidebar { display: none; } .container { padding: 1rem; } }
  .hub-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; margin-top: 1.5rem; }
  .hub-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1.25rem; text-decoration: none; color: inherit; transition: border-color 0.2s, transform 0.15s; display: block; }
  .hub-card:hover { border-color: var(--accent); transform: translateY(-2px); }
  .hub-card h2 { font-size: 1.1rem; color: var(--accent); margin-bottom: 0.5rem; }
  .hub-card p { font-size: 0.85rem; color: var(--text-muted); margin-bottom: 0.75rem; }
  .hub-stats { display: flex; gap: 0.75rem; font-size: 0.8rem; color: var(--text-muted); flex-wrap: wrap; }
</style>
</head>
<body>
"""

SCRIPT_FOOT = """
<script>
(function () {
  function bindToggle() {
    document.querySelectorAll('.req-card .req-header').forEach(function (header) {
      header.addEventListener('click', function (e) {
        e.preventDefault();
        var card = header.closest('.req-card');
        if (card) card.classList.toggle('open');
      });
    });
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bindToggle);
  } else {
    bindToggle();
  }
})();
</script>
</body>
</html>
"""

# Dependency hints per spec (manual inference per skill)
DEP_GRAPHS: dict[str, list[tuple[str, str]]] = {
    "macro-core": [
        ("宏上下文 API", "ExpandResult"),
        ("ExpandResult", "展开器分发"),
        ("通用宏注册", "expandtool"),
    ],
    "macro-expander": [
        ("识别与校验分离", "go/types 识别"),
        ("go/types 识别", "ExpandResult 贴回"),
        ("Provider 激活", "通用展开器分发"),
    ],
    "macro-codegen": [
        ("go generate", "方案 C 写回"),
        ("build tag", "constraint 推导"),
        ("expandtool", "仅展开当前模块"),
    ],
    "macro-contrib": [("独立子 module", "register 注册")],
    "macro-repo-layout": [("多 module 布局", "expand 入口"), ("examples", "readfile 示例")],
    "syntax-inline": [("Inline 桩", "InlineExpand"), ("InlineExpand", "框架边界")],
    "syntax-try": [
        ("Try 桩族", "TryExpand"),
        ("error 位置规则", "Stmts 展开"),
        ("TryExpand", "readfile 示例"),
    ],
}


def slugify(title: str) -> str:
    s = re.sub(r"[^\w\u4e00-\u9fff-]+", "-", title.strip())
    return s.strip("-")[:40] or "req"


def count_keywords(text: str) -> dict[str, int]:
    return {
        "must": len(re.findall(r"\bMUST\b", text)),
        "must_not": len(re.findall(r"\bMUST NOT\b", text)),
        "should": len(re.findall(r"\bSHOULD\b", text)),
        "may": len(re.findall(r"\bMAY\b", text)),
        "recommended": len(re.findall(r"\bRECOMMENDED\b", text)),
    }


def highlight_keywords(text: str) -> str:
    text = html.escape(text)
    text = re.sub(
        r"`([^`]+)`",
        r'<code>\1</code>',
        text,
    )
    replacements = [
        ("MUST NOT", '<span class="kw-mustnot">MUST NOT</span>'),
        ("RECOMMENDED", '<span class="kw-rec">RECOMMENDED</span>'),
        ("MUST", '<span class="kw-must">MUST</span>'),
        ("SHOULD", '<span class="kw-should">SHOULD</span>'),
        ("MAY", '<span class="kw-may">MAY</span>'),
    ]
    for old, new in replacements:
        text = text.replace(old, new)
    return text


def parse_table(lines: list[str]) -> tuple[str, int]:
    rows: list[list[str]] = []
    i = 0
    while i < len(lines) and "|" in lines[i]:
        row = [c.strip() for c in lines[i].strip().strip("|").split("|")]
        if not all(set(c) <= {"-", ":", " "} for c in row):
            rows.append(row)
        i += 1
    if len(rows) < 2:
        return "", i
    header, body = rows[0], rows[1:]
    out = ['<table class="req-table"><tr>']
    for h in header:
        out.append(f"<th>{highlight_keywords(h)}</th>")
    out.append("</tr>")
    for row in body:
        out.append("<tr>")
        for cell in row:
            out.append(f"<td>{highlight_keywords(cell)}</td>")
        out.append("</tr>")
    out.append("</table>")
    return "".join(out), i


def format_desc_block(text: str) -> str:
    parts: list[str] = []
    lines = text.split("\n")
    i = 0
    buf: list[str] = []

    def flush_paragraph():
        nonlocal buf
        if buf:
            joined = "\n".join(buf).strip()
            if joined:
                parts.append(f"<p>{highlight_keywords(joined)}</p>")
            buf = []

    while i < len(lines):
        line = lines[i]
        if line.strip().startswith("```"):
            flush_paragraph()
            lang = line.strip()[3:].strip()
            i += 1
            code_lines: list[str] = []
            while i < len(lines) and not lines[i].strip().startswith("```"):
                code_lines.append(lines[i])
                i += 1
            code = html.escape("\n".join(code_lines))
            parts.append(f'<pre><code class="lang-{html.escape(lang)}">{code}</code></pre>')
            if i < len(lines):
                i += 1
            continue
        if "|" in line and line.strip().startswith("|"):
            flush_paragraph()
            table_lines = []
            while i < len(lines) and "|" in lines[i]:
                table_lines.append(lines[i])
                i += 1
            tbl, _ = parse_table(table_lines)
            if tbl:
                parts.append(tbl)
            continue
        if re.match(r"^[-*]\s+", line.strip()):
            flush_paragraph()
            items: list[str] = []
            while i < len(lines) and re.match(r"^[-*]\s+", lines[i].strip()):
                items.append(re.sub(r"^[-*]\s+", "", lines[i].strip()))
                i += 1
            parts.append("<ul>" + "".join(f"<li>{highlight_keywords(it)}</li>" for it in items) + "</ul>")
            continue
        if not line.strip():
            flush_paragraph()
            i += 1
            continue
        buf.append(line)
        i += 1
    flush_paragraph()
    return "\n".join(parts) if parts else f"<p>{highlight_keywords(text.strip())}</p>"


def dominant_tag(text: str) -> str:
    if "MUST NOT" in text:
        return "mustnot"
    if re.search(r"\bMUST\b", text):
        return "must"
    if re.search(r"\bSHOULD\b", text):
        return "should"
    if re.search(r"\bMAY\b", text):
        return "may"
    return "must"


def parse_scenario_step(line: str) -> tuple[str, str, str]:
    m = re.match(r"^[-*]\s+\*\*(GIVEN|WHEN|THEN)\*\*\s*(.*)$", line.strip(), re.I)
    if m:
        kind = m.group(1).upper()
        body = m.group(2).strip()
        cls = {"GIVEN": "given", "WHEN": "when", "THEN": "then"}[kind]
        return cls, kind.capitalize(), body
    m2 = re.match(r"^[-*]\s+(.*)$", line.strip())
    if m2:
        return "when", "When", m2.group(1).strip()
    return "when", "When", line.strip()


def parse_spec(md: str) -> dict:
    title_m = re.search(r"^#\s+(.+?)\s+Specification\s*$", md, re.M)
    title = title_m.group(1) if title_m else "Unknown"
    purpose_m = re.search(r"^## Purpose\s*\n(.*?)(?=^## |\Z)", md, re.M | re.S)
    purpose = (purpose_m.group(1).strip() if purpose_m else "").replace("\n", " ")

    req_section = re.search(r"^## Requirements\s*\n(.*)", md, re.M | re.S)
    body = req_section.group(1) if req_section else md
    chunks = re.split(r"^### Requirement:\s*", body, flags=re.M)
    requirements = []
    for chunk in chunks[1:]:
        lines = chunk.split("\n")
        req_title = lines[0].strip()
        rest = "\n".join(lines[1:])
        scen_parts = re.split(r"^#### Scenario:\s*", rest, flags=re.M)
        desc = scen_parts[0].strip()
        scenarios = []
        for sp in scen_parts[1:]:
            sl = sp.split("\n")
            sname = sl[0].strip()
            steps_raw = "\n".join(sl[1:]).strip().split("\n")
            steps = []
            for ln in steps_raw:
                if ln.strip():
                    steps.append(parse_scenario_step(ln))
            scenarios.append({"name": sname, "steps": steps})
        requirements.append({"title": req_title, "desc": desc, "scenarios": scenarios})
    return {"title": title, "purpose": purpose, "requirements": requirements}


def render_dep_graph(spec_id: str, edges: list[tuple[str, str]]) -> str:
    if not edges:
        return ""
    nodes = []
    seen = {}
    for a, b in edges:
        for n in (a, b):
            if n not in seen:
                seen[n] = len(nodes)
                nodes.append(n)
    n = len(nodes)
    w, h = 700, max(120, 50 + n * 55)
    box_w = min(200, (w - 40) // max(n, 1))
    positions = {}
    for i, node in enumerate(nodes):
        x = 30 + i * ((w - 60) // max(n - 1, 1)) if n > 1 else w // 2 - box_w // 2
        positions[node] = (x, 30)
    svg_lines = [
        f'<div class="dep-graph" id="graph"><h3>🕸️ 需求依赖关系</h3>',
        f'<svg viewBox="0 0 {w} {h}" xmlns="http://www.w3.org/2000/svg">',
        '<defs><marker id="arrow" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">',
        '<path d="M0,0 L8,3 L0,6" fill="none" stroke="#58a6ff" stroke-width="1.2"/></marker></defs>',
    ]
    for node, (x, y) in positions.items():
        label = html.escape(node[:18] + ("…" if len(node) > 18 else ""))
        svg_lines.append(
            f'<rect x="{x}" y="{y}" width="{box_w}" height="36" rx="6" fill="#161b22" stroke="#58a6ff" stroke-width="1.5"/>'
            f'<text x="{x + box_w/2}" y="{y + 22}" text-anchor="middle" fill="#58a6ff" font-size="11" font-weight="600">{label}</text>'
        )
    for a, b in edges:
        if a in positions and b in positions:
            ax, ay = positions[a]
            bx, by = positions[b]
            x1, y1 = ax + box_w / 2, ay + 36
            x2, y2 = bx + box_w / 2, by
            svg_lines.append(
                f'<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="#58a6ff" stroke-width="1.2" marker-end="url(#arrow)" opacity="0.7"/>'
            )
    svg_lines.append("</svg></div>")
    return "\n".join(svg_lines)


def render_spec_page(spec_id: str, parsed: dict, hub_link: str = "../index.html") -> str:
    reqs = parsed["requirements"]
    full_text = "\n".join(r["desc"] for r in reqs)
    for r in reqs:
        for s in r["scenarios"]:
            for _, _, body in s["steps"]:
                full_text += "\n" + body
    kw = count_keywords(full_text)
    n_scen = sum(len(r["scenarios"]) for r in reqs)
    must_total = kw["must"]

    parts = [
        TEMPLATE_CSS_START,
        f"{html.escape(parsed['title'])} — OpenSpec Viewer",
        TEMPLATE_HEAD_END,
        '<div class="header">',
        f"<h1>📐 {html.escape(parsed['title'])} Specification</h1>",
        f'<div class="purpose">{html.escape(parsed["purpose"])}</div>',
        '<div class="meta">',
        f"<span>📋 {len(reqs)} Requirements</span>",
        f"<span>🧪 {n_scen} Scenarios</span>",
        f'<span>🔑 RFC 2119</span>',
        f'<span><a href="{html.escape(hub_link)}">← 全部 Spec</a></span>',
        "</div></div>",
        '<div class="container"><div class="layout">',
        '<nav class="sidebar"><h3>Requirements</h3>',
    ]
    for i, r in enumerate(reqs, 1):
        rid = f"req{i}"
        short = html.escape(r["title"][:28] + ("…" if len(r["title"]) > 28 else ""))
        parts.append(f'<a href="#{rid}">{i}. {short}</a>')
    parts += [
        '<h3 style="margin-top:1.2rem">Quick Ref</h3>',
        '<a href="#legend">🔑 关键词图例</a>',
    ]
    if spec_id in DEP_GRAPHS:
        parts.append('<a href="#graph">🕸️ 依赖关系</a>')
    parts.append("</nav><main>")
    parts += [
        '<div class="summary-bar">',
        f'<div class="summary-item"><span class="num">{len(reqs)}</span> Requirements</div>',
        f'<div class="summary-item"><span class="num">{n_scen}</span> Scenarios</div>',
        f'<div class="summary-item"><span class="num">{must_total}</span> MUST 约束</div>',
        f'<div class="summary-item"><span class="num">{kw["should"]}</span> SHOULD</div>',
        "</div>",
        '<div class="legend" id="legend">',
        '<div class="legend-item"><span class="tag tag-must">MUST</span> 绝对要求</div>',
        '<div class="legend-item"><span class="tag tag-mustnot">MUST NOT</span> 绝对禁止</div>',
        '<div class="legend-item"><span class="tag tag-should">SHOULD</span> 推荐</div>',
        '<div class="legend-item"><span class="tag tag-may">MAY</span> 可选</div>',
        "</div>",
    ]
    edges = DEP_GRAPHS.get(spec_id, [])
    if edges:
        parts.append(render_dep_graph(spec_id, edges))

    for i, r in enumerate(reqs, 1):
        open_cls = " open" if i == 1 else ""
        tag = dominant_tag(r["desc"])
        n_s = len(r["scenarios"])
        desc_html = format_desc_block(r["desc"])
        parts.append(f'<div class="req-card{open_cls}" id="req{i}">')
        parts.append('<div class="req-header">')
        parts.append(f'<span class="req-num">{i}</span>')
        parts.append(f'<span class="req-title">{html.escape(r["title"])}</span>')
        parts.append('<span class="req-tags">')
        parts.append(f'<span class="tag tag-{tag}">{tag.upper().replace("MUSTNOT", "MUST NOT")}</span>')
        parts.append(f'<span class="tag tag-count">{n_s} scenarios</span>')
        parts.append('<span class="req-toggle">▸</span></div>')
        parts.append('<div class="req-body">')
        parts.append(f'<div class="req-desc">{desc_html}</div>')
        for sc in r["scenarios"]:
            parts.append('<div class="scenario">')
            parts.append(f'<div class="scenario-title">{html.escape(sc["name"])}</div>')
            parts.append('<ul class="scenario-steps">')
            for cls, label, body in sc["steps"]:
                parts.append(
                    f'<li class="step-{cls}"><span class="step-label">{label}</span> '
                    f"{highlight_keywords(body)}</li>"
                )
            parts.append("</ul></div>")
        parts.append("</div></div>")

    parts.append("</main></div></div>")
    parts.append(SCRIPT_FOOT)
    return "".join(parts)


def render_hub(specs: list[tuple[str, dict, str]]) -> str:
    parts = [
        TEMPLATE_CSS_START,
        "OpenSpec — 全部规范",
        TEMPLATE_HEAD_END,
        '<div class="header">',
        "<h1>📐 go-macro OpenSpec 规范总览</h1>",
        '<div class="purpose">交互式 HTML 阅读器：可折叠需求卡片、RFC 2119 关键词高亮、Given/When/Then 场景、依赖关系图。</div>',
        f'<div class="meta"><span>共 {len(specs)} 份 Specification</span></div>',
        "</div>",
        '<div class="container"><main>',
        '<div class="legend" id="legend">',
        '<div class="legend-item"><span class="tag tag-must">MUST</span> 绝对要求</div>',
        '<div class="legend-item"><span class="tag tag-mustnot">MUST NOT</span> 绝对禁止</div>',
        '<div class="legend-item"><span class="tag tag-should">SHOULD</span> 推荐</div>',
        '<div class="legend-item"><span class="tag tag-may">MAY</span> 可选</div>',
        "</div>",
        '<div class="hub-grid">',
    ]
    order = [
        "macro-core",
        "macro-expander",
        "macro-codegen",
        "macro-contrib",
        "macro-repo-layout",
        "syntax-inline",
        "syntax-try",
    ]
    by_id = {sid: (p, path) for sid, p, path in specs}
    for sid in order:
        if sid not in by_id:
            continue
        parsed, rel = by_id[sid]
        n_req = len(parsed["requirements"])
        n_scen = sum(len(r["scenarios"]) for r in parsed["requirements"])
        parts.append(f'<a class="hub-card" href="{html.escape(rel)}">')
        parts.append(f"<h2>{html.escape(parsed['title'])}</h2>")
        parts.append(f"<p>{html.escape(parsed['purpose'][:120])}…</p>" if len(parsed["purpose"]) > 120 else f"<p>{html.escape(parsed['purpose'])}</p>")
        parts.append('<div class="hub-stats">')
        parts.append(f"<span>📋 {n_req} Requirements</span>")
        parts.append(f"<span>🧪 {n_scen} Scenarios</span>")
        parts.append("</div></a>")
    parts.append("</div></main></div>")
    parts.append("</body></html>")
    return "".join(parts)


def main() -> int:
    root = Path(__file__).resolve().parents[3]  # go-macro repo root from skill dir
    if len(sys.argv) > 1:
        root = Path(sys.argv[1]).resolve()
    specs_dir = root / "openspec" / "specs"
    out_dir = root / "openspec" / "viewer"
    if not specs_dir.is_dir():
        print(f"specs dir not found: {specs_dir}", file=sys.stderr)
        return 1

    out_dir.mkdir(parents=True, exist_ok=True)
    collected: list[tuple[str, dict, str]] = []

    for spec_md in sorted(specs_dir.glob("*/spec.md")):
        spec_id = spec_md.parent.name
        md = spec_md.read_text(encoding="utf-8")
        parsed = parse_spec(md)
        sub = out_dir / spec_id
        sub.mkdir(exist_ok=True)
        html_path = sub / "index.html"
        page = render_spec_page(spec_id, parsed, hub_link="../index.html")
        html_path.write_text(page, encoding="utf-8")
        collected.append((spec_id, parsed, f"{spec_id}/index.html"))
        print(f"wrote {html_path}")

    hub = render_hub(collected)
    (out_dir / "index.html").write_text(hub, encoding="utf-8")
    print(f"wrote {out_dir / 'index.html'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
