// Captures test vectors from the upstream module providers (file-s3,
// notification-sendgrid, analytics-posthog) for the Go unit tests: the
// requests they send to the third-party APIs, with a fixed clock and ULIDs.
//
//   node scripts/extract-provider-vectors.js <reference/medusa> <worktree root>
//
// Requires the built upstream checkout (reference/medusa with node_modules).
// Writes packages/modules/providers/<provider>/src/services/testdata/upstream-vectors.json.

const fs = require("fs")
const path = require("path")
const http = require("http")
const Module = require("module")

const [refRoot, outRoot] = process.argv.slice(2)
if (!refRoot || !outRoot) {
  console.error("usage: extract-provider-vectors.js <reference/medusa> <worktree root>")
  process.exit(1)
}
const req = Module.createRequire(path.join(refRoot, "package.json"))

const FIXED = Date.parse("2026-01-02T03:04:05.000Z")
const RealDate = Date
function freezeClock() {
  global.Date = class extends RealDate {
    constructor(...args) {
      super(...(args.length ? args : [FIXED]))
    }
    static now() {
      return FIXED
    }
  }
}
function restoreClock() {
  global.Date = RealDate
}

const logger = { error() {}, info() {}, warn() {}, debug() {} }

function write(provider, data) {
  const dir = path.join(outRoot, "packages/modules/providers", provider, "src/services/testdata")
  fs.mkdirSync(dir, { recursive: true })
  fs.writeFileSync(path.join(dir, "upstream-vectors.json"), JSON.stringify(data, null, 2) + "\n")
  console.log(`${provider}: ${path.join(dir, "upstream-vectors.json")}`)
}

// A local HTTP server recording the requests (method, url, selected headers,
// body) and answering with the given handler.
function recorder(handler) {
  const requests = []
  const server = http.createServer((r, res) => {
    const chunks = []
    r.on("data", (c) => chunks.push(c))
    r.on("end", () => {
      let raw = Buffer.concat(chunks)
      if (r.headers["content-encoding"] === "gzip") {
        raw = require("zlib").gunzipSync(raw)
      }
      const body = raw.toString("utf8")
      requests.push({ method: r.method, url: r.url, headers: r.headers, body })
      handler(r, res, body)
    })
  })
  return new Promise((resolve) => server.listen(0, "127.0.0.1", () => resolve({ server, requests, port: server.address().port })))
}

async function fileS3() {
  const ulid = req("ulid")
  const { S3FileService } = req("@medusajs/file-s3/dist/services/s3-file")
  // the provider calls (0, ulid_1.ulid)(): a fixed value keeps keys stable
  const ulidModule = req.cache[req.resolve("ulid")]
  ulidModule.exports = { ...ulid, ulid: () => "01KDYZZZZZZZZZZZZZZZZZZZZZ" }
  delete req.cache[req.resolve("@medusajs/file-s3/dist/services/s3-file")]
  const { S3FileService: Service } = req("@medusajs/file-s3/dist/services/s3-file")

  const configs = {
    path_style: {
      file_url: "http://localhost:9000/bucket-a", access_key_id: "AKIDEXAMPLE", secret_access_key: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
      region: "us-east-1", bucket: "bucket-a", endpoint: "http://localhost:9000", additional_client_config: { forcePathStyle: true }, prefix: "media/",
    },
    virtual_host: {
      file_url: "https://cdn.example.test", access_key_id: "AKIDEXAMPLE", secret_access_key: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
      region: "eu-west-1", bucket: "bucket-b", download_file_duration: 120,
    },
    us_east_1: {
      file_url: "https://cdn.example.test", access_key_id: "AKIDEXAMPLE", secret_access_key: "secret", session_token: "SESSIONTOKEN",
      region: "us-east-1", bucket: "bucket.with.dots",
    },
    acl_fixed: {
      file_url: "https://cdn.example.test", access_key_id: "AKIDEXAMPLE", secret_access_key: "secret",
      region: "ap-southeast-2", bucket: "bucket-c", acl: "bucket-owner-full-control", cache_control: "no-cache",
    },
    acl_disabled: {
      file_url: "https://cdn.example.test", access_key_id: "AKIDEXAMPLE", secret_access_key: "secret",
      region: "ap-southeast-2", bucket: "bucket-c", acl: false, prefix: "p/",
    },
  }
  const filenames = ["x/../y.png", "../../etc/passwd", "a b/c+d (1).txt", "/abs//dir/./f.tar.gz", "\\win\\path\\f.TXT", ".hidden", "trailing/", "dot.", "ünï cødé.pdf", "..", "./", "a/b/"]

  const out = { fixed_date: new RealDate(FIXED).toISOString(), configs, presigned: [], uploads: [], errors: [] }
  for (const [name, options] of Object.entries(configs)) {
    const s = new Service({ logger }, options)
    const sends = []
    s.client_.send = async (command) => {
      const input = { ...command.input }
      if (input.Body) {
        input.Body = Buffer.from(input.Body).toString("base64")
      }
      sends.push({ command: command.constructor.name, input })
      return {}
    }
    freezeClock()
    try {
      for (const fileKey of ["media/a b/c+d.txt", "plain.png"]) {
        out.presigned.push({ config: name, method: "download", fileKey, url: await s.getPresignedDownloadUrl({ fileKey }) })
      }
      for (const filename of filenames) {
        for (const access of [undefined, "public", "private"]) {
          try {
            const r = await s.getPresignedUploadUrl({ filename, mimeType: "image/png", access })
            out.presigned.push({ config: name, method: "upload", filename, access: access ?? null, result: r })
          } catch (e) {
            out.errors.push({ config: name, method: "getPresignedUploadUrl", filename, access: access ?? null, type: e.type, message: e.message })
          }
        }
      }
      const r = await s.getPresignedUploadUrl({ filename: "short.png", expiresIn: 30 })
      out.presigned.push({ config: name, method: "upload", filename: "short.png", access: null, expiresIn: 30, result: r })
    } finally {
      restoreClock()
    }
    for (const filename of filenames) {
      for (const [content, mimeType] of [["aGVsbG8=", "text/plain"], ["hello ünïcode", "text/csv"], ["\u0089PNGÿ", "image/png"], ["plain", undefined]]) {
        sends.length = 0
        try {
          const r = await s.upload({ filename, content, mimeType, access: "public" })
          out.uploads.push({ config: name, filename, content, mimeType: mimeType ?? null, access: "public", result: r, sent: sends.slice() })
        } catch (e) {
          out.errors.push({ config: name, method: "upload", filename, type: e.type, message: e.message })
        }
        if (name !== "path_style") {
          break
        }
      }
    }
    for (const access of [undefined, "private"]) {
      sends.length = 0
      const r = await s.upload({ filename: "f.txt", content: "x", mimeType: "text/plain", access })
      out.uploads.push({ config: name, filename: "f.txt", content: "x", mimeType: "text/plain", access: access ?? null, result: r, sent: sends.slice() })
    }
  }
  for (const [method, args] of [["upload", [undefined]], ["upload", [{ filename: "" }]], ["getPresignedUploadUrl", [{ filename: "" }]], ["getAsBuffer", [{ fileKey: "" }]], ["getDownloadStream", [{}]], ["getUploadStream", [{ filename: "" }]], ["getUploadStream", [{ filename: ".." }]]]) {
    const s = new Service({ logger }, configs.path_style)
    try {
      await s[method](...args)
      out.errors.push({ config: "path_style", method, args, result: "ok" })
    } catch (e) {
      out.errors.push({ config: "path_style", method, args, type: e.type, message: e.message })
    }
  }
  try {
    new Service({ logger }, { bucket: "b", region: "r", file_url: "u" })
  } catch (e) {
    out.errors.push({ config: "missing_keys", method: "constructor", type: e.type, message: e.message })
  }
  write("file-s3", out)
}

async function sendgrid() {
  const rec = await recorder((r, res, body) => {
    const parsed = body ? JSON.parse(body) : {}
    if (parsed.subject === "fail") {
      res.writeHead(400, { "content-type": "application/json" })
      res.end(JSON.stringify({ errors: [{ message: "The from address does not match a verified Sender Identity.", field: "from", help: null }] }))
      return
    }
    if (parsed.subject === "fail-empty") {
      res.writeHead(500, { "content-type": "application/json" })
      res.end("{}")
      return
    }
    res.writeHead(202)
    res.end()
  })
  const sgmail = req("@sendgrid/mail")
  const { SendgridNotificationService } = req("@medusajs/notification-sendgrid/dist/services/sendgrid")
  const s = new SendgridNotificationService({ logger }, { api_key: "SG.test-key", from: "store@example.test" })
  sgmail.client.setDefaultRequest("baseUrl", `http://127.0.0.1:${rec.port}`)
  const cases = [
    { name: "template", input: { to: "customer@example.test", channel: "email", template: "d-123", data: { order: { id: "order_1", total: 10.5 }, list: [1, "a"] } } },
    { name: "content", input: { to: "customer@example.test", channel: "email", template: "ignored", content: { subject: "Hello", html: "<p>Hi</p>", text: "Hi" }, data: { a: 1 } } },
    { name: "from_override", input: { to: "customer@example.test", from: "  other@example.test  ", channel: "email", template: "d-1" } },
    { name: "from_blank", input: { to: "customer@example.test", from: "   ", channel: "email", template: "d-1" } },
    { name: "attachments", input: { to: "customer@example.test", channel: "email", template: "d-2", attachments: [{ content: "aGVsbG8=", filename: "a.txt", content_type: "text/plain" }, { content: "Ymll", filename: "b.png", content_type: "image/png", disposition: "inline", id: "logo" }] } },
    { name: "personalizations", input: { to: "ignored@example.test", channel: "email", template: "d-3", data: { x: 1 }, provider_data: { personalizations: [{ to: [{ email: "p1@example.test" }], dynamic_template_data: { y: 2 } }] } } },
    { name: "personalizations_empty", input: { to: "customer@example.test", channel: "email", template: "d-3", provider_data: { personalizations: [] } } },
    { name: "empty_content", input: { to: "customer@example.test", channel: "email", template: "d-4", content: null } },
    { name: "no_data", input: { to: "customer@example.test", channel: "email", template: "d-5" } },
    { name: "error", input: { to: "customer@example.test", channel: "email", template: "t", content: { subject: "fail", html: "<p/>" } } },
    { name: "error_empty", input: { to: "customer@example.test", channel: "email", template: "t", content: { subject: "fail-empty", html: "<p/>" } } },
  ]
  const out = { options: { api_key: "SG.test-key", from: "store@example.test" }, cases: [] }
  for (const c of cases) {
    rec.requests.length = 0
    let result
    try {
      result = { result: await s.send(c.input) }
    } catch (e) {
      result = { error: { type: e.type, message: e.message } }
    }
    out.cases.push({
      ...c,
      ...result,
      requests: rec.requests.map((r) => ({
        method: r.method,
        url: r.url,
        headers: { authorization: r.headers.authorization, "content-type": r.headers["content-type"], accept: r.headers.accept },
        body: r.body ? JSON.parse(r.body) : null,
      })),
    })
  }
  try {
    await s.send(undefined)
  } catch (e) {
    out.cases.push({ name: "undefined", input: null, error: { type: e.type, message: e.message }, requests: [] })
  }
  rec.server.close()
  write("notification-sendgrid", out)
}

async function posthog() {
  const rec = await recorder((r, res) => {
    res.writeHead(200, { "content-type": "application/json" })
    res.end("{}")
  })
  const { PosthogAnalyticsService } = req("@medusajs/analytics-posthog/dist/services/posthog-analytics")
  const options = { posthogEventsKey: "phc_test", posthogHost: `http://127.0.0.1:${rec.port}` }
  const s = new PosthogAnalyticsService({ logger }, options)
  const calls = [
    ["track", { event: "order_placed", actor_id: "cus_1", properties: { total: 10, items: ["a"] } }],
    ["track", { event: "cart_created", actor_id: "cus_2", group: { type: "company", id: "comp_1" }, properties: { a: 1 } }],
    ["track", { event: "no_props", actor_id: "cus_3" }],
    ["track", { event: "", actor_id: "cus_1" }],
    ["track", { event: "x" }],
    ["track", { event: "x", actor_id: "a", group: { id: "g" } }],
    ["identify", { actor_id: "cus_1", properties: { email: "c@example.test", name: "C" } }],
    ["identify", { group: { type: "company", id: "comp_1" }, actor_id: "cus_1", properties: { plan: "pro" } }],
    ["identify", { group: { type: "company", id: "comp_2" } }],
    ["identify", { properties: {} }],
  ]
  const out = { options: { posthogEventsKey: "phc_test" }, calls: [] }
  for (const [method, data] of calls) {
    try {
      await s[method](data)
      out.calls.push({ method, data, result: "ok" })
    } catch (e) {
      out.calls.push({ method, data, error: e.message })
    }
  }
  await s.shutdown()
  out.requests = rec.requests.map((r) => ({
    method: r.method,
    url: r.url,
    headers: { "content-type": r.headers["content-type"], "content-encoding": r.headers["content-encoding"], "user-agent": r.headers["user-agent"] },
    body: r.body ? JSON.parse(r.body) : null,
  }))
  try {
    new PosthogAnalyticsService({ logger }, {})
  } catch (e) {
    out.constructor_error = e.message
  }
  rec.server.close()
  write("analytics-posthog", out)
}

async function main() {
  await fileS3()
  await sendgrid()
  await posthog()
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
