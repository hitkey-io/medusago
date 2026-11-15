// Brings the imported (not hand-rewritten) pages in line with MedusaGo: the product name in prose, and the notes that only
// say in which Medusa release a feature appeared (MedusaGo ships the whole feature set of its baseline, so they don't apply).
//   bun scripts/polish-text.mjs [--write] [content subdirectory ...]     (default: a dry run over all of content/)
import fs from "node:fs"
import path from "node:path"
import { applyTerms, dropVersionNotes, dropVersionLines } from "./lib/polish.mjs"

const root = path.resolve(import.meta.dirname, "..")
const write = process.argv.includes("--write")
const dirs = process.argv.slice(2).filter((a) => !a.startsWith("--"))
const files = []
const walk = (d) => {
  for (const e of fs.readdirSync(d, { withFileTypes: true })) {
    const p = path.join(d, e.name)
    if (e.isDirectory()) walk(p)
    else if (/\.mdx$/.test(e.name)) files.push(p)
  }
}
for (const d of dirs.length ? dirs : ["."]) walk(path.join(root, "content", d))

let changed = 0
for (const f of files) {
  const before = fs.readFileSync(f, "utf8")
  // hand-rewritten pages are owned by us; the learn chapters were edited by hand too (they contrast Medusa and MedusaGo on purpose)
  if (path.relative(root, f).startsWith("content/learn/")) continue
  // the API reference is generated from the specs on every build and describes the HTTP API as upstream words it
  if (path.relative(root, f).startsWith("content/api/")) continue
  if (/^---\n(?:[^\n]*\n)*?rewritten: true\n/.test(before)) continue
  const after = applyTerms(dropVersionLines(dropVersionNotes(before)))
  if (after === before) continue
  changed++
  if (write) fs.writeFileSync(f, after)
  else {
    const a = before.split("\n"), b = new Set(after.split("\n"))
    const gone = a.filter((l) => !b.has(l) && l.trim())
    console.log(`\n# ${path.relative(root, f)}`)
    for (const l of gone.slice(0, 6)) console.log("  - " + l.trim().slice(0, 170))
  }
}
console.log(`\n${changed} file(s) ${write ? "changed" : "would change"}`)
