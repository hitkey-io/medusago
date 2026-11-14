// Fails when a page still carries wording of the upstream Node/TypeScript backend
// or of products MedusaGo does not include. Frontend sections (storefront, JS SDK,
// admin customisation, UI kit, user guide) legitimately talk about Node, npm and
// TypeScript, so those words are only checked in the shared rules below.
import fs from "node:fs"
import path from "node:path"

const root = path.resolve(import.meta.dirname, "..")
const content = path.join(root, "content")

const walk = (d) =>
  fs.readdirSync(d, { withFileTypes: true }).flatMap((e) => (e.isDirectory() ? walk(path.join(d, e.name)) : [path.join(d, e.name)]))

// Everywhere.
const ALWAYS = [
  [/\bMedusa (application|applications|server|backend|project|projects|instance)\b/i, 'use "MedusaGo application/server/…"'],
  [/medusa-config\.(ts|js)\b/, "use medusa-config.go"],
  [/\bnpx medusa\b|\byarn medusa\b/, 'use "go run . <command>"'],
  [/\bcreate-medusa-app\b(?!-errors)/, "use create-medusago-app"],
  [/@medusajs\/(framework\/(http|workflows-sdk|modules-sdk|utils|framework)|medusa|core-flows|workflows-sdk|modules-sdk|cli|test-utils)\b/, "backend TypeScript package"],
  [/\bMedusa Cloud\b|\bcloud\.medusajs\.com\b|\bmedusa-cloud\b|!cloud!|\bCloud (user|account|project)s?\b/i, "Medusa Cloud is not part of MedusaGo"],
  [/\b(RBAC|role-based access control|Single Sign-On|OIDC)\b/, "Enterprise feature"],
]
// Anywhere except the frontend sections: backend TypeScript.
const BACKEND_ONLY = [
  [/```(ts|tsx|typescript|js|javascript)\b/, "server code must be Go"],
  [/\bNode\.js\b|\bnpm\b|\byarn\b|\bpnpm\b/, "MedusaGo needs Go, not Node"],
]
const FRONTEND = /^(user-guide|ui|api)\/|^resources\/(storefront-development|js-sdk|nextjs-starter|admin-components|admin-widget-injection-zones|instantsearch)\/|^resources\/commerce-modules\/[^/]+\/(admin-widget-zones|js-sdk)\/|^learn\/(fundamentals\/admin|customization\/customize-admin|storefront-development)(\/|$)/

let problems = 0
for (const file of walk(content).filter((f) => f.endsWith(".mdx"))) {
  const rel = path.relative(content, file)
  const text = fs.readFileSync(file, "utf8")
  const rules = FRONTEND.test(rel) ? ALWAYS : [...ALWAYS, ...BACKEND_ONLY]
  text.split("\n").forEach((line, i) => {
    if (line.includes("terms-ok")) return // {/* terms-ok */}: the line names the upstream project on purpose
    for (const [re, why] of rules) {
      if (re.test(line)) {
        problems++
        console.error(`${rel}:${i + 1}: ${why}: ${line.trim().slice(0, 110)}`)
      }
    }
  })
}
if (problems) {
  console.error(`${problems} terminology problem(s)`)
  if (process.env.STRICT_TERMS) process.exit(1)
} else console.log("terms ok")
