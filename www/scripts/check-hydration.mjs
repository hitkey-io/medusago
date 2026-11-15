// Checks that the browser keeps the HTML that the server rendered. When the first render in the browser differs from the HTML, React throws the
// server-rendered tree away and renders it again, and the page flashes right after it loads. A page is "bad" when its heading, its first paragraph
// or its sidebar is a different DOM node after the page has loaded, or when it logs an error.
//   bun run preview &    (or any server of dist/)
//   bun scripts/check-hydration.mjs [pages=60] [base=http://localhost:4321] [file with one page path per line]
import { chromium } from "playwright-core"
import fs from "node:fs"
import path from "node:path"
const walk = (d) => fs.readdirSync(d, { withFileTypes: true }).flatMap((e) => (e.isDirectory() ? walk(path.join(d, e.name)) : e.name === "index.html" ? [path.join(d, e.name)] : []))
const all = walk(path.resolve(import.meta.dirname, "../dist")).map((f) => "/" + path.relative(path.resolve(import.meta.dirname, "../dist"), path.dirname(f)) + "/").filter((p) => !p.startsWith("/pagefind") && p !== "./")
let seed = 11; const rnd = (n) => { seed = (seed * 16807) % 2147483647; return seed % n }
const sample = process.argv[4] ? fs.readFileSync(process.argv[4], "utf8").split(/\s+/).filter(Boolean) : []
while (!process.argv[4] && sample.length < (+process.argv[2] || 60)) { const p = all[rnd(all.length)]; if (!sample.includes(p)) sample.push(p) }
const b = await chromium.launch({ channel: "chrome", headless: true })
const ctx = await b.newContext({ viewport: { width: 1440, height: 800 } })
await ctx.addInitScript(() => { document.addEventListener("DOMContentLoaded", () => { window.__ssr = { h1: document.querySelector("h1"), p: document.querySelector("[data-pagefind-body] p, [data-pagefind-body] li"), side: document.querySelector("[data-testid=sidebar-item-category-container]") } }) })
let bad = 0, errs = 0
for (const p of sample) {
  const page = await ctx.newPage()
  let err = ""
  page.on("pageerror", (e) => (err = String(e).slice(0, 50)))
  await page.goto((process.argv[3] || "http://localhost:4321") + p, { waitUntil: "networkidle" }); await page.waitForTimeout(900)
  const r = await page.evaluate(() => ({ h1: window.__ssr?.h1 ? window.__ssr.h1 === document.querySelector("h1") : null, p: window.__ssr?.p ? window.__ssr.p === document.querySelector("[data-pagefind-body] p, [data-pagefind-body] li") : null, side: window.__ssr?.side ? window.__ssr.side === document.querySelector("[data-testid=sidebar-item-category-container]") : null }))
  if (err) errs++
  if (r.h1 === false || r.p === false || r.side === false || err) { bad++; console.log("BAD", p, JSON.stringify(r), err) }
  await page.close()
}
if (bad) process.exitCode = 1
console.log(`checked ${sample.length} pages, ${bad} with a discarded tree or an error (${errs} errors)`)
await b.close()
