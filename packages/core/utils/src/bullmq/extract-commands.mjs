// Copies the Lua commands of the bullmq version used by upstream (includes
// already inlined by its build) to ./commands, named
// <command>-<number of keys>.lua like the bullmq dist scripts, and the
// bullmq license next to them.
//
//   node packages/core/utils/src/bullmq/extract-commands.mjs
//
// MEDUSA_REFERENCE_DIR overrides the upstream checkout (reference/medusa).
import { createRequire } from "node:module"
import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, "../../../../..")
const reference = process.env.MEDUSA_REFERENCE_DIR ?? path.join(root, "reference/medusa")
const bullmqDir = path.join(reference, "node_modules/bullmq")
const require = createRequire(path.join(bullmqDir, "package.json"))
const version = JSON.parse(fs.readFileSync(path.join(bullmqDir, "package.json"), "utf8")).version
const scriptsDir = path.join(bullmqDir, "dist/cjs/scripts")
const outDir = path.join(here, "commands")

fs.rmSync(outDir, { recursive: true, force: true })
fs.mkdirSync(outDir, { recursive: true })
let count = 0
for (const file of fs.readdirSync(scriptsDir).sort()) {
  if (!file.endsWith(".js") || file === "index.js") continue
  const mod = require(path.join(scriptsDir, file))
  for (const def of Object.values(mod)) {
    fs.writeFileSync(path.join(outDir, `${def.name}-${def.keys}.lua`), def.content)
    count++
  }
}
fs.copyFileSync(path.join(bullmqDir, "LICENSE"), path.join(here, "LICENSE"))
fs.writeFileSync(path.join(here, "VERSION"), version + "\n")
console.log(`bullmq ${version}: ${count} commands`)
