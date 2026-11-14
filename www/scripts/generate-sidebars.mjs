// Generates src/generated/<section>-sidebar.mjs from sidebars/<section>.mjs with
// the upstream sidebar generator (autogenerate_path, titles from page metadata,
// numbering). The generator reads pages from ./app, so it runs in a temporary
// directory whose "app" is a link to content/<section>.
import fs from "node:fs"
import os from "node:os"
import path from "node:path"
import { pathToFileURL } from "node:url"

const root = path.resolve(import.meta.dirname, "..")
const sections = process.argv.slice(2)
const all = sections.length ? sections : fs.readdirSync(path.join(root, "sidebars")).map((f) => f.replace(/\.mjs$/, ""))

const { generateSidebar } = await import(pathToFileURL(path.join(root, "vendor/build-scripts/generate-sidebar.ts")).href)

// Sidebar items whose page is not in content/ yet are left out, so the sidebar
// always matches the pages that exist (and a section can be filled step by step).
const pageExists = (section, p) => {
  const rel = p.replace(/^\/|\/$/g, "")
  return fs.existsSync(path.join(root, "content", section, rel, "page.mdx"))
}
const dirHasPages = (section, p) => {
  const dir = path.join(root, "content", section, p.replace(/^\/|\/$/g, ""))
  if (!fs.existsSync(dir)) return false
  return fs.readdirSync(dir, { recursive: true }).some((f) => String(f).endsWith("page.mdx"))
}
function prune(section, items, pruned) {
  const out = []
  for (const item of items || []) {
    const copy = { ...item }
    if (copy.custom_autogenerate) {
      pruned.push(copy.custom_autogenerate)
      continue
    }
    if (copy.children) copy.children = prune(section, copy.children, pruned)
    if (copy.type === "separator") {
      if (out.length && out[out.length - 1].type !== "separator") out.push(copy)
      continue
    }
    if ((copy.type === "link" || copy.type === "ref") && copy.path && !pageExists(section, copy.path)) {
      pruned.push(copy.path)
      if (!copy.children?.length) continue
      // keep the children of a missing page: hoist them
      out.push(...copy.children)
      continue
    }
    if (copy.autogenerate_path && !dirHasPages(section, copy.autogenerate_path)) {
      pruned.push(copy.autogenerate_path)
      if (!copy.children?.length) continue
      delete copy.autogenerate_path
    }
    if ((copy.type === "category" || copy.type === "sub-category") && !copy.autogenerate_path && !copy.autogenerate_tags && !copy.children?.length) {
      continue
    }
    out.push(copy)
  }
  while (out.length && out[out.length - 1].type === "separator") out.pop()
  return out
}

for (const section of all) {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "sidebar-"))
  fs.symlinkSync(path.join(root, "content", section), path.join(tmp, "app"))
  process.chdir(tmp)
  const { sidebar } = await import(pathToFileURL(path.join(root, "sidebars", section + ".mjs")).href)
  const pruned = []
  const raw = sidebar.filter((sb) => !sb.custom_autogenerate).map((sb) => ({ ...sb, items: prune(section, sb.items, pruned) }))
  const generated = await generateSidebar(raw, { writeToFile: false, addNumbering: section === "learn" })
  if (pruned.length) console.log(`  ${pruned.length} sidebar entries without a page yet`)
  fs.writeFileSync(
    path.join(root, "src/generated", `${section}-sidebar.mjs`),
    `export const generatedSidebars = ${JSON.stringify(generated, null, 2)}\n`
  )
  fs.rmSync(tmp, { recursive: true, force: true })
  console.log("generated sidebar for", section)
}
