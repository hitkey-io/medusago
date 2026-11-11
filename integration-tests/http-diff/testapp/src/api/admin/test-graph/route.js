// GET /admin/test-graph?entity=&fields=a,b&filters={json}&order={json}&offset=&limit=
// query.graph over any entity (the brand test module, its links).
exports.GET = async (req, res) => {
  const query = req.scope.resolve("query")
  const fields = typeof req.query.fields === "string" ? req.query.fields.split(",") : ["*"]
  const filters = req.query.filters ? JSON.parse(req.query.filters) : {}
  const pagination = {}
  if (req.query.offset !== undefined) {
    pagination.skip = Number(req.query.offset)
  }
  if (req.query.limit !== undefined) {
    pagination.take = Number(req.query.limit)
  }
  if (req.query.order) {
    pagination.order = JSON.parse(req.query.order)
  }
  const { data, metadata } = await query.graph({
    entity: req.query.entity,
    fields,
    filters,
    pagination,
  })
  res.json({ data, metadata: metadata ?? null })
}
