// Takes the admin screenshots of the documentation from a running MedusaGo stand
// (scripts/screenshots/reset.sh) with the original dashboard, like the upstream pictures:
// the window without browser chrome, 1435x807 CSS px at 2x, light theme.
//   bun scripts/screenshots/shoot.mjs [shot-id ...]      (no ids: every shot of shots.mjs)
import { chromium } from "playwright-core"
import fs from "node:fs"
import path from "node:path"
import os from "node:os"
import sharp from "sharp"
import { shots } from "./shots.mjs"

const BASE = process.env.STAND_URL || "http://localhost:9000"
const OUT = path.resolve(import.meta.dirname, "../../public/images/shots")
const only = new Set(process.argv.slice(2))
fs.mkdirSync(OUT, { recursive: true })

const browser = await chromium.launch({ channel: "chrome", headless: true })
const context = await browser.newContext({
  viewport: { width: 1435, height: 807 },
  deviceScaleFactor: 2,
  colorScheme: "light",
  locale: "en-US",
  timezoneId: "Europe/Copenhagen",
  // `start` runs with NODE_ENV=production, where the session cookie is Secure and only issued
  // behind a TLS proxy: say so, like a real proxy would
  extraHTTPHeaders: { "x-forwarded-proto": "https" },
})
const page = await context.newPage()
await page.addInitScript(() => {
  localStorage.setItem("medusa_admin_theme", "light")
})

// the Admin API with the logged-in session, to find the records a shot needs
const api = async (p) => (await context.request.get(`${BASE}/admin${p}`)).json()
// writes, for the few shots that need a record in a given state (the change stays in the stand's database)
const send = async (method, p, data) => {
  const r = await context.request.fetch(`${BASE}/admin${p}`, { method, data })
  if (!r.ok()) throw new Error(`${method} ${p}: ${r.status()} ${(await r.text()).slice(0, 160)}`)
  return r.json().catch(() => ({}))
}
const ctx = { api, send, BASE }

// log in through the form, like a user
async function login() {
  await page.goto(BASE + "/app/login", { waitUntil: "networkidle" })
  await page.getByPlaceholder("Email").fill("admin@medusa-test.com")
  await page.getByPlaceholder("Password").fill("supersecret")
  await page.getByRole("button", { name: "Continue with Email" }).click()
  await page.waitForURL((u) => !u.pathname.endsWith("/login"), { timeout: 15000 })
}

let failed = 0
let loggedIn = false
for (const shot of shots) {
  if (shot.file || (only.size && !only.has(shot.id))) continue
  try {
    if (shot.anonymous) {
      // pictures of the pages before login: a clean context without the session
      await context.clearCookies()
      loggedIn = false
    } else if (!loggedIn) {
      await login()
      loggedIn = true
    }
    const route = typeof shot.route === "function" ? await shot.route(ctx) : shot.route
    await page.goto(BASE + "/app" + route, { waitUntil: "networkidle" })
    await page.waitForTimeout(shot.wait ?? 900)
    if (shot.run) await shot.run(page, ctx)
    await page.waitForTimeout(400)
    const file = path.join(os.tmpdir(), `shot-${shot.id}.png`) // converted to WebP below
    if (shot.clip) {
      // a computed crop in CSS px (a form: from its title to its last field)
      const clip = await shot.clip(page, ctx); if (process.env.DEBUG_CLIP) console.log(shot.id, JSON.stringify(clip)); await page.screenshot({ path: file, clip })
    } else if (shot.el) {
      // a section/form: the element only, like the cropped upstream pictures
      const loc = await shot.el(page, ctx)
      await loc.first().screenshot({ path: file })
    } else {
      await page.screenshot({ path: file })
    }
    // text-heavy screenshots stay sharp at a fraction of the PNG size
    await sharp(file).webp({ quality: 90, effort: 6 }).toFile(path.join(OUT, `${shot.id}.webp`))
    fs.rmSync(file)
    console.log("ok  ", shot.id)
  } catch (e) {
    failed++
    // what the page looked like, to see why (kept out of the site: .fail/ is ignored by git)
    fs.mkdirSync(path.resolve(import.meta.dirname, ".fail"), { recursive: true })
    await page.screenshot({ path: path.resolve(import.meta.dirname, `.fail/${shot.id}.png`) }).catch(() => {})
    console.log("FAIL", shot.id, String(e.message).split("\n")[0].slice(0, 160))
  }
}
await browser.close()
process.exit(failed ? 1 : 0)
