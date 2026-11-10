// Builds the Medusa admin dashboard with the original @medusajs/admin-bundler
// (Vite), like `medusa build` upstream (compiler.buildAppFrontend).
//
// Usage: node build-admin.mjs <options.json>
//
// options.json:
//   {
//     "resolveFrom": "<dir whose node_modules provide @medusajs/admin-bundler>",
//     "workDir": "<cwd of the build: .medusa/client is written there>",
//     "outDir": "<absolute output directory>",
//     "mode": "runtime" | "static",
//     "path": "/app", "backendUrl": "/", "storefrontUrl": "", "maxUploadFileSize": 1048576,
//     "sources": ["<project>/src/admin"], "plugins": []
//   }
//
// "static" passes the admin options to the bundler exactly like upstream: the
// values are compiled into the bundle (used for `medusa build --admin-only`).
//
// "runtime" produces a template the Go server configures when it serves it:
// the base path is the placeholder BASE_PLACEHOLDER, and the values upstream
// compiles in with Vite `define` read globals that the server sets in an
// inline script inserted at RUNTIME_MARKER in index.html.

import { readFile, writeFile } from "node:fs/promises"
import { createRequire } from "node:module"
import path from "node:path"

const BASE_PLACEHOLDER = "/__MEDUSA_ADMIN_BASE__"
const RUNTIME_MARKER = "<!--__MEDUSA_ADMIN_RUNTIME_CONFIG__-->"

const options = JSON.parse(await readFile(process.argv[2], "utf-8"))

const require = createRequire(path.join(options.resolveFrom, "package.json"))
let bundlerPath
try {
  bundlerPath = require.resolve("@medusajs/admin-bundler")
} catch {
  console.error(
    `Cannot find @medusajs/admin-bundler from ${options.resolveFrom}. Install the admin build dependencies (npm install @medusajs/admin-bundler @medusajs/dashboard) to compile admin extensions.`
  )
  process.exit(1)
}
const { build } = require(bundlerPath)

process.chdir(options.workDir)

const bundlerOptions = {
  disable: false,
  sources: options.sources ?? [],
  plugins: options.plugins ?? [],
  path: options.path ?? "/app",
  backendUrl: options.backendUrl,
  storefrontUrl: options.storefrontUrl,
  maxUploadFileSize: options.maxUploadFileSize,
  outDir: options.outDir,
}

if (options.mode === "runtime") {
  bundlerOptions.path = BASE_PLACEHOLDER
  bundlerOptions.vite = () => ({
    define: {
      __BASE__: "__MEDUSA_ADMIN_PATH__",
      __BACKEND_URL__: "__MEDUSA_ADMIN_BACKEND_URL__",
      __STOREFRONT_URL__: "__MEDUSA_ADMIN_STOREFRONT_URL__",
      __AUTH_TYPE__: "__MEDUSA_ADMIN_AUTH_TYPE__",
      __JWT_TOKEN_STORAGE_KEY__: "__MEDUSA_ADMIN_JWT_TOKEN_STORAGE_KEY__",
      __MAX_UPLOAD_FILE_SIZE__: "__MEDUSA_ADMIN_MAX_UPLOAD_FILE_SIZE__",
      "process.env": "__MEDUSA_ADMIN_PROCESS_ENV__",
    },
  })
}

await build(bundlerOptions)

if (options.mode === "runtime") {
  const htmlPath = path.join(options.outDir, "index.html")
  const html = await readFile(htmlPath, "utf-8")
  if (!html.includes("<head>")) {
    throw new Error("index.html has no <head> element")
  }
  await writeFile(
    htmlPath,
    html.replace("<head>", `<head>\n    ${RUNTIME_MARKER}`)
  )
}
