#!/usr/bin/env node

// Temporary markdown-to-html renderer for local docs editing workflow.
// It is not part of the production release pipeline; the embedded site docs
// still come from frontend/public/docs/index.html during GitHub Actions builds.

import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '..')
const markedModule = pathToFileURL(
  path.join(repoRoot, 'frontend/node_modules/marked/lib/marked.esm.js'),
).href
const { marked } = await import(markedModule)

const sourcePath = process.argv[2] || '/opt/sub2api/data/docs-source.md'
const outputPath = process.argv[3] || '/opt/sub2api/data/public/docs/index.html'
const sourceDir = path.dirname(sourcePath)
const outputDir = path.dirname(outputPath)

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

function isExternalUrl(value) {
  return /^(https?:)?\/\//i.test(value) || /^data:/i.test(value)
}

function slugify(text) {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\u4e00-\u9fff]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function uniqueSlug(text, seen) {
  const base = slugify(text) || 'section'
  const count = seen.get(base) || 0
  seen.set(base, count + 1)
  return count === 0 ? base : `${base}-${count}`
}

function extractImagePaths(markdown) {
  const results = []
  const regex = /!\[[^\]]*]\(([^)\s]+)(?:\s+"[^"]*")?\)/g
  let match
  while ((match = regex.exec(markdown)) !== null) {
    const raw = match[1].trim().replace(/^<|>$/g, '')
    if (!raw || isExternalUrl(raw) || raw.startsWith('/')) {
      continue
    }
    results.push(raw)
  }
  return [...new Set(results)]
}

async function copyRelativeAssets(markdown) {
  const imagePaths = extractImagePaths(markdown)
  for (const relativePath of imagePaths) {
    const sourceFile = path.resolve(sourceDir, relativePath)
    const targetFile = path.resolve(outputDir, relativePath)
    await fs.mkdir(path.dirname(targetFile), { recursive: true })
    await fs.copyFile(sourceFile, targetFile)
  }
}

const markdown = await fs.readFile(sourcePath, 'utf8')
await fs.mkdir(outputDir, { recursive: true })
await copyRelativeAssets(markdown)

const tokens = marked.lexer(markdown, { gfm: true, breaks: false })
const slugSeen = new Map()
const headingIds = []
const toc = []

for (const token of tokens) {
  if (token.type !== 'heading') {
    continue
  }
  const id = uniqueSlug(token.text, slugSeen)
  headingIds.push(id)
  if (token.depth >= 2 && token.depth <= 3) {
    toc.push({ id, depth: token.depth, text: token.text })
  }
}

const pageTitleToken = tokens.find((token) => token.type === 'heading' && token.depth === 1)
const pageTitle = pageTitleToken?.text || '天才程序员拼车站文档'
const leadToken = tokens.find((token) => token.type === 'paragraph')
const pageDescription = leadToken?.text || '天才程序员拼车站使用文档'

const headingQueue = [...headingIds]
const renderer = new marked.Renderer()

renderer.heading = function heading(token) {
  const id = headingQueue.shift() || uniqueSlug(token.text, slugSeen)
  const inner = this.parser.parseInline(token.tokens)
  return `<h${token.depth} id="${id}">${inner}</h${token.depth}>`
}

renderer.link = function link(token) {
  const href = token.href || '#'
  const titleAttr = token.title ? ` title="${escapeHtml(token.title)}"` : ''
  const externalAttrs = isExternalUrl(href) ? ' target="_blank" rel="noopener noreferrer"' : ''
  const inner = this.parser.parseInline(token.tokens)
  return `<a href="${escapeHtml(href)}"${titleAttr}${externalAttrs}>${inner}</a>`
}

renderer.image = function image(token) {
  const src = token.href || ''
  const alt = token.text || ''
  const titleAttr = token.title ? ` title="${escapeHtml(token.title)}"` : ''
  return `<img src="${escapeHtml(src)}" alt="${escapeHtml(alt)}"${titleAttr} />`
}

const articleHtml = marked.parse(markdown, {
  gfm: true,
  breaks: false,
  renderer,
})

const tocHtml = toc
  .map(
    (item) =>
      `<a class="toc-item toc-depth-${item.depth}" href="#${escapeHtml(item.id)}">${escapeHtml(item.text)}</a>`,
  )
  .join('\n')

const html = `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>${escapeHtml(pageTitle)}</title>
    <meta name="description" content="${escapeHtml(pageDescription)}" />
    <style>
      :root {
        color-scheme: light;
        --bg: #f3f7fb;
        --panel: rgba(255, 255, 255, 0.92);
        --panel-strong: #ffffff;
        --text: #152033;
        --muted: #607087;
        --line: rgba(18, 28, 45, 0.08);
        --primary: #0f766e;
        --primary-deep: #115e59;
        --primary-soft: rgba(15, 118, 110, 0.1);
        --shadow: 0 26px 70px rgba(15, 23, 42, 0.08);
        --code-bg: #0c1628;
        --code-text: #eff6ff;
      }

      @media (prefers-color-scheme: dark) {
        :root {
          color-scheme: dark;
          --bg: #07111f;
          --panel: rgba(9, 18, 32, 0.92);
          --panel-strong: #0b1527;
          --text: #eef4ff;
          --muted: #94a3b8;
          --line: rgba(148, 163, 184, 0.14);
          --primary: #49d0c2;
          --primary-deep: #31b3a5;
          --primary-soft: rgba(73, 208, 194, 0.12);
          --shadow: 0 26px 70px rgba(0, 0, 0, 0.35);
          --code-bg: #020817;
          --code-text: #dbeafe;
        }
      }

      * {
        box-sizing: border-box;
      }

      html {
        scroll-behavior: smooth;
      }

      body {
        margin: 0;
        font-family:
          "PingFang SC",
          "Hiragino Sans GB",
          "Microsoft YaHei",
          "Segoe UI",
          sans-serif;
        color: var(--text);
        background:
          radial-gradient(circle at top left, rgba(20, 184, 166, 0.12), transparent 28%),
          radial-gradient(circle at top right, rgba(59, 130, 246, 0.14), transparent 24%),
          linear-gradient(180deg, var(--bg), var(--bg));
        line-height: 1.72;
      }

      a {
        color: inherit;
        text-decoration: none;
      }

      .page {
        width: min(1240px, calc(100% - 32px));
        margin: 0 auto;
        padding: 24px 0 72px;
      }

      .topbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        margin-bottom: 20px;
      }

      .brand {
        display: inline-flex;
        align-items: center;
        gap: 14px;
        min-width: 0;
      }

      .brand img {
        width: 48px;
        height: 48px;
        border-radius: 14px;
        background: #fff;
        box-shadow: 0 10px 24px rgba(15, 23, 42, 0.12);
        object-fit: contain;
      }

      .brand-copy strong {
        display: block;
        font-size: 18px;
        line-height: 1.2;
      }

      .brand-copy span {
        display: block;
        margin-top: 4px;
        color: var(--muted);
        font-size: 13px;
      }

      .top-actions {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        justify-content: flex-end;
      }

      .button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        padding: 11px 16px;
        border-radius: 999px;
        border: 1px solid var(--line);
        background: var(--panel);
        color: var(--text);
        font-size: 14px;
        font-weight: 700;
        backdrop-filter: blur(12px);
      }

      .button.primary {
        border-color: transparent;
        background: linear-gradient(135deg, var(--primary-deep), var(--primary));
        color: #fff;
      }

      .hero {
        border: 1px solid var(--line);
        border-radius: 30px;
        background:
          linear-gradient(135deg, rgba(15, 118, 110, 0.08), rgba(59, 130, 246, 0.08)),
          var(--panel-strong);
        box-shadow: var(--shadow);
        padding: 28px 30px;
      }

      .hero h1 {
        margin: 0;
        font-size: clamp(30px, 4vw, 46px);
        line-height: 1.08;
        letter-spacing: -0.03em;
      }

      .hero p {
        margin: 14px 0 0;
        max-width: 780px;
        color: var(--muted);
        font-size: 16px;
      }

      .layout {
        display: grid;
        grid-template-columns: 280px minmax(0, 1fr);
        gap: 24px;
        margin-top: 24px;
      }

      .sidebar {
        position: sticky;
        top: 20px;
        align-self: start;
        border: 1px solid var(--line);
        border-radius: 24px;
        background: var(--panel);
        box-shadow: var(--shadow);
        padding: 18px;
        backdrop-filter: blur(14px);
      }

      .sidebar h2 {
        margin: 0 0 12px;
        font-size: 15px;
      }

      .toc {
        display: grid;
        gap: 8px;
      }

      .toc-item {
        display: block;
        padding: 10px 12px;
        border-radius: 14px;
        color: var(--muted);
        font-size: 14px;
        transition: 0.18s ease;
      }

      .toc-item:hover {
        background: var(--primary-soft);
        color: var(--primary);
      }

      .toc-depth-3 {
        margin-left: 12px;
        font-size: 13px;
      }

      .help {
        margin-top: 18px;
        padding-top: 16px;
        border-top: 1px solid var(--line);
      }

      .help p {
        margin: 0 0 10px;
        color: var(--muted);
        font-size: 13px;
      }

      .help img {
        width: 100%;
        max-width: 180px;
        border-radius: 18px;
        border: 1px solid var(--line);
        background: #fff;
      }

      .article-shell {
        border: 1px solid var(--line);
        border-radius: 28px;
        background: var(--panel);
        box-shadow: var(--shadow);
        padding: 28px;
        backdrop-filter: blur(14px);
      }

      .markdown-body h1 {
        margin: 0 0 18px;
        font-size: 36px;
        line-height: 1.15;
      }

      .markdown-body h2 {
        margin: 38px 0 14px;
        padding-bottom: 10px;
        border-bottom: 1px solid var(--line);
        font-size: 28px;
        line-height: 1.2;
      }

      .markdown-body h3 {
        margin: 28px 0 10px;
        font-size: 21px;
      }

      .markdown-body h4 {
        margin: 22px 0 10px;
        font-size: 17px;
      }

      .markdown-body p,
      .markdown-body li {
        color: var(--muted);
        font-size: 15px;
      }

      .markdown-body p {
        margin: 0 0 14px;
      }

      .markdown-body ul,
      .markdown-body ol {
        margin: 0 0 16px;
        padding-left: 22px;
      }

      .markdown-body li + li {
        margin-top: 7px;
      }

      .markdown-body strong {
        color: var(--text);
      }

      .markdown-body a {
        color: var(--primary);
        text-decoration: underline;
        text-underline-offset: 3px;
      }

      .markdown-body img {
        display: block;
        max-width: 100%;
        height: auto;
        margin: 14px 0 20px;
        border-radius: 18px;
        border: 1px solid var(--line);
        background: var(--panel-strong);
        box-shadow: var(--shadow);
      }

      .markdown-body pre {
        margin: 14px 0 18px;
        overflow-x: auto;
        border-radius: 18px;
        background: var(--code-bg);
        color: var(--code-text);
        padding: 16px 18px;
        font-size: 13px;
        line-height: 1.65;
      }

      .markdown-body code {
        font-family:
          "SFMono-Regular",
          Consolas,
          "Liberation Mono",
          Menlo,
          monospace;
      }

      .markdown-body :not(pre) > code {
        display: inline-block;
        padding: 2px 8px;
        border-radius: 999px;
        background: var(--primary-soft);
        color: var(--primary);
        font-size: 13px;
        font-weight: 700;
      }

      .markdown-body blockquote {
        margin: 16px 0;
        padding: 14px 16px;
        border-left: 4px solid var(--primary);
        border-radius: 0 16px 16px 0;
        background: rgba(15, 118, 110, 0.06);
      }

      .markdown-body hr {
        margin: 26px 0;
        border: 0;
        border-top: 1px solid var(--line);
      }

      .markdown-body table {
        width: 100%;
        border-collapse: collapse;
        margin: 14px 0 18px;
        min-width: 640px;
      }

      .markdown-body th,
      .markdown-body td {
        padding: 12px 14px;
        border-bottom: 1px solid var(--line);
        text-align: left;
        vertical-align: top;
        font-size: 14px;
      }

      .markdown-body th {
        color: var(--text);
        background: rgba(148, 163, 184, 0.08);
        font-size: 13px;
        text-transform: uppercase;
        letter-spacing: 0.04em;
      }

      .table-wrap {
        overflow-x: auto;
      }

      .footer {
        margin-top: 22px;
        color: var(--muted);
        font-size: 13px;
        text-align: center;
      }

      @media (max-width: 980px) {
        .layout {
          grid-template-columns: 1fr;
        }

        .sidebar {
          position: static;
        }
      }

      @media (max-width: 720px) {
        .page {
          width: min(100% - 24px, 100%);
          padding: 16px 0 48px;
        }

        .topbar {
          flex-direction: column;
          align-items: stretch;
        }

        .top-actions {
          justify-content: flex-start;
        }

        .hero,
        .article-shell {
          padding: 20px;
        }

        .markdown-body h1 {
          font-size: 30px;
        }

        .markdown-body h2 {
          font-size: 24px;
        }
      }
    </style>
  </head>
  <body>
    <div class="page">
      <header class="topbar">
        <a class="brand" href="/home">
          <img src="/logo.png" alt="天才程序员拼车站 Logo" />
          <span class="brand-copy">
            <strong>天才程序员拼车站</strong>
            <span>一群程序员共建的高质量 AI API 拼车站</span>
          </span>
        </a>
        <div class="top-actions">
          <a class="button" href="/register">注册</a>
          <a class="button" href="/login">登录</a>
          <a class="button primary" href="/redeem">去兑换 / 去购买</a>
        </div>
      </header>

      <section class="hero">
        <h1>${escapeHtml(pageTitle)}</h1>
        <p>${escapeHtml(pageDescription)}</p>
      </section>

      <div class="layout">
        <aside class="sidebar">
          <h2>目录</h2>
          <nav class="toc">
            ${tocHtml}
          </nav>
          <div class="help">
            <p>客服 QQ：2143428872</p>
          </div>
        </aside>

        <main class="article-shell">
          <article class="markdown-body">
            ${articleHtml}
          </article>
        </main>
      </div>

      <div class="footer">
        天才程序员拼车站文档页。若站内能力更新，以站内实际页面和公告为准。
      </div>
    </div>
  </body>
</html>
`

await fs.writeFile(outputPath, html, 'utf8')
console.log(`rendered ${sourcePath} -> ${outputPath}`)
