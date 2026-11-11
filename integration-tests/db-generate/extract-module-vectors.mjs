#!/usr/bin/env node
//
// Captures what `medusa db:generate <module>` generates upstream for every
// module of the Medusa monorepo, starting from an empty snapshot: the
// statements of the generated migration and the snapshot file.
//
// Usage:
//   node integration-tests/db-generate/extract-module-vectors.mjs [UPSTREAM_DIR]
//
// The models are loaded like loadResources does (every export of the files
// of dist/models, index.* excluded, in directory order) and go through the
// upstream generator (Migrations.generate over mikroOrmCreateConnection with
// the CustomTsMigrationGenerator) into a temporary directory. The output,
// golden/modules.json, is compared by generate_test.go. Medusa Enterprise
// Edition modules are skipped.

import { createRequire } from "node:module"
import fs from "node:fs"
import os from "node:os"
import path from "node:path"
import { fileURLToPath } from "node:url"
import { parseMigration } from "./parse-migration.mjs"

const HERE = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(HERE, "../..")
const UPSTREAM = path.resolve(process.argv[2] ?? path.join(ROOT, "reference/medusa"))
const OUT = path.join(HERE, "golden/modules.json")

const require = createRequire(path.join(UPSTREAM, "package.json"))
const utils = require("@medusajs/framework/utils")
const { Migrations } = require(path.join(UPSTREAM, "packages/core/utils/dist/migrations"))
const { mikroOrmCreateConnection } = utils.DALUtils ?? utils

const enterprisePaths = fs
  .readFileSync(path.join(UPSTREAM, "ENTERPRISE-LICENSE.md"), "utf8")
  .split("\n")
  .map((l) => l.match(/^- `(.*)`$/)?.[1])
  .filter(Boolean)
  .map((p) => p.replace(/\/$/, ""))

async function generate(models, moduleName) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "gen-vectors-"))
  utils.mikroORMEntityBuilder.clear()
  const orm = await mikroOrmCreateConnection(
    {
      clientUrl: "postgres://postgres@localhost:5432/unused",
      snapshotName: `.snapshot-${utils.kebabCase(moduleName.replace("Service", ""))}`,
    },
    utils.toMikroOrmEntities(models),
    dir
  )
  const result = await new Migrations(orm).generate()
  const files = fs.readdirSync(dir)
  const snapshotFile = files.find((f) => f.startsWith(".snapshot-"))
  const out = {
    snapshotName: snapshotFile,
    snapshot: fs.readFileSync(path.join(dir, snapshotFile), "utf8"),
    ...(result.fileName ? parseMigration(result.code) : { up: null, down: null }),
  }
  fs.rmSync(dir, { recursive: true, force: true })
  return out
}

function loadModels(dir) {
  const out = []
  for (const file of fs.readdirSync(dir)) {
    if (file.startsWith("index.") || [".ts.map", ".js.map", ".d.ts"].some((e) => file.endsWith(e))) {
      continue
    }
    if (!fs.statSync(path.join(dir, file)).isFile()) {
      continue
    }
    for (const v of Object.values(require(path.join(dir, file)))) {
      if (utils.DmlEntity.isDmlEntity(v) && !out.includes(v)) {
        out.push(v)
      }
    }
  }
  return out
}

const vectors = {}
for (const mod of fs.readdirSync(path.join(UPSTREAM, "packages/modules")).sort()) {
  const pkgDir = `packages/modules/${mod}`
  const distModels = path.join(UPSTREAM, pkgDir, "dist/models")
  if (!fs.existsSync(distModels) || enterprisePaths.includes(pkgDir)) {
    continue
  }
  const models = loadModels(distModels)
  if (!models.length) {
    continue
  }
  vectors[mod] = await generate(models, mod)
}
fs.mkdirSync(path.dirname(OUT), { recursive: true })
fs.writeFileSync(OUT, JSON.stringify(vectors, null, 2) + "\n")
console.log(`modules: ${Object.keys(vectors).length}`)
