// Copies pages of the Medusa documentation (MIT) into content/, converting
// what must differ on this site:
//  - images hosted by Medusa are removed and recorded in images.json (every
//    image of this site is ours, see the plan);
//  - Enterprise pages are never copied.
// Usage: node scripts/import-upstream.mjs <section> <upstream page dir>...
import fs from "node:fs"
import path from "node:path"

const root = path.resolve(import.meta.dirname, "..")
const upstream = path.resolve(root, "../reference/medusa/www/apps")
const [section, ...args] = process.argv.slice(2)
const appDir = { "user-guide": "user-guide/app", learn: "book/app/learn", resources: "resources/app", ui: "ui/app" }[section]
if (!appDir) throw new Error("unknown section " + section)

function walk(dir, rel = "") {
  return fs.readdirSync(path.join(dir, rel), { withFileTypes: true }).flatMap((d) => {
    const p = rel ? `${rel}/${d.name}` : d.name
    if (d.isDirectory()) return d.name.startsWith("_") ? [] : walk(dir, p)
    return d.name === "page.mdx" ? [rel] : []
  })
}
const allPages = args.some((a) => a === "--all" || a.endsWith("/**")) ? walk(path.join(upstream, appDir)).sort() : []
const pages = args.flatMap((a) => {
  if (a === "--all") return allPages
  if (a.endsWith("/**")) {
    const prefix = a.slice(0, -3)
    return allPages.filter((p) => p === prefix || p.startsWith(prefix + "/"))
  }
  return [a]
})

// Pages whose content is server-side TypeScript, left for a Go rewrite.
const SKIP = [
  /^resources\/nextjs-starter\/guides\/(customize-stripe|storefront-returns)$/,
  // infrastructure modules: configuration is TypeScript; a Go rewrite is pending
  /^resources\/infrastructure-modules(\/|$)/,
  // listings of TypeScript workflows/steps and TS-only customisation guides
  /^resources\/commerce-modules\/[^/]+\/(workflows|module-options|extend|guides)(\/|$)/,
]
// Backend sections: server-side TypeScript cannot be shown, so every chunk (from a heading to the
// next one) holding a ts/js fence is dropped; a page left with too little text is skipped.
const BACKEND_TS = /^resources\/(commerce-modules|infrastructure-modules)(\/|$)/
const MIN_WORDS = 90
const TS_FENCE = /^\s*```(ts|tsx|typescript|js|javascript)\b/
function dropTsChunks(text) {
  const lines = text.split("\n")
  const chunks = []
  let cur = []
  let inFence = false
  for (const line of lines) {
    if (/^\s*```/.test(line)) inFence = !inFence
    if (!inFence && /^#{2,4}\s/.test(line) && cur.length) {
      chunks.push(cur)
      cur = []
    }
    cur.push(line)
  }
  chunks.push(cur)
  // front matter / intro chunk (first) is kept unless it holds TS
  return chunks.filter((c) => !c.some((l) => TS_FENCE.test(l))).map((c) => c.join("\n")).join("\n")
}
const wordCount = (text) =>
  text.replace(/^---[\s\S]*?---/, "").replace(/```[\s\S]*?```/g, "").replace(/<[^>]+>|\{[^}]*\}|import .*|export .*/g, "").split(/\s+/).filter(Boolean).length

const ENTERPRISE = [/^settings\/roles/, /^settings\/policies/, /^enterprise(\/|$)/]
// Targeted edits for features MedusaGo does not include (RBAC, SSO). Each entry:
// [page regex, line regex] drops the matching paragraph, [page, regex, replacement] replaces text.
const EDITS = [
  [/^settings\/translations$/, /^With only the `translation:read` policy/],
  [/^settings\/users\/invites$/, /^4\. If your store uses the role-based access control/],
  [/^settings\/users\/invites$/, /^The \*\*Roles\*\* field only appears if your store uses the RBAC feature/],
  [/^settings\/users\/invites$/, / If your store uses the RBAC feature, the table also shows the roles of each invite\./, ""],
  [/^settings\/users$/, /^If your store uses the role-based access control \(RBAC\) feature, the user's details page/],
  [/^settings\/users$/, /^If your store uses the role-based access control \(RBAC\) feature, you can't delete the last user/],
]
// pictures we shot ourselves (scripts/screenshots) are kept at the place of the upstream one
const { shots } = await import("./screenshots/shots.mjs")
// an entry with `file` reuses the picture of another entry
const ownShots = new Map(shots.filter((x) => fs.existsSync(path.join(root, "public/images/shots", `${x.file ?? x.id}.png`))).map((x) => [`${x.page}|${x.alt}`, x.file ?? x.id]))
const imagesFile = path.join(root, "images.json")
// one row per picture of a page (older runs appended duplicates; "done" wins)
const images = []
for (const i of fs.existsSync(imagesFile) ? JSON.parse(fs.readFileSync(imagesFile, "utf8")) : []) {
  const same = images.find((x) => x.page === i.page && x.upstream === i.upstream)
  if (!same) images.push(i)
  else if (i.status === "done") same.status = "done"
}

// MedusaGo terminology. Applied to prose only (never inside code fences). "Medusa"
// stays where it names the upstream project, its API, the Admin or the UI kit.
const TERMS = [
  [/`medusa-config\.(ts|js)`/g, "`medusa-config.go`"],
  [/\bcreate-medusa-app\b(?!-errors)/g, "create-medusago-app"],
  [/\bMedusa (application|applications|server|backend|project|projects|instance|instances|installation)\b/gi, "MedusaGo $1"],
]
function applyTerms(text) {
  let inFence = false
  return text
    .split("\n")
    .map((line) => {
      if (/^\s*```/.test(line)) inFence = !inFence
      if (inFence || /^\s*```/.test(line)) return line
      return TERMS.reduce((l, [re, to]) => l.replace(re, to), line)
    })
    .join("\n")
}

// A page that is skipped must not linger from an earlier import.
const removeStale = (dst) => {
  if (fs.existsSync(dst) && !isRewritten(dst)) fs.rmSync(dst)
}

// Pages rewritten by hand (front matter "rewritten: true") are never overwritten.
const isRewritten = (file) => fs.existsSync(file) && /^---\n(?:[^\n]*\n)*?rewritten: true\n/.test(fs.readFileSync(file, "utf8"))

// Copies the MDX partials a page imports (import X from "./_x.mdx") that are not in content/ yet.
function copyImportedPartials(srcFile, dstFile) {
  const text = fs.readFileSync(dstFile, "utf8")
  for (const m of text.matchAll(/^import\s+\w+\s+from\s+"(\.{1,2}\/[^"]+\.mdx)"/gm)) {
    const partialSrc = path.resolve(path.dirname(srcFile), m[1])
    const partialDst = path.resolve(path.dirname(dstFile), m[1])
    if (fs.existsSync(partialDst) || !fs.existsSync(partialSrc)) continue
    fs.mkdirSync(path.dirname(partialDst), { recursive: true })
    fs.writeFileSync(partialDst, applyTerms(fs.readFileSync(partialSrc, "utf8")))
    copyImportedPartials(partialSrc, partialDst)
  }
}

// Removes the {...} object literals (and a following comma) that contain a match of `re`.
function removeObjects(text, re) {
  for (;;) {
    const m = re.exec(text)
    if (!m) return text
    let start = m.index
    for (let depth = 0; start > 0; start--) {
      if (text[start] === "}") depth++
      else if (text[start] === "{") {
        if (depth === 0) break
        depth--
      }
    }
    let end = start
    for (let depth = 0; end < text.length; end++) {
      if (text[end] === "{") depth++
      else if (text[end] === "}" && --depth === 0) break
    }
    end++
    if (text[end] === ",") end++
    while (start > 0 && /[ \t]/.test(text[start - 1])) start--
    if (text[start - 1] === "\n") start--
    text = text.slice(0, start) + text.slice(end)
  }
}

// Removes a "## heading" section (and a separator right before it) up to the next heading of the same or a higher level.
function dropSections(text, re) {
  const lines = text.split("\n")
  const out = []
  for (let i = 0; i < lines.length; i++) {
    const m = /^(#{1,6})\s+(.*)$/.exec(lines[i])
    if (m && re.test(m[2])) {
      while (out.length && /^(---|\s*)$/.test(out[out.length - 1])) out.pop()
      const level = m[1].length
      i++
      while (i < lines.length) {
        const n = /^(#{1,6})\s/.exec(lines[i])
        if (n && n[1].length <= level) break
        i++
      }
      i--
      out.push("")
      continue
    }
    out.push(lines[i])
  }
  return out.join("\n")
}

for (const page of pages) {
  const rel = page.replace(/^\/|\/$/g, "")
  if (SKIP.some((re) => re.test(`${section}/${rel}`))) {
    console.log("skip (needs a Go rewrite):", `${section}/${rel}`)
    removeStale(path.join(root, "content", section, rel, "page.mdx"))
    continue
  }
  if (ENTERPRISE.some((re) => re.test(rel))) {
    console.log("skip enterprise:", rel)
    continue
  }
  const src = path.join(upstream, appDir, rel, "page.mdx")
  const dst = path.join(root, "content", section, rel, "page.mdx")
  if (isRewritten(dst)) {
    console.log("keep rewritten:", `${section}/${rel}`)
    continue
  }
  let text = fs.readFileSync(src, "utf8")
  const file = `${section}/${rel || "index"}`
  text = text.replace(/^[ \t]*!\[([^\]]*)\]\((https?:\/\/[^)]+)\)[ \t]*\n?/gm, (_, alt, url) => {
    const own = ownShots.get(`${file}|${alt}`)
    const known = images.find((i) => i.page === file && i.upstream === url)
    if (known) known.status = own ? "done" : "todo"
    else images.push({ page: file, alt, upstream: url, status: own ? "done" : "todo" })
    return own ? `![${alt}](/images/shots/${own}.png)\n` : ""
  })
  // Links to the upstream site become links to the same page of this site.
  text = text.replace(/\]\(https:\/\/docs\.medusajs\.com(\/(?:api|resources|ui|user-guide|learn)(?:[/#)][^)]*)?)\)/g, "]($1)")
  // troubleshooting guides are not ported (they cover the Node installer and TypeScript server): drop their sections and imports
  if (/troubleshooting\/_sections/.test(text)) {
    text = text.replace(/^import \w+ from "[^"]*troubleshooting\/_sections[^"]*"\n/gm, "")
    text = dropSections(text, /troubleshooting|common (installation )?errors/i)
  }
  text = dropSections(text, /permissions across guides|single sign-on|errors with pnpm|how to create an auth module provider|how to create a custom payment provider|login with cloud|rbac module|assign roles in invites|view a user's roles/i)
  for (const [pageRe, re, replacement] of EDITS) {
    if (!pageRe.test(rel)) continue
    if (replacement !== undefined) {
      text = text.replace(re, replacement)
    } else {
      const lines = text.split("\n")
      const out = []
      for (let i = 0; i < lines.length; i++) {
        if (re.test(lines[i])) {
          if (lines[i + 1] === "") i++
          continue
        }
        out.push(lines[i])
      }
      text = out.join("\n")
    }
  }
  if (BACKEND_TS.test(`${section}/${rel}`) && !/\/(admin-widget-zones|js-sdk)(\/|$)/.test(rel)) {
    // Enterprise cards/rows (RBAC, SSO) in lists and tables
    text = removeObjects(text, /title: "(?:RBAC Module|OIDC \(SSO\))"/)
    text = text.split("\n").filter((l) => !/\bRBAC\b|RbacRole|<EnterpriseNotice|Serverless Application: Use the Commerce Modules|Node\.js Application: Use the Commerce Modules/.test(l)).join("\n")
    text = text.replace("generated randomly using the `crypto` module in Node.js.", "generated randomly with a cryptographically secure random generator.")
    // the loyalty plugin ships with MedusaGo: enabling it needs an import and a config entry
    text = text.replace(/### 2\. Run Migrations[\s\S]*?npx medusa db:migrate\n```/, [
      "### 1. Enable the Plugin",
      "",
      "The plugin is part of the MedusaGo repository. Import its package in the `main.go` file of your project to register it with the framework:",
      "",
      '```go title="main.go" file="loyalty/loyalty.go" region="import"',
      "```",
      "",
      "Then add it to the `Plugins` of your configuration in `medusa-config.go`:",
      "",
      '```go title="medusa-config.go" file="loyalty/loyalty.go" region="plugin"',
      "```",
      "",
      "### 2. Run Migrations",
      "",
      "Run the migrations to create the plugin's tables in the database:",
      "",
      "```bash",
      "go run . db:migrate",
      "```",
    ].join("\n"))
    text = text.replace(/^[ \t]*<CommerceModuleSections\b[^>]*\/>[ \t]*\n?/gm, "")
    text = dropTsChunks(text)
    if (wordCount(text) < MIN_WORDS) {
      console.log("skip (too little left after dropping TypeScript):", `${section}/${rel}`)
      removeStale(dst)
      continue
    }
  }
  // Medusa Cloud is not part of MedusaGo.
  text = text.replace(/<Note[^>]*title="Cloud [Uu]ser\??"[^>]*>[\s\S]*?<\/Note>\n*/g, "")
  text = text.replace("such as [Medusa Search](!cloud!/search) or PostgreSQL", "such as PostgreSQL")
  text = text.split("\n").filter((l) => !/available to Cloud users|\(!cloud!/.test(l)).join("\n")
  text = applyTerms(text)
  // Backend start commands. Admin extensions (src/admin) are compiled by the build, there is no dev server for them.
  const startBackend = (_, label) => "```bash" + (label ? ` badgeLabel="MedusaGo Application" badgeColor="green"` : "") + "\ngo run . develop\n```"
  text = text.replace(/```bash npm2yarn( badgeLabel="Medusa Application" badgeColor="green")?\nnpm run dev\n```/g, (m, label) =>
    label || !/^(learn\/(fundamentals\/admin|customization\/customize-admin))/.test(rel)
      ? startBackend(m, label)
      : "```bash\ngo run . build\n```\n\nThis compiles the admin extensions with Node.js (they are bundled by the original admin bundler). Then start the server as explained in the [Build](!docs!/learn/build) chapter.")
  text = text.replace(/```bash npx2yarnExec\nnpx create-medusa-app@latest --with-nextjs-starter\n```/g, "```bash\ngo run github.com/hitkey-io/medusago/packages/cli/create-medusago-app@latest my-medusa-store --with-nextjs-starter\n```")
  // RBAC is an Enterprise feature that MedusaGo does not include.
  text = text.replace(/^[ \t]*<PermissionsBadge\b[^>]*\/>[ \t]*\n(\n)?/gm, "")
  const flagged = text.split("\n").filter((l) => /rbac|role-based|enterprise|\bpolicies\b|\bPermissionsBadge\b|EnterpriseNotice/i.test(l))
  if (flagged.length) console.log(`  REVIEW ${file}: ${flagged.length} line(s) mention RBAC/Enterprise`)
  fs.mkdirSync(path.dirname(dst), { recursive: true })
  fs.writeFileSync(dst, text)
  copyImportedPartials(src, dst)
  // partials used by the page: _*.mdx files and _*/ directories next to it
  const srcDir = path.dirname(src)
  for (const entry of fs.readdirSync(srcDir, { withFileTypes: true })) {
    if (!entry.name.startsWith("_")) continue
    fs.cpSync(path.join(srcDir, entry.name), path.join(path.dirname(dst), entry.name), { recursive: true })
  }
  console.log("imported", file)
}
fs.writeFileSync(imagesFile, JSON.stringify(images, null, 2) + "\n")
