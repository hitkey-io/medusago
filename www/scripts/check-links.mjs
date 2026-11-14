// Fails the build when a page links to an internal page that does not exist.
import fs from "node:fs"
import path from "node:path"

const dist = path.resolve(import.meta.dirname, "../dist")
const walk = (d) =>
  fs.readdirSync(d, { withFileTypes: true }).flatMap((e) => (e.isDirectory() ? walk(path.join(d, e.name)) : [path.join(d, e.name)]))
const files = walk(dist)
const exists = (p) =>
  files.includes(path.join(dist, p)) ||
  files.includes(path.join(dist, p, "index.html")) ||
  files.includes(path.join(dist, p + ".html"))

const broken = new Map()
for (const f of files.filter((f) => f.endsWith(".html"))) {
  const html = fs.readFileSync(f, "utf8")
  for (const m of html.matchAll(/href="(\/[^"#?]*)/g)) {
    const href = decodeURI(m[1])
    if (href.startsWith("//") || exists(href)) continue
    const page = "/" + path.relative(dist, f).replace(/index\.html$/, "")
    if (!broken.has(href)) broken.set(href, new Set())
    broken.get(href).add(page)
  }
}
for (const [href, pages] of broken) console.error(`broken link ${href} in ${[...pages].slice(0, 3).join(", ")}${pages.size > 3 ? ` (+${pages.size - 3})` : ""}`)
if (broken.size) process.exit(1)
console.log(`links ok (${files.filter((f) => f.endsWith(".html")).length} pages)`)
