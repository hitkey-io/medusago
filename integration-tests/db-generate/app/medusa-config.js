// Configuration of the reference application generate.sh runs upstream
// `medusa db:generate` / `medusa db:migrate` with (copied over a copy of the
// reference application): the brand test module whose models are the
// versions under ../versions.
const { loadEnv, defineConfig } = require("@medusajs/framework/utils")

loadEnv(process.env.NODE_ENV || "development", process.cwd())

module.exports = defineConfig({
  projectConfig: {
    databaseUrl: process.env.DATABASE_URL,
    http: {
      jwtSecret: "supersecret",
      cookieSecret: "supersecret",
    },
  },
  admin: { disable: true },
  // The Medusa Cloud plugin is loaded by default; MedusaGo does not have it.
  plugins: [{ resolve: "@medusajs/cloud", options: { disabled: true } }],
  modules: [{ resolve: "./src/modules/brand" }],
})
