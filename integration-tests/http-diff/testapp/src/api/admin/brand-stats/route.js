// GET /admin/brand-stats: listBrandStats; POST { stats: [...] }: upsert by
// the composite primary key (brand_id, period).
exports.GET = async (req, res) => {
  const brandModule = req.scope.resolve("brand")
  const stats = await brandModule.listBrandStats({}, { order: { brand_id: "ASC", period: "ASC" } })
  res.json({ stats })
}

exports.POST = async (req, res) => {
  const brandModule = req.scope.resolve("brand")
  const stats = await brandModule.upsertBrandStats(req.body.stats)
  res.json({ stats })
}
