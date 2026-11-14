// Run after the imports: markdown links to pages this site does not have are
// redirected to the closest page that exists (an API operation -> its tag page)
// or turned into plain text. Idempotent. Skips content/api (generated).
//   bun scripts/fix-links.mjs [--dry]
import fs from "node:fs"
import path from "node:path"

const root = path.resolve(import.meta.dirname, "..")
const content = path.join(root, "content")
const dry = process.argv.includes("--dry")

const walk = (d) =>
  fs.readdirSync(d, { withFileTypes: true }).flatMap((e) => (e.isDirectory() ? walk(path.join(d, e.name)) : [path.join(d, e.name)]))
const files = walk(content).filter((f) => f.endsWith(".mdx"))
const pages = new Set(files.filter((f) => f.endsWith("/page.mdx")).map((f) => "/" + path.relative(content, path.dirname(f)).replace(/\\/g, "/")))
pages.add("/")

const PREFIX = { docs: "", resources: "/resources", api: "/api", ui: "/ui", "user-guide": "/user-guide" }

// Site path of a link found in `file`, or null when it is external / not a page link.
function sitePath(link, file) {
  let url = link.split("#")[0]
  if (!url) return null
  const bang = /^!([\w-]+)!(.*)$/.exec(url)
  if (bang) {
    if (!(bang[1] in PREFIX)) return "!missing" // !cloud! and friends
    url = PREFIX[bang[1]] + bang[2]
  } else if (/^https?:|^mailto:/.test(url)) {
    return null
  } else if (!url.startsWith("/")) {
    if (!/page\.mdx?$/.test(url)) return null
    const section = path.relative(content, file).split(path.sep)[0]
    const abs = path.resolve(path.dirname(file), url)
    url = "/" + path.relative(content, path.dirname(abs)).replace(/\\/g, "/")
    if (!url.startsWith("/" + section)) return null
  }
  return url.replace(/\/+$/, "") || "/"
}

function redirect(p) {
  // /api/<area>/<tag>/<operation> -> /api/<area>/<tag>
  const m = /^\/api\/(admin|store)\/([^/]+)\/[^/]+$/.exec(p)
  if (m && pages.has(`/api/${m[1]}/${m[2]}`)) return `/api/${m[1]}/${m[2]}`
  if (/^\/api\/(admin|store)\//.test(p) && pages.has(p.split("/").slice(0, 3).join("/"))) return p.split("/").slice(0, 3).join("/")
  return null
}

let changed = 0
const dropped = new Map()
for (const file of files) {
  if (path.relative(content, file).startsWith("api" + path.sep)) continue
  const section = path.relative(content, file).split(path.sep)[0]
  let text = fs.readFileSync(file, "utf8")
  // JSX props (href: "/x", href="/x") written for an app served under /<section>: add the prefix
  text = text.replace(/(href(?::\s*|=))"(\/[^"#?]*)([^"]*)"/g, (whole, lead, p, rest) => {
    const clean = p.replace(/\/+$/, "") || "/"
    if (pages.has(clean) || !pages.has(`/${section}${clean}`)) return whole
    return `${lead}"/${section}${p}${rest}"`
  })
  // object props of card/prerequisite lists (link: "/x"): same prefix rule; a prop whose page is missing is removed
  text = text.replace(/^([ \t]*)(link|href):\s*"(\/[^"#?]*)([^"]*)"(,?)[ \t]*\n/gm, (whole, ind, key, p, rest, comma) => {
    const clean = p.replace(/\/+$/, "") || "/"
    if (pages.has(clean)) return whole
    if (pages.has(`/${section}${clean}`)) return `${ind}${key}: "/${section}${p}${rest}"${comma}\n`
    dropped.set(clean, (dropped.get(clean) || 0) + 1)
    return ""
  })
  const out = text.replace(/(?<!!)\[([^\]\n]*)\]\(([^)\s]+)\)/g, (whole, label, link) => {
    const p = sitePath(link, file)
    if (p === null || (p !== "!missing" && pages.has(p))) return whole
    const to = p === "!missing" ? null : redirect(p)
    const key = p === "!missing" ? link : p
    dropped.set(key, (dropped.get(key) || 0) + 1)
    if (to) return `[${label}](${link.startsWith("!api!") ? "!api!" + to.replace(/^\/api/, "") : to})`
    return label
  })
  if (out !== fs.readFileSync(file, "utf8")) {
    changed++
    if (!dry) fs.writeFileSync(file, out)
  }
}
console.log(`${changed} file(s) ${dry ? "would change" : "changed"}; links without a page: ${[...dropped.values()].reduce((a, b) => a + b, 0)}`)
const top = [...dropped].sort((a, b) => b[1] - a[1]).slice(0, 25)
for (const [k, n] of top) console.log(`  ${n}x ${k}`)
