// Runs module service calls against an upstream Medusa application and
// writes their results as JSON. Executed with `medusa exec` from the
// reference application:
//
//   medusa exec <this file> <cases.json> <out.json>
//
// cases.json: [{ "name", "module", "method", "args": [...] }]
// out.json:   { "<name>": { "result": ... } | { "error": { "type", "message" } } }

const fs = require("fs")

exports.default = async function runCalls({ container, args }) {
  const [casesFile, outFile] = args
  const cases = JSON.parse(fs.readFileSync(casesFile, "utf8"))
  const out = {}

  for (const c of cases) {
    const service = container.resolve(c.module)
    try {
      const result = await service[c.method](...(c.args ?? []))
      out[c.name] = { result: JSON.parse(JSON.stringify(result ?? null)) }
    } catch (e) {
      out[c.name] = { error: { type: e.type ?? e.name, message: e.message } }
    }
  }

  fs.writeFileSync(outFile, JSON.stringify(out, null, 2) + "\n")
}
