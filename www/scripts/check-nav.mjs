// The top navigation (vendor/docs-ui/constants.tsx) and the homepage are rendered
// by JavaScript, so scripts/check-links.mjs cannot see their links. This checks
// every site path they use against the built pages.
import fs from "node:fs"
import path from "node:path"

const root = path.resolve(import.meta.dirname, "..")
const dist = path.join(root, "dist")
const sources = [
  "vendor/docs-ui/constants.tsx",
  "vendor/docs-ui/components/MainNav/index.tsx",
  "vendor/docs-ui/components/MainNav/DesktopMenu/index.tsx",
  ...fs.readdirSync(path.join(root, "src/components/Homepage"), { recursive: true }).filter((f) => String(f).endsWith(".tsx")).map((f) => `src/components/Homepage/${f}`),
]
const exists = (p) => fs.existsSync(path.join(dist, p, "index.html")) || fs.existsSync(path.join(dist, p + ".html"))

const missing = new Map()
for (const file of sources) {
  const text = fs.readFileSync(path.join(root, file), "utf8")
  for (const m of text.matchAll(/(?:link|href):\s*[`"](\/[^`"$#?]*)[`"]/g)) {
    const p = m[1].replace(/\/+$/, "") || "/"
    if (p !== "/" && !exists(p)) missing.set(p, [...(missing.get(p) || []), path.basename(file)])
  }
}
for (const [p, files] of missing) console.error(`navigation link ${p} has no page (${[...new Set(files)].join(", ")})`)
if (missing.size && process.env.STRICT_LINKS) process.exit(1)
console.log(missing.size ? `${missing.size} navigation link(s) without a page` : "navigation ok")
