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
const imagesFile = path.join(root, "images.json")
const images = fs.existsSync(imagesFile) ? JSON.parse(fs.readFileSync(imagesFile, "utf8")) : []

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
  if (ENTERPRISE.some((re) => re.test(rel))) {
    console.log("skip enterprise:", rel)
    continue
  }
  const src = path.join(upstream, appDir, rel, "page.mdx")
  const dst = path.join(root, "content", section, rel, "page.mdx")
  let text = fs.readFileSync(src, "utf8")
  const file = `${section}/${rel || "index"}`
  text = text.replace(/^[ \t]*!\[([^\]]*)\]\((https?:\/\/[^)]+)\)[ \t]*\n?/gm, (_, alt, url) => {
    images.push({ page: file, alt, upstream: url, status: "todo" })
    return ""
  })
  text = dropSections(text, /permissions across guides|single sign-on|assign roles in invites|view a user's roles/i)
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
  // RBAC is an Enterprise feature that MedusaGo does not include.
  text = text.replace(/^[ \t]*<PermissionsBadge\b[^>]*\/>[ \t]*\n(\n)?/gm, "")
  const flagged = text.split("\n").filter((l) => /rbac|role-based|enterprise|\bpolicies\b|\bPermissionsBadge\b|EnterpriseNotice/i.test(l))
  if (flagged.length) console.log(`  REVIEW ${file}: ${flagged.length} line(s) mention RBAC/Enterprise`)
  fs.mkdirSync(path.dirname(dst), { recursive: true })
  fs.writeFileSync(dst, text)
  console.log("imported", file)
}
fs.writeFileSync(imagesFile, JSON.stringify(images, null, 2) + "\n")
