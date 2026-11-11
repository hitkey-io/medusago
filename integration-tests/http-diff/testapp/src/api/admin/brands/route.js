// GET /admin/brands: query.graph over brands; POST: createBrands.
exports.GET = async (req, res) => {
  const query = req.scope.resolve("query")
  const fields = typeof req.query.fields === "string" ? req.query.fields.split(",") : ["*"]
  const filters = req.query.filters ? JSON.parse(req.query.filters) : {}
  const { data, metadata } = await query.graph({
    entity: "brand",
    fields,
    filters,
    pagination: {
      skip: Number(req.query.offset ?? 0),
      take: Number(req.query.limit ?? 20),
      ...(req.query.order ? { order: JSON.parse(req.query.order) } : {}),
    },
  })
  res.json({ brands: data, count: metadata.count, offset: metadata.skip, limit: metadata.take })
}

exports.POST = async (req, res) => {
  const brandModule = req.scope.resolve("brand")
  const [brand] = await brandModule.createBrands([req.body])
  res.json({ brand })
}
