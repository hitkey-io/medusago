// Configuration of the reference application the scenarios are recorded
// with (copy testapp/ over a reference application directory).
const { loadEnv, defineConfig } = require("@medusajs/framework/utils")
const fs = require("fs")
const path = require("path")

loadEnv(process.env.NODE_ENV || "development", process.cwd())

// The index module `schema` option: MEDUSA_TEST_INDEX_SCHEMA names a GraphQL
// file of testapp/index-schemas (record.sh passes MEDUSA_HTTP_DIFF_DIR),
// MEDUSA_TEST_INDEX_SCHEMA_RESTART replaces it (set by a restart step).
const indexSchemaName =
  process.env.MEDUSA_TEST_INDEX_SCHEMA_RESTART ||
  process.env.MEDUSA_TEST_INDEX_SCHEMA
const indexOptions = indexSchemaName
  ? {
      schema: fs.readFileSync(
        path.join(
          process.env.MEDUSA_HTTP_DIFF_DIR || __dirname,
          process.env.MEDUSA_HTTP_DIFF_DIR ? "testapp/index-schemas" : "index-schemas",
          indexSchemaName
        ),
        "utf8"
      ),
    }
  : undefined

module.exports = defineConfig({
  projectConfig: {
    databaseUrl: process.env.DATABASE_URL,
    databaseLogging: process.env.MEDUSA_DB_LOGGING === "true",
    http: {
      storeCors: process.env.STORE_CORS,
      adminCors: process.env.ADMIN_CORS,
      authCors: process.env.AUTH_CORS,
      jwtSecret: process.env.JWT_SECRET || "supersecret",
      cookieSecret: process.env.COOKIE_SECRET || "supersecret",
    },
  },
  admin: { disable: true },
  // Flag-gated modules need both the feature flag and the module entry
  // (index engine, translation); the brand test module (custom module and
  // links) is enabled with MEDUSA_TEST_BRAND.
  modules: [
    ...(process.env.MEDUSA_FF_INDEX_ENGINE === "true"
      ? [{ resolve: "@medusajs/index", options: indexOptions }]
      : []),
    ...(process.env.MEDUSA_FF_TRANSLATION === "true"
      ? [{ resolve: "@medusajs/medusa/translation" }]
      : []),
    ...(process.env.MEDUSA_TEST_BRAND === "true"
      ? [{ resolve: "./src/modules/brand" }]
      : []),
  ],
})
