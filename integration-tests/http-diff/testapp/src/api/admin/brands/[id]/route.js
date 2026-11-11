// GET /admin/brands/:id (retrieveBrand with relations), POST (updateBrands),
// DELETE (softDeleteBrands, cascading to the logos and the profile).
exports.GET = async (req, res) => {
  const brandModule = req.scope.resolve("brand")
  const brand = await brandModule.retrieveBrand(req.params.id, { relations: ["logos", "profile"] })
  res.json({ brand })
}

exports.POST = async (req, res) => {
  const brandModule = req.scope.resolve("brand")
  const brand = await brandModule.updateBrands({ id: req.params.id, ...req.body })
  res.json({ brand })
}

exports.DELETE = async (req, res) => {
  const brandModule = req.scope.resolve("brand")
  const cascaded = await brandModule.softDeleteBrands([req.params.id], { returnLinkableKeys: true })
  res.json({ id: req.params.id, deleted: true, cascaded: cascaded ?? null })
}
