// The MedusaGo version shown in the header: the newest release tag of the repository,
// linked to that tag on GitHub. Needs the tags (CI checks out with fetch-depth 0);
// without them the committed value is kept.
import { execFileSync } from "node:child_process"
import fs from "node:fs"
import path from "node:path"

const root = path.resolve(import.meta.dirname, "..")
const out = path.join(root, "src/generated/version.json")
const repo = "https://github.com/hitkey-io/medusago"

let version = null
try {
  const tag = execFileSync("git", ["tag", "--list", "v[0-9]*", "--sort=-v:refname"], { cwd: root, encoding: "utf8" })
    .split("\n")
    .find(Boolean)
  if (tag) {
    const date = execFileSync("git", ["log", "-1", "--format=%cI", `${tag}^{commit}`], { cwd: root, encoding: "utf8" }).trim()
    version = { number: tag.slice(1), releaseUrl: `${repo}/releases/tag/${tag}`, releaseDate: date }
  }
} catch {
  version = null
}

const existing = fs.existsSync(out) ? JSON.parse(fs.readFileSync(out, "utf8")) : null
const result = version ?? existing
if (!result) throw new Error("no release tag and no committed version")
fs.writeFileSync(out, JSON.stringify(result, null, 2) + "\n")
console.log(`version: v${result.number}`)
