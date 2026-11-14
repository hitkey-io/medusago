// Takes the admin screenshots of the documentation from a running MedusaGo stand
// (scripts/screenshots/reset.sh) with the original dashboard, like the upstream pictures:
// the window without browser chrome, 1435x807 CSS px at 2x, light theme.
//   bun scripts/screenshots/shoot.mjs [shot-id ...]      (no ids: every shot of shots.mjs)
import { chromium } from "playwright-core"
import fs from "node:fs"
import path from "node:path"
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
  localStorage.setItem("medusa_admin_sidebar_open", "true")
})

// log in through the form, like a user (the stand runs with NODE_ENV=production, so the session
// cookie is Secure; Chrome accepts it on localhost)
await page.goto(BASE + "/app/login", { waitUntil: "networkidle" })
await page.getByPlaceholder("Email").fill("admin@medusa-test.com")
await page.getByPlaceholder("Password").fill("supersecret")
await page.getByRole("button", { name: "Continue with Email" }).click()
await page.waitForURL((u) => !u.pathname.endsWith("/login"), { timeout: 15000 })

let failed = 0
for (const shot of shots) {
  if (only.size && !only.has(shot.id)) continue
  try {
    await page.goto(BASE + "/app" + shot.path, { waitUntil: "networkidle" })
    await page.waitForTimeout(shot.wait ?? 800)
    if (shot.run) await shot.run(page)
    await page.waitForTimeout(300)
    const file = path.join(OUT, `${shot.id}.png`)
    await page.screenshot({ path: file, clip: shot.clip })
    console.log("ok  ", shot.id)
  } catch (e) {
    failed++
    console.log("FAIL", shot.id, String(e.message).split("\n")[0])
  }
}
await browser.close()
process.exit(failed ? 1 : 0)
