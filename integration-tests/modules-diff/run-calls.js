// Runs module service calls against an upstream Medusa application and
// writes their results as JSON. Executed with `medusa exec` from the
// reference application:
//
//   medusa exec <this file> <cases.json> <out.json>
//
// cases.json: [{ "name", "module", "method", "args": [...] }]
// out.json:   { "<name>": { "result": ... } | { "error": { "type", "message" } } }
//
// Cases run in order on the same database. A string argument of the form
// "{{<case name>.<path>}}" is replaced with the value at that path of an
// earlier case output (e.g. "{{shirt.result.0.id}}").

const fs = require("fs")

function resolveRefs(value, out) {
  if (typeof value === "string") {
    const m = value.match(/^\{\{([^.}]+)\.([^}]+)\}\}$/)
    if (!m) {
      return value
    }
    return m[2].split(".").reduce((acc, key) => (acc == null ? acc : acc[key]), out[m[1]])
  }
  if (Array.isArray(value)) {
    return value.map((v) => resolveRefs(v, out))
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, resolveRefs(v, out)]))
  }
  return value
}

exports.default = async function runCalls({ container, args }) {
  const [casesFile, outFile] = args
  const cases = JSON.parse(fs.readFileSync(casesFile, "utf8"))
  const out = {}

  for (const c of cases) {
    const service = container.resolve(c.module)
    try {
      const result = await service[c.method](...resolveRefs(c.args ?? [], out))
      out[c.name] = { result: JSON.parse(JSON.stringify(result ?? null)) }
    } catch (e) {
      out[c.name] = { error: { type: e.type ?? e.name, message: e.message } }
    }
  }

  fs.writeFileSync(outFile, JSON.stringify(out, null, 2) + "\n")
}
