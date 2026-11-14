// Last-modified dates of the content pages (from git), used by the "Edited <date>" line
// under every page. Needs the full git history (CI checks out with fetch-depth 0).
// Without git history a page simply has no date.
import { execFileSync } from "node:child_process"
import fs from "node:fs"
import path from "node:path"

const root = path.resolve(import.meta.dirname, "..")
const out = path.join(root, "src/generated/edit-dates.json")

let dates = {}
try {
  const log = execFileSync("git", ["log", "--format=@@%aI", "--name-only", "--", "content"], {
    cwd: root,
    maxBuffer: 256 * 1024 * 1024,
    encoding: "utf8",
  })
  let current = ""
  for (const line of log.split("\n")) {
    if (line.startsWith("@@")) current = line.slice(2)
    else if (line.endsWith("/page.mdx")) {
      // newest first: keep the first date seen; names are relative to the repository root
      const file = "/" + line.replace(/^www\//, "")
      if (!(file in dates)) dates[file] = current
    }
  }
} catch {
  dates = {}
}

const existing = fs.existsSync(out) ? JSON.parse(fs.readFileSync(out, "utf8")) : {}
// a shallow checkout must not wipe the dates that are committed
const merged = Object.keys(dates).length ? dates : existing
const sorted = Object.fromEntries(Object.entries(merged).sort(([a], [b]) => a.localeCompare(b)))
fs.writeFileSync(out, JSON.stringify(sorted, null, 2) + "\n")
console.log(`edit dates: ${Object.keys(sorted).length} page(s)`)
