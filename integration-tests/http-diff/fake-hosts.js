// Sends the upstream server's outbound requests for third-party hosts to the
// fake services of the http-diff harness (see fakes_test.go), like the Go
// harness does with http.DefaultTransport. The reference application's
// medusa-config.js requires this file when MEDUSA_TEST_FAKE_HOSTS is set:
//
//   MEDUSA_TEST_FAKE_PORT=12451
//   MEDUSA_TEST_FAKE_HOSTS=github.com=/github,api.github.com=/github-api,...
//
// Covers the global fetch (auth-github, auth-google token exchange) and
// jwks-rsa (auth-google signing keys).

const Module = require("module")

if (!global.__medusaTestFakeHosts) {
  global.__medusaTestFakeHosts = true

  const base = `http://localhost:${process.env.MEDUSA_TEST_FAKE_PORT}`
  const prefixes = Object.fromEntries(
    (process.env.MEDUSA_TEST_FAKE_HOSTS || "")
      .split(",")
      .map((p) => p.trim().split("="))
      .filter((p) => p.length === 2)
  )

  const rewrite = (url) => {
    const u = new URL(url)
    const prefix = prefixes[u.host]
    return prefix === undefined ? url : `${base}${prefix}${u.pathname}${u.search}`
  }

  const originalFetch = globalThis.fetch
  globalThis.fetch = (input, init) => {
    if (typeof input === "string" || input instanceof URL) {
      return originalFetch(rewrite(input.toString()), init)
    }
    return originalFetch(new Request(rewrite(input.url), input), init)
  }

  try {
    const fromGoogle = Module.createRequire(require.resolve("@medusajs/auth-google/package.json", { paths: [process.cwd()] }))
    const jwksPath = fromGoogle.resolve("jwks-rsa")
    const jwksClient = fromGoogle("jwks-rsa")
    const patched = Object.assign((options) => jwksClient({ ...options, jwksUri: rewrite(options.jwksUri) }), jwksClient)
    require.cache[jwksPath].exports = patched
  } catch (e) {
    // auth-google is not installed
  }
}
