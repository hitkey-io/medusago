// React streams the content of Suspense boundaries after the page shell (a hidden
// <div id="…S:n"> plus a $RC() script). Browsers swap it in, but crawlers and the
// search indexer do not run scripts. This puts every boundary's content back in
// place, exactly the markup React would have produced without streaming, so the
// HTML is complete and hydration still finds its <!--$--> markers.
import fs from "node:fs"
import path from "node:path"

const dist = path.resolve(import.meta.dirname, "../dist")
const walk = (d) =>
  fs.readdirSync(d, { withFileTypes: true }).flatMap((e) => (e.isDirectory() ? walk(path.join(d, e.name)) : [path.join(d, e.name)]))

// Returns [start, end) of the balanced <div …>…</div> starting at `start`.
function balancedDiv(html, start) {
  const re = /<div\b|<\/div>/g
  re.lastIndex = start
  let depth = 0
  for (let m; (m = re.exec(html)); ) {
    depth += m[0] === "</div>" ? -1 : 1
    if (depth === 0) return [start, m.index + m[0].length]
  }
  throw new Error("unbalanced div")
}

function inline(html) {
  let changed = 0
  for (;;) {
    const m = /<!--\$\?--><template id="([^"]+)"><\/template>/.exec(html)
    if (!m) break
    const boundary = m[1]
    const segment = boundary.replace(/B:/, "S:")
    const open = `<div hidden id="${segment}">`
    const at = html.indexOf(open)
    if (at < 0) throw new Error(`no streamed content for ${boundary}`)
    const [s, e] = balancedDiv(html, at)
    const content = html.slice(s + open.length, e - "</div>".length)
    html = html.slice(0, s) + html.slice(e) // drop the hidden segment
    const close = html.indexOf("<!--/$-->", m.index)
    html = html.slice(0, m.index) + "<!--$-->" + content + html.slice(close)
    html = html.replace(new RegExp(`;?\\$RC\\("${boundary}","${segment}"\\)`), "")
    changed++
  }
  return { html, changed }
}

let pages = 0
for (const file of walk(dist).filter((f) => f.endsWith(".html"))) {
  const source = fs.readFileSync(file, "utf8")
  if (!source.includes("<!--$?-->")) continue
  const { html, changed } = inline(source)
  if (html.includes("<!--$?-->")) throw new Error(`pending boundaries left in ${file}`)
  fs.writeFileSync(file, html)
  pages += changed ? 1 : 0
}
console.log(`inlined streamed content in ${pages} pages`)
