// Fills a code fence from a file of the compile-checked examples module:
//
//   ```go title="src/modules/brand/models.go" file="brand/models.go"
//   ```
//
// `region="name"` keeps only the lines between `// region: name` and
// `// endregion: name` (markers are removed, indentation is trimmed). Without a
// region, all region markers are stripped. Paths are relative to www/examples.
import fs from "node:fs"
import path from "node:path"
import { visit } from "unist-util-visit"

const root = path.resolve(import.meta.dirname, "../examples")
const attr = (meta, name) => new RegExp(`(?:^|\\s)${name}="([^"]+)"`).exec(meta || "")

const MARKER = /^\s*\/\/ (?:end)?region: .*$/

function region(text, name, file) {
  const lines = text.split("\n")
  const start = lines.findIndex((l) => l.trim() === `// region: ${name}`)
  const end = lines.findIndex((l) => l.trim() === `// endregion: ${name}`)
  if (start < 0 || end < start) throw new Error(`region "${name}" not found in ${file}`)
  const body = lines.slice(start + 1, end).filter((l) => !MARKER.test(l))
  const indent = Math.min(...body.filter((l) => l.trim()).map((l) => /^\t*/.exec(l)[0].length))
  return body.map((l) => l.replace(new RegExp(`^\\t{0,${indent}}`), "")).join("\n")
}

export function remarkIncludeCode() {
  return (tree, vfile) => {
    visit(tree, "code", (node) => {
      const f = attr(node.meta, "file")
      if (!f) return
      const full = path.join(root, f[1])
      if (!fs.existsSync(full)) throw new Error(`${vfile.path}: example file not found: ${f[1]}`)
      let text = fs.readFileSync(full, "utf8")
      const r = attr(node.meta, "region")
      text = r ? region(text, r[1], f[1]) : text.split("\n").filter((l) => !MARKER.test(l)).join("\n")
      node.value = text.replace(/\n+$/, "")
      node.meta = node.meta.replace(/\s*file="[^"]+"/, "").replace(/\s*region="[^"]+"/, "").trim()
    })
  }
}
