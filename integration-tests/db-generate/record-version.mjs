#!/usr/bin/env node
//
// record-version.mjs <golden.json> <migrations dir> <snapshot file> [migration.ts]
//
// Writes the golden of one model version (used by generate.sh): the
// statements of the migration `medusa db:generate` generated (null when it
// reported no changes) and the snapshot file content. The TypeScript
// migration is then converted to a CommonJS file the JavaScript reference
// application can run.

import fs from "node:fs"
import path from "node:path"
import { parseMigration } from "./parse-migration.mjs"

const [out, dir, snapshotFile, migration] = process.argv.slice(2)

const golden = { up: null, down: null }
if (migration) {
  const code = fs.readFileSync(migration, "utf8")
  Object.assign(golden, parseMigration(code))
  const className = path.basename(migration, ".ts")
  const js = code
    .replace(/^import \{ Migration \} from (".*?");$/m, "const { Migration } = require($1);")
    .replace(/^export class /m, "class ")
    .replace(/override async (up|down)\(\): Promise<void> \{/g, "async $1() {")
  fs.writeFileSync(path.join(dir, `${className}.js`), `${js}\nexports.${className} = ${className};\n`)
  fs.rmSync(migration)
}
golden.snapshot = fs.readFileSync(snapshotFile, "utf8")
fs.writeFileSync(out, JSON.stringify(golden, null, 2) + "\n")
