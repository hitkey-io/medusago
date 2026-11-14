// The Learn titles are written as `${pageNumber} Title`: the chapter number comes
// from the generated sidebar (see scripts/generate-sidebars.mjs, addNumbering).
// Every page gets `export const pageNumber = "<n>"` ("" when it has no number).
import fs from "node:fs"
import path from "node:path"
import { Parser } from "acorn"

const root = path.resolve(import.meta.dirname, "..")

let numbers
function load() {
  numbers = new Map()
  const file = path.join(root, "src/generated/learn-sidebar.mjs")
  if (!fs.existsSync(file)) return
  const text = fs.readFileSync(file, "utf8")
  const sidebars = JSON.parse(text.slice(text.indexOf("[")))
  const walk = (items) =>
    (items || []).forEach((it) => {
      if (it.path && it.number) numbers.set(it.path.replace(/\/$/, ""), it.number)
      walk(it.children)
    })
  sidebars.forEach((sb) => walk(sb.items))
}

export function remarkPageNumber() {
  return (tree, vfile) => {
    if (!numbers) load()
    const m = /\/content\/(.+?)\/?page\.mdx$/.exec(vfile.path || "")
    const sitePath = m ? "/" + m[1].replace(/\/$/, "") : ""
    const number = numbers.get(sitePath) || ""
    const value = `export const pageNumber = ${JSON.stringify(number)}`
    tree.children.unshift({
      type: "mdxjsEsm",
      value,
      data: { estree: Parser.parse(value, { ecmaVersion: 2022, sourceType: "module" }) },
    })
  }
}
