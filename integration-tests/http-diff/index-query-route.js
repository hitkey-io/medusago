// Test-only index query route: POST /index-query runs query.index(body)
// (Query), or the index module's query(body) with "mode": "module" (the
// documents read back from the index).
const { ContainerRegistrationKeys, Modules } = require("@medusajs/framework/utils")

exports.POST = async (req, res) => {
  const { mode, ...input } = req.body ?? {}
  if (mode === "module") {
    const result = await req.scope.resolve(Modules.INDEX).query(input)
    return res.json(result)
  }
  const result = await req.scope.resolve(ContainerRegistrationKeys.QUERY).index(input)
  res.json(result)
}
