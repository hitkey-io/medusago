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
const [section, ...pages] = process.argv.slice(2)
const appDir = { "user-guide": "user-guide/app", learn: "book/app/learn", resources: "resources/app", ui: "ui/app" }[section]
if (!appDir) throw new Error("unknown section " + section)

const ENTERPRISE = [/^settings\/roles/, /^settings\/policies/, /^enterprise(\/|$)/]
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
  text = dropSections(text, /permissions across guides/i)
  // RBAC is an Enterprise feature that MedusaGo does not include.
  text = text.replace(/^[ \t]*<PermissionsBadge\b[^>]*\/>[ \t]*\n(\n)?/gm, "")
  const flagged = text.split("\n").filter((l) => /rbac|role-based|enterprise|\bpolicies\b|\bPermissionsBadge\b|EnterpriseNotice/i.test(l))
  if (flagged.length) console.log(`  REVIEW ${file}: ${flagged.length} line(s) mention RBAC/Enterprise`)
  fs.mkdirSync(path.dirname(dst), { recursive: true })
  fs.writeFileSync(dst, text)
  console.log("imported", file)
}
fs.writeFileSync(imagesFile, JSON.stringify(images, null, 2) + "\n")
