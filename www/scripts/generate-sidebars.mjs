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

for (const section of all) {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "sidebar-"))
  fs.symlinkSync(path.join(root, "content", section), path.join(tmp, "app"))
  process.chdir(tmp)
  const { sidebar } = await import(pathToFileURL(path.join(root, "sidebars", section + ".mjs")).href)
  const generated = await generateSidebar(sidebar, { writeToFile: false, addNumbering: section === "learn" })
  fs.writeFileSync(
    path.join(root, "src/generated", `${section}-sidebar.mjs`),
    `export const generatedSidebars = ${JSON.stringify(generated, null, 2)}\n`
  )
  fs.rmSync(tmp, { recursive: true, force: true })
  console.log("generated sidebar for", section)
}
